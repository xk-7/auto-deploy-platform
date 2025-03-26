package controllers

import (
	"auto-deploy-platform/config"
	_ "auto-deploy-platform/config"
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ListPlaybooks 获取可用的 Ansible Playbook 文件列表
// @Summary 获取 Playbook 列表
// @Description 返回指定目录下所有允许的 .yml/.yaml 类型的 Ansible Playbook 文件名列表
// @Tags 自动化部署
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]string "playbooks 文件列表"
// @Failure 500 {object} models.ErrorResponse "目录不存在或读取失败"
// @Router /ansible/playbooks [get]
func ListPlaybooks(c *gin.Context) {
	playbookDir := config.Conf.Ansible.PlaybookDir
	playbooks := []string{} // 👈 初始化为空数组，避免 null

	if _, err := os.Stat(playbookDir); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Playbook 目录不存在: " + playbookDir})
		return
	}

	err := filepath.Walk(playbookDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isAllowedExtension(info.Name()) {
			playbooks = append(playbooks, info.Name())
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取 Playbook 目录: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playbooks": playbooks})
}

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
	if err := c.ShouldBindJSON(&req); err != nil || req.Inventory == "" || req.Playbook == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Inventory 和 Playbook 不能为空"})
		return
	}

	// 🔸 计算 Playbook 和 Inventory 绝对路径
	playbookPath := filepath.Join(config.Conf.Ansible.PlaybookDir, req.Playbook)
	inventoryPath := filepath.Join(config.Conf.Ansible.InventoryDir, req.Inventory)

	// 🔸 安全检查：防止路径遍历攻击
	if strings.Contains(req.Playbook, "..") || strings.Contains(req.Inventory, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
		return
	}

	// 🔸 确保文件存在
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Playbook 文件不存在: %s", req.Playbook)})
		return
	}
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Inventory 文件不存在: %s", req.Inventory)})
		return
	}

	// 🔸 构建 Ansible 命令
	cmd := exec.Command("ansible-playbook", "-i", inventoryPath, playbookPath)

	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 Ansible 输出失败"})
		return
	}

	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动 Ansible 失败"})
		return
	}

	// 🔹 实时返回日志流
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			cmd.Process.Kill()
			return false
		default:
			if stdoutScanner.Scan() {
				c.Writer.Write([]byte(stdoutScanner.Text() + "\n"))
				c.Writer.Flush()
				return true
			}
			if stderrScanner.Scan() {
				c.Writer.Write([]byte(stderrScanner.Text() + "\n"))
				c.Writer.Flush()
				return true
			}
			return false
		}
	})

	cmd.Wait()
}

// 🔹 只允许 .yml 和 .yaml 后缀的文件
func isAllowedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range config.Conf.Ansible.AllowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

// 5️⃣ 等待子进程退出
//	err = cmd.Wait()
//	if err != nil {
//		fmt.Printf("Ansible playbook execution failed: %v\n", err)
//	}
//}
