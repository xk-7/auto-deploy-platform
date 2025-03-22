package main

import (
	"auto-deploy-platform/api/v1"
	"auto-deploy-platform/config"
	_ "auto-deploy-platform/docs"
	"auto-deploy-platform/middlewares"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

// @title 服务器平台 API 文档
// @version 1.0
// @description 文件上传、部署、Compose、文件管理接口文档
// @contact.name AutoDeploy Team
// @contact.email kliu4403@gmail.com
// @host localhost:8081
// @BasePath /
func main() {
	config.InitConfig()

	r := gin.Default()
	// Redoc 页面
	r.Static("/docs", "./static/redoc")

	//swagger页面
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Middlewares
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.ErrorHandler())

	// 静态目录 ➡️ 访问 http://localhost:8081/static/filename
	r.Static("/static", "./static")

	// ✅ Compose 文件目录映射
	r.Static("/compose-files", "./compose-files") // ⭐️ 挂载 Compose 文件目录

	// 设置首页 ➡️ 访问 http://localhost:8081/ 直接显示 index.html
	//r.StaticFile("/", "./static/index.html")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})

	// Routes
	v1.RegisterRoutes(r)

	log.Println("✅ Server starting on :8081...")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
