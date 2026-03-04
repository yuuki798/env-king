package skills

import (
	"net/http"
	"yuuki798/env-king/biz/skills"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/skills")
	g.GET("/mcp", listMcp)
	g.POST("/mcp", installMcp)
	g.DELETE("/mcp/:name", uninstallMcp)
	g.GET("/list", listSkills)
	g.POST("/install", installSkill)
	g.DELETE("/skill/:name", uninstallSkill)
	g.GET("/load-balance", getLoadBalance)
	g.PUT("/load-balance", setLoadBalance)
}

func listMcp(c *gin.Context) {
	list := skills.ListMCP()
	c.JSON(http.StatusOK, list)
}

func installMcp(c *gin.Context) {
	var req struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source required"})
		return
	}
	if err := skills.InstallMCP(req.Source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func uninstallMcp(c *gin.Context) {
	name := c.Param("name")
	if err := skills.UninstallMCP(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func listSkills(c *gin.Context) {
	list := skills.ListSkills()
	c.JSON(http.StatusOK, list)
}

func installSkill(c *gin.Context) {
	var req struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source required"})
		return
	}
	if err := skills.InstallSkill(req.Source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func uninstallSkill(c *gin.Context) {
	name := c.Param("name")
	if err := skills.UninstallSkill(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getLoadBalance(c *gin.Context) {
	cfg, err := skills.GetLoadBalanceConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func setLoadBalance(c *gin.Context) {
	var cfg skills.LoadBalanceConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := skills.SetLoadBalanceConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
