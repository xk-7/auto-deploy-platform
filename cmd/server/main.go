package main

import (
	"auto-deploy-platform/api/v1"
	"auto-deploy-platform/config"
	"auto-deploy-platform/middlewares"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitConfig()

	r := gin.Default()

	// Middlewares
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.ErrorHandler())

	// 静态目录 ➡️ 访问 http://localhost:8081/static/filename
	r.Static("/static", "./static")

	// 设置首页 ➡️ 访问 http://localhost:8081/ 直接显示 index.html
	r.StaticFile("/", "./static/index.html")
	// Routes
	v1.RegisterRoutes(r)

	log.Println("✅ Server starting on :8081...")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
