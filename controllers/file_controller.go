package controllers

import (
	"auto-deploy-platform/models"
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

// GetFileConfig 获取文件配置
// @Summary 获取文件管理配置
// @Description 返回默认基础目录和是否允许任意目录
// @Tags 文件管理
// @Produce json
// @Success 200 {object} models.FileConfigResponse
// @Router /api/v1/files/config [get]
func GetFileConfig(c *gin.Context) {
	c.JSON(http.StatusOK, models.FileConfigResponse{
		BaseDir:    baseDir,
		AllowAll:   allowAll,
		ApiBaseUrl: "/api/v1", // 这里设置
	})
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
	path := filepath.Clean(c.Query("path"))
	if path == "" {
		path = baseDir
	} else if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Code:    403,
			Message: "越权操作",
		})
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		// 尝试修复
		os.MkdirAll(path, 0755)
		files, err = os.ReadDir(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Code:    500,
				Message: "无法读取目录",
			})
			return
		}
	}

	var list []models.FileInfo
	for _, f := range files {
		fi, _ := f.Info()
		list = append(list, models.FileInfo{
			Name:    f.Name(),
			IsDir:   f.IsDir(),
			Mode:    fi.Mode().Perm().String(),
			Size:    fi.Size(),
			ModTime: fi.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, models.ListFilesResponse{
		Current: path,
		Files:   list,
	})
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
	path := filepath.Clean(c.PostForm("path"))
	if path == "" {
		path = baseDir
	} else if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Code:    403,
			Message: "越权",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    500,
			Message: "未选择文件",
		})
		return
	}

	saveDir := filepath.Join(path)
	os.MkdirAll(saveDir, 0755)
	savePath := filepath.Join(saveDir, file.Filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    500,
			Message: "上传失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Code:    200,
		Message: "上传成功",
	})
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
	var req models.FileDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    400,
			Message: "参数错误",
		})
		return
	}
	fullPath := filepath.Clean(filepath.Join(req.Path, req.Name))
	if !allowAll && !filepath.HasPrefix(fullPath, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Code:    403,
			Message: "越权",
		})
		return
	}
	if err := os.RemoveAll(fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    500,
			Message: "删除失败",
		})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Code:    200,
		Message: "删除成功",
	})
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
	var req models.MkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    400,
			Message: "参数错误",
		})
		return
	}
	fullPath := filepath.Clean(filepath.Join(req.Path, req.Name))
	if !allowAll && !filepath.HasPrefix(fullPath, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Code:    403,
			Message: "越权",
		})
		return
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    500,
			Message: "创建失败",
		})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Code:    200,
		Message: "创建成功",
	})
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
	path := filepath.Clean(c.Query("path"))
	if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Code:    403,
			Message: "越权",
		})
		return
	}
	c.FileAttachment(path, filepath.Base(path))
}

// RenameFile 文件/目录重命名
// @Summary 重命名文件或文件夹
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param rename body models.RenameRequest true "重命名参数"
// @Success 200 {object} models.SuccessResponse "重命名成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "失败"
// @Router /api/v1/files/rename [post]
func RenameFile(c *gin.Context) {
	var req models.RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.OldName == "" || req.NewName == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	oldPath := filepath.Join(req.Path, req.OldName)
	newPath := filepath.Join(req.Path, req.NewName)
	if !allowAll && !filepath.HasPrefix(oldPath, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Code: 403, Message: "越权"})
		return
	}
	err := os.Rename(oldPath, newPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Code: 500, Message: "重命名失败"})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Code: 200, Message: "重命名成功"})
}

// BatchDelete 批量删除
// @Summary 批量删除文件或文件夹
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param delete body models.BatchDeleteRequest true "批量删除参数"
// @Success 200 {object} models.SuccessResponse "删除成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "失败"
// @Router /api/v1/files/batch-delete [post]
func BatchDelete(c *gin.Context) {
	var req models.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Names) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	for _, name := range req.Names {
		fullPath := filepath.Join(req.Path, name)
		if !allowAll && !filepath.HasPrefix(fullPath, baseDir) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{Code: 403, Message: "越权"})
			return
		}
		os.RemoveAll(fullPath)
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Code: 200, Message: "批量删除成功"})
}

// ViewFile 查看文件内容
// @Summary 查看文本文件内容
// @Tags 文件管理
// @Produce json
// @Param path query string true "文件完整路径"
// @Success 200 {object} models.FileContentResponse "文件内容"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "读取失败"
// @Router /api/v1/files/view [get]
func ViewFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	if !allowAll && !filepath.HasPrefix(path, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Code: 403, Message: "越权"})
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Code: 500, Message: "读取失败"})
		return
	}
	c.JSON(http.StatusOK, models.FileContentResponse{Content: string(content)})
}

type SaveFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// SaveFile 保存文件内容
// @Summary 保存文件内容
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param save body models.SaveFileRequest true "保存内容参数"
// @Success 200 {object} models.SuccessResponse "保存成功"
// @Failure 400 {object} models.ErrorResponse "参数错误"
// @Failure 403 {object} models.ErrorResponse "越权"
// @Failure 500 {object} models.ErrorResponse "保存失败"
// @Router /api/v1/files/save [post]
func SaveFile(c *gin.Context) {
	var req models.SaveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	if !allowAll && !filepath.HasPrefix(req.Path, baseDir) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Code: 403, Message: "越权"})
		return
	}
	err := os.WriteFile(req.Path, []byte(req.Content), 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Code: 500, Message: "保存失败"})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Code: 200, Message: "保存成功"})
}
