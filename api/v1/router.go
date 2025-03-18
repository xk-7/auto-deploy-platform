package v1

import (
	"auto-deploy-platform/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", controllers.Ping)
		v1.POST("/run-ansible", controllers.RunAnsible) // ✅ Ansible 也顺带
		v1.GET("/ws-system", controllers.SystemInfoWS)  // ✅ ws推送状态
		v1.GET("/containers", controllers.ListContainers)

	}
}
