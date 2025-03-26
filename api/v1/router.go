package v1

import (
	"auto-deploy-platform/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// 基本功能
		v1.GET("/ping", controllers.Ping)
		v1.GET("/ansible/playbooks", controllers.ListPlaybooks)
		v1.POST("/run-ansible", controllers.RunAnsible) // ✅ Ansible
		v1.GET("/ws-system", controllers.SystemInfoWS)  // ✅ ws推送状态

		// 容器管理
		v1.GET("/containers", controllers.ListContainers)
		v1.POST("/container/start/:id", controllers.StartContainer)
		v1.POST("/container/stop/:id", controllers.StopContainer)
		v1.GET("/ws/container-logs/:id", controllers.ContainerLogsWS)
		v1.POST("/container/create", controllers.CreateContainer)

		// 🧩 Compose 管理
		v1.POST("/compose/upload", controllers.UploadCompose)
		v1.GET("/compose/list", controllers.ListCompose)
		v1.GET("/compose/status", controllers.ComposeStatus)
		v1.POST("/compose/up", controllers.StartCompose)
		v1.POST("/compose/down", controllers.StopCompose)
		v1.POST("/compose/delete", controllers.DeleteCompose)
		v1.GET("/ws/compose-logs", controllers.ComposeLogsWS)

		// 🧩 文件管理
		v1.GET("/files/config", controllers.GetFileConfig)
		v1.GET("/files/list", controllers.ListFiles)
		v1.POST("/files/upload", controllers.UploadFile)
		v1.POST("/files/delete", controllers.DeleteFile)
		v1.POST("/files/mkdir", controllers.Mkdir)
		v1.GET("/files/download", controllers.DownloadFile)
		v1.POST("/files/rename", controllers.RenameFile)
		v1.POST("/files/batch-delete", controllers.BatchDelete)
		v1.GET("/files/view", controllers.ViewFile)
		v1.POST("/files/save", controllers.SaveFile)
		v1.POST("/files/chmod", controllers.ChmodFile)
		v1.POST("/files/compress", controllers.CompressFiles)
		v1.POST("/files/extract", controllers.ExtractZip)
		v1.POST("/files/move", controllers.MoveFile)
		v1.POST("/files/batch-download", controllers.BatchDownload)
		v1.POST("/files/batch-chmod", controllers.BatchChmod)

		//for _, route := range r.Routes() {
		//	fmt.Printf("%s -> %s\n", route.Method, route.Path)
		//}

	}
}
