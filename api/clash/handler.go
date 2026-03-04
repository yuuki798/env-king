package clash

import (
	"net/http"
	"runtime"
	"yuuki798/env-king/biz/clash"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/clash")
	g.GET("/status", getStatus)
	g.POST("/start", start)
	g.POST("/stop", stop)
}

func getStatus(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusOK, gin.H{"running": false, "note": "clash only supports Linux"})
		return
	}
	running := clash.IsRunning()
	port := clash.ProxyPort()
	c.JSON(http.StatusOK, gin.H{"running": running, "proxyPort": port, "mixedPort": port})
}

func start(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clash only supports Linux"})
		return
	}
	if err := clash.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func stop(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clash only supports Linux"})
		return
	}
	if err := clash.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
