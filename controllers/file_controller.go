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

// 列出文件
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取目录"})
		return
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

// 上传
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

// 删除
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

// 创建文件夹
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

// 下载
func DownloadFile(c *gin.Context) {
	path := c.Query("path")
	if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "越权"})
		return
	}
	c.FileAttachment(path, filepath.Base(path))
}
