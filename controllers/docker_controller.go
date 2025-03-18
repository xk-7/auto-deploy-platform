package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ------------------- 容器列表 -------------------
func ListContainers(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "List containers failed"})
		return
	}

	var containerInfos []gin.H
	for _, container := range containers {
		name := ""
		if len(container.Names) > 0 {
			name = container.Names[0]
		}
		containerInfos = append(containerInfos, gin.H{
			"id":      container.ID[:12],
			"name":    name,
			"status":  container.Status,
			"image":   container.Image,
			"created": container.Created,
		})
	}

	c.JSON(http.StatusOK, gin.H{"containers": containerInfos})
}

// ------------------- Start 容器 -------------------
func StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	if err := cli.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start container"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container started"})
}

// ------------------- Stop 容器 -------------------
func StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	timeout := 10 * time.Second
	if err := cli.ContainerStop(context.Background(), containerID, &timeout); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop container"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container stopped"})
}

// ------------------- 日志 WebSocket -------------------

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ContainerLogsWS(c *gin.Context) {
	containerID := c.Param("id")
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 1️⃣ 获取容器信息，确认 TTY 是否开启
	inspect, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error inspecting container"))
		return
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
	}

	logsReader, err := cli.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error fetching logs"))
		return
	}
	defer logsReader.Close()

	// 2️⃣ 处理日志数据
	if inspect.Config.Tty {
		// TTY 模式，直接复制
		buf := make([]byte, 1024)
		for {
			n, err := logsReader.Read(buf)
			if n > 0 {
				conn.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				break
			}
		}
	} else {
		// 非 TTY，解复用处理
		stdout := websocketWriter{Conn: conn}
		stderr := websocketWriter{Conn: conn}
		_, err := stdcopy.StdCopy(stdout, stderr, logsReader)
		if err != nil {
			log.Printf("StdCopy error: %v", err)
		}
	}
}

// ------------------- WebSocket Writer Helper -------------------

type websocketWriter struct {
	Conn *websocket.Conn
}

func (w websocketWriter) Write(p []byte) (int, error) {
	err := w.Conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}
