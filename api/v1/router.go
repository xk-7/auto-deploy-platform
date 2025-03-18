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
		v1.POST("/run-ansible", controllers.RunAnsible) // ✅ Ansible
		v1.GET("/ws-system", controllers.SystemInfoWS)  // ✅ ws推送状态

		// 容器管理
		v1.GET("/containers", controllers.ListContainers)
		v1.POST("/container/start/:id", controllers.StartContainer)
		v1.POST("/container/stop/:id", controllers.StopContainer)
		v1.GET("/ws/container-logs/:id", controllers.ContainerLogsWS)
	}
}
