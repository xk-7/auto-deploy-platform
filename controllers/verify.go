package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
}

func VerifyTurnstile(c *gin.Context) {
	token := c.PostForm("cf_token")
	if token == "" {
		c.String(http.StatusBadRequest, "验证失败: 未收到 token")
		return
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		url.Values{
			"secret":   {"0x4AAAAAABCmHplkkRZJaLONrDmuguawo0U"},
			"response": {token},
			"remoteip": {c.ClientIP()},
		})
	if err != nil {
		c.String(http.StatusInternalServerError, "验证请求失败")
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Println("Cloudflare 返回：", string(bodyBytes))

	var result TurnstileResponse
	json.Unmarshal(bodyBytes, &result)

	if !result.Success {
		c.String(http.StatusForbidden, "验证失败: "+strings.Join(result.ErrorCodes, ", "))
		return
	}
	// ✅ 设置 Cookie，1 小时内有效
	c.SetCookie("validated", "true", 3600, "/", "", false, true)
	c.Redirect(http.StatusFound, "/static/dashboard.html")
}
