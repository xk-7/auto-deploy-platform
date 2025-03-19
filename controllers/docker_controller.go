package controllers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ---------------- WebSocket Upgrader -------------------
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ---------------- 容器列表 -------------------
func ListContainers(c *gin.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Println("Docker client init failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Println("List containers failed:", err)
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

// ---------------- Start 容器 -------------------
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

// ---------------- Stop 容器 -------------------
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

// ---------------- 容器日志 WebSocket -------------------
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

	if inspect.Config.Tty {
		// 容器启用 TTY
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
		// 非 TTY，需 demux 解码
		stdout := websocketWriter{Conn: conn}
		stderr := websocketWriter{Conn: conn}
		_, err := stdcopy.StdCopy(stdout, stderr, logsReader)
		if err != nil {
			log.Printf("StdCopy error: %v", err)
		}
	}
}

// ---------------- WebSocket Writer 帮助 -------------------
type websocketWriter struct {
	Conn *websocket.Conn
}

func (w websocketWriter) Write(p []byte) (int, error) {
	err := w.Conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

// ---------------- 容器 CPU/内存监控 WebSocket -------------------
func ContainerStatsWS(c *gin.Context) {
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

	stats, err := cli.ContainerStats(context.Background(), containerID, true)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error fetching stats"))
		return
	}
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)
	for {
		var v *types.StatsJSON
		if err := decoder.Decode(&v); err != nil {
			break
		}

		cpuPercent := calculateCPUPercentUnix(v)
		memUsage := v.MemoryStats.Usage
		memLimit := v.MemoryStats.Limit

		data := gin.H{
			"cpu_percent":  cpuPercent,
			"memory_usage": memUsage / (1024 * 1024),
			"memory_limit": memLimit / (1024 * 1024),
		}

		msg, _ := json.Marshal(data)
		conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(1 * time.Second)
	}
}

func calculateCPUPercentUnix(v *types.StatsJSON) float64 {
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

// ---------------- 容器创建 API -------------------
type CreateContainerRequest struct {
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	Ports         []string `json:"ports"`          // ["8080:80"]
	Env           []string `json:"env"`            // ["ENV=prod"]
	Volumes       []string `json:"volumes"`        // ["/host/path:/container/path"]
	CPUQuota      int64    `json:"cpu_quota"`      // e.g. 50000 = 50% CPU
	MemoryMB      int64    `json:"memory_mb"`      // 单位 MB
	RestartPolicy string   `json:"restart_policy"` // always, no, unless-stopped
	NetworkMode   string   `json:"network_mode"`   // bridge, host, none
}

func CreateContainer(c *gin.Context) {
	var req CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image is required"})
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	// --- 1️⃣ 自动补全 tag ---
	if !strings.Contains(req.Image, ":") {
		req.Image = req.Image + ":latest"
	}

	// --- 2️⃣ 自动生成容器名 ---
	containerName := req.Name
	if containerName == "" {
		containerName = "auto-" + time.Now().Format("20060102150405")
	}

	// --- 3️⃣ 端口映射 ---
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, p := range req.Ports {
		if p == "" {
			continue
		}
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			continue
		}
		hostPort := parts[0]
		containerPort := parts[1]
		port, _ := nat.NewPort("tcp", containerPort)
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	// --- 4️⃣ Volume 挂载 ---
	binds := []string{}
	for _, v := range req.Volumes {
		if v != "" && strings.Contains(v, ":") {
			binds = append(binds, v)
		}
	}

	// --- 5️⃣ 环境变量 ---
	envList := []string{}
	for _, e := range req.Env {
		if e != "" {
			envList = append(envList, e)
		}
	}

	// --- 6️⃣ 镜像检查 & 拉取 ---
	_, _, imgErr := cli.ImageInspectWithRaw(context.Background(), req.Image)
	if imgErr != nil {
		log.Printf("Image not found locally, pulling: %s", req.Image)

		out, err := cli.ImagePull(context.Background(), req.Image, types.ImagePullOptions{
			All: false,
		})
		if err != nil {
			log.Printf("ImagePull failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pull image", "detail": err.Error()})
			return
		}
		defer out.Close()

		io.Copy(io.Discard, out) // 等待拉取完成
		log.Printf("Image pulled successfully: %s", req.Image)
	}

	// --- 7️⃣ HostConfig ---
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
	}

	// CPU 限制
	if req.CPUQuota > 0 {
		hostConfig.CPUQuota = req.CPUQuota
	}

	// 内存限制
	if req.MemoryMB > 0 {
		hostConfig.Memory = req.MemoryMB * 1024 * 1024
	}

	// 重启策略
	if req.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: req.RestartPolicy}
	}

	// 网络模式
	if req.NetworkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(req.NetworkMode)
	}

	// --- 8️⃣ 创建容器 ---
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image:        req.Image,
		Env:          envList,
		ExposedPorts: exposedPorts,
	}, hostConfig, &network.NetworkingConfig{}, nil, containerName)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create failed", "detail": err.Error()})
		return
	}

	// --- 9️⃣ 启动容器 ---
	err = cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start container", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container created", "id": resp.ID[:12]})
}
