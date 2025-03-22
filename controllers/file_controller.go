package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
)

var (
	baseDir  = "/tmp/file-manager"
	allowAll = true
)

// 初始化配置
func InitFileConfig(dir string, allow bool) {
	baseDir = dir
	allowAll = allow
	os.MkdirAll(baseDir, 0755)
}

// ListFiles 列出服务器目录文件
// @Summary 获取文件列表
// @Description 根据指定路径列出文件和目录，路径为空默认列出基础路径
// @Tags 文件管理
// @Produce json
// @Param path query string false "目标路径，默认基础目录"
// @Success 200 {object} models.ListFilesResponse "成功返回文件列表"
// @Failure 403 {object} models.ErrorResponse "越权操作"
// @Failure 500 {object} models.ErrorResponse "读取目录失败"
func ListFiles(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = baseDir
	} else if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权操作"})
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		// 尝试修复
		os.MkdirAll(path, 0755)
		files, err = os.ReadDir(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录"})
			return
		}
	}

	var list []gin.H
	for _, f := range files {
		fi, _ := f.Info()
		list = append(list, gin.H{
			"name":     f.Name(),
			"is_dir":   f.IsDir(),
			"mode":     fi.Mode().Perm().String(),
			"size":     fi.Size(),
			"mod_time": fi.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"files": list, "current": path})
}

// UploadFile 上传文件
// @Summary 上传文件
// @Description 上传文件到指定路径
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Param path formData string false "目标路径，默认基础目录"
// @Param file formData file true "上传文件"
// @Success 200 {object} models.SuccessResponse "上传成功"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "上传失败"
// @Router /files/upload [post]
func UploadFile(c *gin.Context) {
	path := c.PostForm("path")
	if path == "" {
		path = baseDir
	} else if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权"})
		return
	}
	file, _ := c.FormFile("file")
	saveDir := filepath.Join(path)
	os.MkdirAll(saveDir, 0755)
	savePath := filepath.Join(saveDir, file.Filename)
	c.SaveUploadedFile(file, savePath)
	c.JSON(http.StatusOK, gin.H{"message": "上传成功"})
}

// DeleteFile 删除文件/文件夹
// @Summary 删除文件或文件夹
// @Description 删除指定路径下的文件或目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param file body models.FileDeleteRequest true "删除文件参数"
// @Success 200 {object} models.SuccessResponse "删除成功"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "删除失败"
// @Router /files/delete [post]
func DeleteFile(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	c.BindJSON(&req)
	fullPath := filepath.Join(req.Path, req.Name)
	if !allowAll && !filepath.HasPrefix(fullPath, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权"})
		return
	}
	os.RemoveAll(fullPath)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Mkdir 创建文件夹
// @Summary 创建文件夹
// @Description 在指定路径下创建新目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param dir body models.MkdirRequest true "创建目录参数"
// @Success 200 {object} models.SuccessResponse "创建成功"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "创建失败"
// @Router /files/mkdir [post]
func Mkdir(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	c.BindJSON(&req)
	fullPath := filepath.Join(req.Path, req.Name)
	if !allowAll && !filepath.HasPrefix(fullPath, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权"})
		return
	}
	os.MkdirAll(fullPath, 0755)
	c.JSON(http.StatusOK, gin.H{"message": "创建成功"})
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 下载指定路径文件
// @Tags 文件管理
// @Produce application/octet-stream
// @Param path query string true "文件路径"
// @Success 200 {file} file "文件流"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "下载失败"
// @Router /files/download [get]
func DownloadFile(c *gin.Context) {
	path := c.Query("path")
	if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权"})
		return
	}
	c.FileAttachment(path, filepath.Base(path))
}
