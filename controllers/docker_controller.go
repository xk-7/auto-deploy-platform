package controllers

import (
	_ "auto-deploy-platform/models"
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

// ListContainers 列出所有 Docker 容器
// @Summary 获取容器列表
// @Description 获取本机所有 Docker 容器的详细信息
// @Tags 容器管理
// @Produce json
// @Success 200 {object} models.ContainerListResponse "成功返回容器列表"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /containers [get]
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

// StartContainer 启动容器
// @Summary 启动容器
// @Description 启动指定容器
// @Tags 容器管理
// @Param id path string true "容器ID"
// @Success 200 {object} models.SuccessResponse "成功启动容器"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /container/{id}/start [post]
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

// StopContainer 停止指定容器
// @Summary 停止容器
// @Description 通过容器ID停止正在运行的容器
// @Tags 容器管理
// @Param id path string true "容器ID"
// @Success 200 {object} models.SuccessResponse "成功停止容器"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /container/{id}/stop [post]
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

// CreateContainer 创建 Docker 容器
// @Summary 创建容器
// @Description 创建一个新的 Docker 容器，支持设置端口映射、卷挂载、环境变量、资源限制等
// @Tags 容器管理
// @Accept json
// @Produce json
// @Param container body models.CreateContainerRequest true "创建容器请求参数"
// @Success 200 {object} models.CreateContainerResponse "创建成功返回容器ID"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /container/create [post]
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

// 辅助函数 → Base64 Encode
func encodeAuthToBase64(authConfig types.AuthConfig) string {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Printf("Error encoding auth: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ContainerLogsWS 容器日志 WebSocket
// @Summary 实时获取容器日志
// @Description 通过 WebSocket 连接获取指定容器的实时日志流，连接后服务器持续推送日志内容
// @Tags 容器管理
// @Param id path string true "容器ID"
// @Produce plain
// @Success 101 {string} string "WebSocket 连接已建立，开始推送日志"
// @Failure 500 {object} models.ErrorResponse "Docker client 初始化失败 或 WebSocket 升级失败"
// @Router /container/{id}/logs/ws [get]
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
