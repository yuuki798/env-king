package agent

import (
	"net/http"
	"yuuki798/env-king/biz/agent"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/agent")
	g.GET("/state", getState)
	g.POST("/chat", chat)
	g.GET("/cron", listCron)
	g.POST("/cron", addCron)
	g.DELETE("/cron/:name", removeCron)
	g.POST("/self-modify", selfModify)
	g.POST("/git/commit", gitCommit)
	g.POST("/git/push", gitPush)
	g.POST("/deploy", deploy)
}

func getState(c *gin.Context) {
	s := agent.GetState()
	c.JSON(http.StatusOK, s)
}

func chat(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message required"})
		return
	}
	reply, err := agent.Chat(req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func listCron(c *gin.Context) {
	jobs, err := agent.CronList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func addCron(c *gin.Context) {
	var req struct {
		Spec   string `json:"spec"`
		Script string `json:"script"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agent.CronAdd(req.Spec, req.Script); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func removeCron(c *gin.Context) {
	name := c.Param("name")
	if err := agent.CronRemove(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func selfModify(c *gin.Context) {
	var req struct {
		Diff string `json:"diff"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agent.SelfModify(req.Diff); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func gitCommit(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agent.GitCommit(req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func gitPush(c *gin.Context) {
	if err := agent.GitPush(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func deploy(c *gin.Context) {
	if err := agent.Deploy(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
