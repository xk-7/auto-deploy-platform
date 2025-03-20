package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var composeBasePath = "./compose-files" // 📁 Compose 文件存储目录

// ---------------- 上传 Compose ----------------
func UploadCompose(c *gin.Context) {
	name := c.PostForm("name")
	file, err := c.FormFile("compose_file")
	if err != nil || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	saveDir := filepath.Join(composeBasePath, name)
	os.MkdirAll(saveDir, 0755)
	savePath := filepath.Join(saveDir, "docker-compose.yml")
	c.SaveUploadedFile(file, savePath)

	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}

// ---------------- Compose 列表 ----------------
func ListCompose(c *gin.Context) {
	entries, err := os.ReadDir(composeBasePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
		return
	}
	var apps []string
	for _, e := range entries {
		if e.IsDir() {
			apps = append(apps, e.Name())
		}
	}
	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

// ---------------- Compose 状态 ----------------
func ComposeStatus(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client failed"})
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "List containers failed"})
		return
	}

	composeApps := make(map[string][]gin.H)
	for _, container := range containers {
		project := container.Labels["com.docker.compose.project"]
		if project == "" {
			continue
		}
		portStr := ""
		for _, p := range container.Ports {
			portStr += fmt.Sprintf("%d:%d ", p.PublicPort, p.PrivatePort)
		}
		composeApps[project] = append(composeApps[project], gin.H{
			"id":     container.ID[:12],
			"name":   container.Names[0],
			"image":  container.Image,
			"status": container.Status,
			"ports":  portStr,
		})
	}

	// 获取所有 compose-files 目录
	entries, _ := os.ReadDir(composeBasePath)
	var result []gin.H
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		project := e.Name()
		containers := composeApps[project]
		status := fmt.Sprintf("Running (%d/%d)", len(containers), len(containers))
		if len(containers) == 0 {
			status = "Not Running"
		}
		result = append(result, gin.H{
			"name":       project,
			"status":     status,
			"containers": containers,
		})
	}

	c.JSON(http.StatusOK, gin.H{"apps": result})
}

// ---------------- Start ----------------
func StartCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = dir
	cmd.Run()
	c.JSON(http.StatusOK, gin.H{"message": "Started"})
}

// ---------------- Stop ----------------
func StopCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = dir
	cmd.Run()
	c.JSON(http.StatusOK, gin.H{"message": "Stopped"})
}

// ---------------- Delete ----------------
func DeleteCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	os.RemoveAll(dir)
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// ---------------- 日志 WebSocket ----------------
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func ComposeLogsWS(c *gin.Context) {
	name := c.Query("name")
	dir := filepath.Join(composeBasePath, name)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	cmd := exec.Command("docker-compose", "logs", "-f")
	cmd.Dir = dir
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			conn.WriteMessage(websocket.TextMessage, buf[:n])
		}
		if err != nil {
			break
		}
	}
}
