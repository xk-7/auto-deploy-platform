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

// UploadCompose 上传 Docker Compose 文件
// @Summary 上传 Compose 文件
// @Description 上传并保存 Docker Compose 文件
// @Tags Compose管理
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Compose 文件名称"
// @Param compose_file formData file true "Compose 文件 (YAML格式)"
// @Success 200 {object} models.SuccessResponse "上传成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /compose/upload [post]
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

// ListCompose 获取 Compose 应用列表
// @Summary 获取 Compose 应用列表
// @Description 列出当前存在的所有 Compose 应用
// @Tags Compose管理
// @Produce json
// @Success 200 {object} models.ListComposeResponse "成功返回 Compose 应用列表"
// @Failure 500 {object} models.ErrorResponse "读取目录失败"
// @Router /compose/list [get]
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

// ComposeStatus 获取 Compose 应用容器状态
// @Summary 获取 Compose 应用状态
// @Description 查看各 Compose 应用包含的容器及运行状态
// @Tags Compose管理
// @Produce json
// @Success 200 {object} models.ComposeStatusResponse "成功返回 Compose 容器状态"
// @Failure 500 {object} models.ErrorResponse "Docker client 初始化或容器列表失败"
// @Router /compose/status [get]
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

// StartCompose 启动 Compose 应用
// @Summary 启动 Compose 应用
// @Description 通过应用名称启动对应 Compose 应用
// @Tags Compose管理
// @Accept json
// @Produce json
// @Param compose body models.ComposeActionRequest true "Compose 应用名称"
// @Success 200 {object} models.SuccessResponse "启动成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 500 {object} models.ErrorResponse "启动失败"
// @Router /compose/start [post]
func StartCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = dir
	cmd.Run()
	c.JSON(http.StatusOK, gin.H{"message": "Started"})
}

// StopCompose 停止 Compose 应用
// @Summary 停止 Compose 应用
// @Description 通过应用名称停止对应 Compose 应用
// @Tags Compose管理
// @Accept json
// @Produce json
// @Param compose body models.ComposeActionRequest true "Compose 应用名称"
// @Success 200 {object} models.SuccessResponse "停止成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 500 {object} models.ErrorResponse "停止失败"
// @Router /compose/stop [post]
func StopCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = dir
	cmd.Run()
	c.JSON(http.StatusOK, gin.H{"message": "Stopped"})
}

// DeleteCompose 删除 Compose 应用
// @Summary 删除 Compose 应用
// @Description 删除指定 Compose 应用及其目录
// @Tags Compose管理
// @Accept json
// @Produce json
// @Param compose body models.ComposeActionRequest true "Compose 应用名称"
// @Success 200 {object} models.SuccessResponse "删除成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 500 {object} models.ErrorResponse "删除失败"
// @Router /compose/delete [post]
func DeleteCompose(c *gin.Context) {
	var req struct{ Name string }
	c.BindJSON(&req)
	dir := filepath.Join(composeBasePath, req.Name)
	os.RemoveAll(dir)
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// ComposeLogsWS 获取 Compose 应用日志 WebSocket
// @Summary 获取 Compose 应用日志
// @Description 通过 WebSocket 连接实时获取指定 Compose 应用的日志流
// @Tags Compose管理
// @Produce plain
// @Param name query string true "Compose 应用名称"
// @Success 101 {string} string "WebSocket 连接已建立，开始推送日志"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 500 {object} models.ErrorResponse "WebSocket 升级失败 或 日志启动失败"
// @Router /compose/logs/ws [get]
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
