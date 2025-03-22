package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 辅助函数 → Base64 Encode
func encodeAuthToBase64(authConfig types.AuthConfig) string {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Printf("Error encoding auth: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}

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

// ------------------- 创建容器 -------------------
func CreateContainer(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Image   string `json:"image"`
		Ports   string `json:"ports"`
		Volumes string `json:"volumes"`
		Envs    string `json:"envs"`
		CPU     string `json:"cpu"`
		Memory  string `json:"memory"`
		Restart string `json:"restart"`
		Network string `json:"network"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Image == "" {
		log.Printf("❌ Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("❌ Docker client init failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker client init failed"})
		return
	}

	ctx := context.Background()

	// 标准化 image 名
	if !strings.Contains(req.Image, ":") {
		req.Image = req.Image + ":latest"
	}

	// 先 inspect
	_, _, err = cli.ImageInspectWithRaw(ctx, req.Image)
	if err != nil {
		log.Printf("镜像不存在，本地拉取: %s", req.Image)

		out, pullErr := cli.ImagePull(ctx, req.Image, types.ImagePullOptions{
			RegistryAuth: encodeAuthToBase64(types.AuthConfig{}),
		})
		if pullErr != nil {
			log.Printf("❌ Image pull failed: %v", pullErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Pull image failed", "detail": pullErr.Error()})
			return
		}

		defer out.Close()

		// 读取拉取流，确保拉取完成
		io.Copy(io.Discard, out)
		log.Println("镜像拉取完成！")
	}

	// 端口映射
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	if req.Network != "host" && req.Ports != "" {
		portPairs := strings.Split(req.Ports, ",")
		for _, p := range portPairs {
			parts := strings.Split(p, ":")
			if len(parts) != 2 {
				continue
			}
			portKey := nat.Port(parts[1] + "/tcp")
			exposedPorts[portKey] = struct{}{}
			portBindings[portKey] = []nat.PortBinding{{HostPort: parts[0]}}
		}
	}

	// 目录挂载
	var binds []string
	if req.Volumes != "" {
		volList := strings.Split(req.Volumes, ",")
		for _, v := range volList {
			binds = append(binds, v)
		}
	}

	// 环境变量
	var envList []string
	if req.Envs != "" {
		envList = strings.Split(req.Envs, ",")
	}

	// 资源限制
	hostConfig := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
	}
	if req.CPU != "" {
		hostConfig.NanoCPUs = parseCPU(req.CPU)
	}
	if req.Memory != "" {
		memLimit := parseMemory(req.Memory)
		hostConfig.Memory = memLimit
	}
	if req.Restart != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: req.Restart}
	}
	if req.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(req.Network)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        req.Image,
		Env:          envList,
		ExposedPorts: exposedPorts,
	}, hostConfig, &network.NetworkingConfig{}, nil, req.Name)
	if err != nil {
		log.Printf("❌ Container create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create failed", "detail": err.Error()})
		return
	}

	cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	c.JSON(http.StatusOK, gin.H{"message": "Container created", "id": resp.ID[:12]})
}

// 辅助函数 解析 CPU
func parseCPU(cpu string) int64 {
	val, err := parseFloat(cpu)
	if err != nil {
		return 0
	}
	return int64(val * 1e9) // CPU 核心数 → NanoCPU
}

// 辅助函数 解析内存
func parseMemory(mem string) int64 {
	val, err := parseFloat(mem)
	if err != nil {
		return 0
	}
	return int64(val * 1024 * 1024) // MB → Bytes
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
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
