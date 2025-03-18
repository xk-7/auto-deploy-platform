package controllers

import (
	"github.com/gin-gonic/gin"
	"os/exec"
	"net/http"
	"fmt"
	"io"
)

func RunAnsible(c *gin.Context) {
	var req struct {
		Inventory string `json:"inventory"`
		Playbook  string `json:"playbook"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	cmd := exec.Command("ansible-playbook", "-i", req.Inventory, req.Playbook)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start: %v", err)})
		return
	}

	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 1024)
		n, _ := stdout.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			c.Writer.Flush()
			return true
		}
		n, _ = stderr.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			c.Writer.Flush()
			return true
		}
		return false
	})

	cmd.Wait()
}
