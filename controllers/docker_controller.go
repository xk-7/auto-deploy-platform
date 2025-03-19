package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 获取 Docker client（动态 host）
func getDockerClient(c *gin.Context) (*client.Client, error) {
	hostIP := c.Query("host")

	var cli *client.Client
	var err error

	if hostIP == "" {
		cli, err = client.NewClientWithOpts(client.FromEnv) // 默认本机
	} else {
		dockerURL := fmt.Sprintf("tcp://%s:2375", hostIP)
		cli, err = client.NewClientWithOpts(
			client.WithHost(dockerURL),
			client.WithAPIVersionNegotiation(),
		)
	}
	return cli, err
}

// ------------------- 容器列表 -------------------
func ListContainers(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "List containers failed"})
		return
	}

	var containerInfos []gin.H
	for _, container := range containers {
		containerInfos = append(containerInfos, gin.H{
			"id":      container.ID[:12],
			"name":    container.Names[0],
			"status":  container.Status,
			"image":   container.Image,
			"created": container.Created,
		})
	}

	c.JSON(http.StatusOK, gin.H{"containers": containerInfos})
}

// ------------------- Start 容器 -------------------
func StartContainer(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containerID := c.Param("id")

	if err := cli.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start container"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container started"})
}

// ------------------- Stop 容器 -------------------
func StopContainer(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containerID := c.Param("id")
	timeout := 10 * time.Second
	if err := cli.ContainerStop(context.Background(), containerID, &timeout); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop container"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container stopped"})
}

// ------------------- 容器 Stats 实时推送 -------------------
func ContainerStatsWS(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containerID := c.Param("id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()
	stats, err := cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error fetching stats"))
		return
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)

	for {
		var v *types.StatsJSON
		if err := decoder.Decode(&v); err != nil {
			log.Printf("Decode error: %v", err)
			break
		}

		// 计算 CPU 使用率
		cpuPercent := calculateCPUPercent(v)

		// 计算内存使用
		memUsageMB := v.MemoryStats.Usage / (1024 * 1024)
		memLimitMB := v.MemoryStats.Limit / (1024 * 1024)

		info := gin.H{
			"cpu_percent":     cpuPercent,
			"memory_usage_mb": memUsageMB,
			"memory_limit_mb": memLimitMB,
		}

		jsonData, _ := json.Marshal(info)
		conn.WriteMessage(websocket.TextMessage, jsonData)
		time.Sleep(2 * time.Second)
	}
}

// CPU 计算
func calculateCPUPercent(stat *types.StatsJSON) float64 {
	cpuDelta := float64(stat.CPUStats.CPUUsage.TotalUsage) - float64(stat.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stat.CPUStats.SystemUsage) - float64(stat.PreCPUStats.SystemUsage)
	var cpuPercent = 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(stat.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

// ------------------- 创建容器 -------------------
type CreateContainerRequest struct {
	Image         string   `json:"image"`
	Name          string   `json:"name"`
	Ports         []string `json:"ports"`
	Env           []string `json:"env"`
	Volumes       []string `json:"volumes"`
	CPUQuota      int64    `json:"cpu_quota"`
	MemoryMB      int64    `json:"memory_mb"`
	RestartPolicy string   `json:"restart_policy"`
	NetworkMode   string   `json:"network_mode"`
}

func CreateContainer(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	portBindings := map[nat.Port][]nat.PortBinding{}
	exposedPorts := nat.PortSet{}
	for _, p := range req.Ports {
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			continue
		}
		hostPort := parts[0]
		containerPort := parts[1]
		port, _ := nat.NewPort("tcp", containerPort)
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{{HostPort: hostPort}}
	}

	binds := req.Volumes

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	}

	if req.CPUQuota > 0 {
		hostConfig.Resources.CPUQuota = req.CPUQuota
	}
	if req.MemoryMB > 0 {
		hostConfig.Resources.Memory = req.MemoryMB * 1024 * 1024
	}
	if req.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: req.RestartPolicy}
	}
	if req.NetworkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(req.NetworkMode)
	}

	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image:        req.Image,
		Env:          req.Env,
		ExposedPorts: exposedPorts,
	}, hostConfig, &network.NetworkingConfig{}, nil, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create failed", "detail": err.Error()})
		return
	}

	cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	c.JSON(http.StatusOK, gin.H{"message": "Container created", "id": resp.ID[:12]})
}

// ------------------- 日志 WebSocket -------------------
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ContainerLogsWS(c *gin.Context) {
	cli, err := getDockerClient(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	containerID := c.Param("id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "50"}
	out, err := cli.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error fetching logs"))
		return
	}
	defer out.Close()

	buf := make([]byte, 1024)
	for {
		n, err := out.Read(buf)
		if n > 0 {
			conn.WriteMessage(websocket.TextMessage, buf[:n])
		}
		if err != nil {
			break
		}
	}
}
