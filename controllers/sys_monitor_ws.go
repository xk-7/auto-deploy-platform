package controllers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var sysUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type SystemInfo struct {
	Uptime      string  `json:"uptime"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
}

func SystemInfoWS(c *gin.Context) {
	conn, err := sysUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("✅ System Info WebSocket client connected")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		info := collectSystemInfo()

		data, _ := json.Marshal(info)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket send error: %v", err)
			break
		}
	}
}

func collectSystemInfo() SystemInfo {
	// Uptime
	var uptimeBuf bytes.Buffer
	cmd := exec.Command("sh", "-c", "uptime")
	cmd.Stdout = &uptimeBuf
	cmd.Run()
	uptime := uptimeBuf.String()

	// CPU
	cpuPercent, _ := cpu.Percent(0, false)
	cpuUsage := cpuPercent[0]

	// Memory
	vmStat, _ := mem.VirtualMemory()

	// Disk
	diskStat, _ := disk.Usage("/")

	return SystemInfo{
		Uptime:      uptime,
		CPUUsage:    cpuUsage,
		MemoryUsage: vmStat.Used / (1024 * 1024), // MB
		MemoryTotal: vmStat.Total / (1024 * 1024),
		DiskUsed:    diskStat.Used / (1024 * 1024),
		DiskTotal:   diskStat.Total / (1024 * 1024),
	}
}
