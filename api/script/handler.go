package script

import (
	"net/http"

	bizscript "yuuki798/env-king/biz/script"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	g := r.Group("/script")
	g.GET("/jobs", listJobs)
	g.GET("/jobs/:id", getJob)
	g.POST("/jobs/:id/cancel", cancelJob)
	g.POST("/run-flow", runFlow)
	g.POST("/run-command", runCommand)
	g.GET("/presets", presets)
	g.GET("/workflows", listWorkflows)
	g.GET("/workflows/:id", getWorkflow)
	g.POST("/workflows", createWorkflow)
	g.PUT("/workflows/:id", updateWorkflow)
	g.DELETE("/workflows/:id", deleteWorkflow)
	g.POST("/workflows/init", initWorkflows)   // 静态路径放前面，避免被 :id 匹配
	g.POST("/workflows/:id/run", runWorkflow)
}

func listJobs(c *gin.Context) {
	jobs := bizscript.ListJobs()
	c.JSON(http.StatusOK, jobs)
}

func getJob(c *gin.Context) {
	id := c.Param("id")
	job, ok := bizscript.GetJob(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func cancelJob(c *gin.Context) {
	id := c.Param("id")
	job, err := bizscript.CancelJob(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

func runFlow(c *gin.Context) {
	var req bizscript.FlowRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := bizscript.RunFlow(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

func runCommand(c *gin.Context) {
	var req bizscript.RunCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := bizscript.RunCommand(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

func presets(c *gin.Context) {
	c.JSON(http.StatusOK, bizscript.Presets())
}

func listWorkflows(c *gin.Context) {
	list, err := bizscript.ListWorkflows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func getWorkflow(c *gin.Context) {
	id := c.Param("id")
	wf, err := bizscript.GetWorkflow(id)
	if err != nil || wf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}
	c.JSON(http.StatusOK, wf)
}

func createWorkflow(c *gin.Context) {
	var body struct {
		Name          string                       `json:"name"`
		WorkDir       string                       `json:"workDir"`
		Steps         []bizscript.Step            `json:"steps"`
		DefaultInputs map[string]string           `json:"defaultInputs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wf := &bizscript.Workflow{
		Name:          body.Name,
		WorkDir:       body.WorkDir,
		Steps:         body.Steps,
		DefaultInputs: body.DefaultInputs,
	}
	out, err := bizscript.CreateWorkflow(wf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func updateWorkflow(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Name          string                       `json:"name"`
		WorkDir       string                       `json:"workDir"`
		Steps         []bizscript.Step            `json:"steps"`
		DefaultInputs map[string]string           `json:"defaultInputs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wf := &bizscript.Workflow{
		ID:            id,
		Name:          body.Name,
		WorkDir:       body.WorkDir,
		Steps:         body.Steps,
		DefaultInputs: body.DefaultInputs,
	}
	if err := bizscript.UpdateWorkflow(wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func deleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if err := bizscript.DeleteWorkflow(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func runWorkflow(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		WorkDir string            `json:"workDir,omitempty"`
		Inputs  map[string]string `json:"inputs,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := bizscript.WorkflowRunRequest{
		WorkflowID: id,
		WorkDir:    body.WorkDir,
		Inputs:     body.Inputs,
	}
	job, err := bizscript.RunWorkflow(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

func initWorkflows(c *gin.Context) {
	if err := bizscript.InitPresetWorkflows(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
