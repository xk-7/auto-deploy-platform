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

// @title Auto Deploy Platform API
// @version 1.0
// @description 自动化部署平台接口文档，支持容器、Compose、文件、Ansible管理。
// @contact.name Dev Team
// @contact.url https://www.xkkk.online
// @contact.email kliu4403@gmail.com
// @host api.xkkk.online
// @BasePath /
// @schemes https
// @securityDefinitions.apikey BearerToken
// @in header
// @name Authorization
// @description 请输入 Bearer Token 认证，比如: Bearer eyJhbGciOi...

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
