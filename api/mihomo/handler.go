package mihomo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"yuuki798/env-king/biz/mihomo"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/mihomo")
	g.GET("/status", getStatus)
	g.POST("/start", start)
	g.POST("/stop", stop)
	g.GET("/config", getConfig)
	g.PUT("/config", putConfig)
	g.POST("/restart", restart)
	// 系统代理
	g.GET("/system-proxy", getSystemProxy)
	g.POST("/system-proxy/on", systemProxyOn)
	g.POST("/system-proxy/off", systemProxyOff)
	// 订阅
	g.GET("/subscriptions", listSubscriptions)
	g.POST("/subscriptions", addSubscription)
	g.DELETE("/subscriptions/:id", deleteSubscription)
	g.POST("/subscriptions/apply", applySubscriptions)
	// 运行模式（转发到 mihomo external-controller）
	g.GET("/mode", getMode)
	g.PUT("/mode", putMode)
	// 代理/节点（转发到 mihomo external-controller）
	g.GET("/proxies", getProxies)
	g.GET("/proxies/:name", getProxy)
	g.PUT("/proxies/:name", putProxy)
}

func getStatus(c *gin.Context) {
	running := mihomo.IsRunning()
	port := mihomo.ProxyPort()
	c.JSON(http.StatusOK, gin.H{"running": running, "proxyPort": port, "mixedPort": port})
}

func start(c *gin.Context) {
	if err := mihomo.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func stop(c *gin.Context) {
	if err := mihomo.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getConfig(c *gin.Context) {
	body, err := mihomo.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/x-yaml", body)
}

func putConfig(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := mihomo.SetConfig(body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func restart(c *gin.Context) {
	_ = mihomo.Stop()
	if err := mihomo.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getSystemProxy(c *gin.Context) {
	c.JSON(http.StatusOK, mihomo.GetSystemProxyStatus())
}

func systemProxyOn(c *gin.Context) {
	if err := mihomo.SetSystemProxy(true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": mihomo.GetSystemProxyStatus()})
}

func systemProxyOff(c *gin.Context) {
	if err := mihomo.SetSystemProxy(false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": mihomo.GetSystemProxyStatus()})
}

func listSubscriptions(c *gin.Context) {
	list, err := mihomo.ListSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []mihomo.Subscription{}
	}
	c.JSON(http.StatusOK, list)
}

func addSubscription(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub, err := mihomo.AddSubscription(body.Name, body.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func deleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	if err := mihomo.DeleteSubscription(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func applySubscriptions(c *gin.Context) {
	if err := mihomo.ApplySubscriptionsToConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getMode(c *gin.Context) {
	data, code, err := mihomo.ProxyRequest("GET", "/configs", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if code != http.StatusOK {
		c.Data(code, "application/json", data)
		return
	}
	// 只返回 mode 字段
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		c.Data(http.StatusOK, "application/json", data)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mode": cfg["mode"]})
}

func putMode(c *gin.Context) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed := map[string]bool{"rule": true, "global": true, "direct": true}
	if !allowed[body.Mode] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be rule / global / direct"})
		return
	}
	data, code, err := mihomo.ProxyRequest("PATCH", "/configs", map[string]string{"mode": body.Mode})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if code != http.StatusOK && code != http.StatusNoContent {
		c.Data(code, "application/json", data)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "mode": body.Mode})
}

func getProxies(c *gin.Context) {
	data, code, err := mihomo.ProxyRequest("GET", "/proxies", nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if code != http.StatusOK {
		c.Data(code, "application/json", data)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func getProxy(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	data, code, err := mihomo.ProxyRequest("GET", "/proxies/"+name, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if code != http.StatusOK {
		c.Data(code, "application/json", data)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func putProxy(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, code, err := mihomo.ProxyRequest("PUT", "/proxies/"+name, map[string]string{"name": body.Name})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if code != http.StatusOK && code != http.StatusNoContent {
		c.Data(code, "application/json", data)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
