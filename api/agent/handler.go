package agent

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"yuuki798/env-king/biz/agent"
	"yuuki798/env-king/biz/skills"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/agent")
	g.GET("/state", getState)
	g.GET("/skills", listSkills)
	g.GET("/default-skills", getDefaultSkills)
	g.PUT("/default-skills", putDefaultSkills)
	g.GET("/sessions", listSessions)
	g.GET("/sessions/:id", getSession)
	g.POST("/chat", chat)
	g.POST("/chat/stream", chatStream)
	g.DELETE("/sessions/:id", deleteSession)
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

func getDefaultSkills(c *gin.Context) {
	ids, err := agent.GetDefaultSkillIDs()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"defaultSkills": []string{}})
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"defaultSkills": ids})
}

func putDefaultSkills(c *gin.Context) {
	var req struct {
		DefaultSkills []string `json:"defaultSkills"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "defaultSkills required"})
		return
	}
	if err := agent.SetDefaultSkillIDs(req.DefaultSkills); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getState(c *gin.Context) {
	s := agent.GetState()
	c.JSON(http.StatusOK, s)
}

func listSessions(c *gin.Context) {
	list, err := agent.ListConversations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []agent.Conversation{}
	}
	c.JSON(http.StatusOK, list)
}

func getSession(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	sess, err := agent.GetConversation(id)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sess)
}

func deleteSession(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	if err := agent.DeleteConversation(id); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
	flushMemoryAfterChat("", req.Message, reply)
	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func chatStream(c *gin.Context) {
	var req struct {
		SessionId    string   `json:"sessionId"`
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

	history := make([]struct{ Role, Content string }, 0, 32)
	var summary string
	if req.SessionId != "" {
		if sess, err := agent.GetConversation(req.SessionId); err == nil {
			// 对已有会话执行短期记忆压缩：早期轮次合并为摘要，仅保留最近若干条完整消息。
			if sum, turns, changed, cerr := agent.CompressSessionHistory(c.Request.Context(), sess); cerr == nil {
				summary = strings.TrimSpace(sum)
				for _, t := range turns {
					history = append(history, struct{ Role, Content string }{Role: t.Role, Content: t.Content})
				}
				if changed {
					_ = agent.SaveConversation(sess)
				}
			} else {
				// 压缩失败时退回到完整 history，保证功能可用性
				for _, m := range sess.Messages {
					history = append(history, struct{ Role, Content string }{Role: m.Role, Content: m.Content})
				}
			}
		}
	}
	if len(history) == 0 && len(req.History) > 0 {
		for i := range req.History {
			history = append(history, struct{ Role, Content string }{Role: req.History[i].Role, Content: req.History[i].Content})
		}
	}

	// 若存在会话摘要，则以一条“虚拟助手消息”形式插入到 history 开头，供 LLM 参考。
	if summary != "" {
		prefix := "（以下是本次会话的简要摘要，请认真参考其中的事实与约束，但不要逐字复述，也不要单独输出这段文字。）\n" + summary
		history = append([]struct{ Role, Content string }{
			{Role: "assistant", Content: prefix},
		}, history...)
	}

	streamFn := func(ctx context.Context, chunk []byte) error {
		c.SSEvent("message", string(chunk))
		c.Writer.Flush()
		return nil
	}
	fullReply, formFill, err := agent.ChatStream(c.Request.Context(), req.CurrentPage, req.Message, history, req.ActiveSkills, streamFn)
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		fullReply = "[错误] " + err.Error()
	}
	if formFill != nil {
		raw, _ := json.Marshal(formFill)
		c.SSEvent("form_fill", string(raw))
		c.Writer.Flush()
	}
	var sessionId string
	if req.SessionId != "" {
		_, _ = agent.AppendToConversation(req.SessionId, req.Message, fullReply)
		sessionId = req.SessionId
	} else {
		if created, createErr := agent.CreateConversation(req.Message, fullReply); createErr == nil {
			sessionId = created.ID
		} else {
			log.Printf("[agent] CreateConversation failed: %v", createErr)
		}
	}
	if sessionId != "" {
		c.SSEvent("session", sessionId)
		c.Writer.Flush()
	}
	flushMemoryAfterChat(sessionId, req.Message, fullReply)
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

const memoryValueMaxLen = 4000

// flushMemoryAfterChat 在 Agent 对话结束后写入记忆摘要。
// 当前实现：
// - 将本轮 user+assistant 的简要内容写入 {workspaceDir}/memory/long-term.md 作为长期记忆。
// - 保留少量进程内短期记忆（last_turn / session:{id}:last），供后续扩展使用。
func flushMemoryAfterChat(sessionId, userMsg, assistantReply string) {
	value := strings.TrimSpace(userMsg) + "\n---\n" + strings.TrimSpace(assistantReply)
	if len(value) > memoryValueMaxLen {
		value = value[:memoryValueMaxLen] + "…"
	}
	// 进程内短期记忆（可选保留，暂未对外暴露）
	store := skills.GetMemoryStore()
	_ = store.AddShortTerm("last_turn", value)
	if sessionId != "" {
		_ = store.AddShortTerm("session:"+sessionId+":last", value)
	}
}
