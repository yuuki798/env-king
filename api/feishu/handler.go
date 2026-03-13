package feishu

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"yuuki798/env-king/biz/feishu"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/feishu")
	g.GET("/status", getStatus)
	g.POST("/enable", enableFeishu)
	g.POST("/disable", disableFeishu)
}

func getStatus(c *gin.Context) {
	conn := feishu.GetConnector()
	if conn == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"enabled":    false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"enabled":    conn.IsEnabled(),
	})
}

func enableFeishu(c *gin.Context) {
	conn := feishu.GetConnector()
	if conn == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "飞书未配置 app_id/app_secret"})
		return
	}
	conn.Enable()
	c.JSON(http.StatusOK, gin.H{"ok": true, "enabled": true})
}

func disableFeishu(c *gin.Context) {
	conn := feishu.GetConnector()
	if conn == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "飞书未配置 app_id/app_secret"})
		return
	}
	conn.Disable()
	c.JSON(http.StatusOK, gin.H{"ok": true, "enabled": false})
}
