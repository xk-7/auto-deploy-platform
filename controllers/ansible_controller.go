package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os/exec"
)

// RunAnsible 运行 Ansible Playbook
// @Summary 执行 Ansible Playbook
// @Description 通过提供 Inventory 文件和 Playbook 路径，执行 Ansible 并实时返回日志流
// @Tags 自动化部署
// @Accept json
// @Produce plain
// @Param ansible body models.RunAnsibleRequest true "Ansible 执行参数"
// @Success 200 {string} string "执行日志流（stream）"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "Ansible 执行失败"
// @Router /ansible/run [post]
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
