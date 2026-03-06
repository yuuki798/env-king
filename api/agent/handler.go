package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"yuuki798/env-king/biz/agent"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/agent")
	g.GET("/state", getState)
	g.GET("/skills", listSkills)
	g.POST("/chat", chat)
	g.POST("/chat/stream", chatStream)
	g.GET("/cron", listCron)
	g.POST("/cron", addCron)
	g.DELETE("/cron/:name", removeCron)
	g.POST("/self-modify", selfModify)
	g.POST("/git/commit", gitCommit)
	g.POST("/git/push", gitPush)
	g.POST("/deploy", deploy)
}

func listSkills(c *gin.Context) {
	skills, err := agent.ListSkills()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if skills == nil {
		skills = []agent.Skill{}
	}
	c.JSON(http.StatusOK, skills)
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

func chatStream(c *gin.Context) {
	var req struct {
		Message      string   `json:"message"`
		CurrentPage  string   `json:"currentPage"`
		ActiveSkills []string `json:"activeSkills"`
		History      []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message required"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()

	history := make([]struct{ Role, Content string }, len(req.History))
	for i := range req.History {
		history[i].Role = req.History[i].Role
		history[i].Content = req.History[i].Content
	}

	streamFn := func(ctx context.Context, chunk []byte) error {
		c.SSEvent("message", string(chunk))
		c.Writer.Flush()
		return nil
	}
	_, formFill, err := agent.ChatStream(c.Request.Context(), req.CurrentPage, req.Message, history, req.ActiveSkills, streamFn)
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}
	if formFill != nil {
		raw, _ := json.Marshal(formFill)
		c.SSEvent("form_fill", string(raw))
		c.Writer.Flush()
	}
	c.SSEvent("done", "")
	c.Writer.Flush()
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
