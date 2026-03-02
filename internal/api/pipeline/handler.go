package pipeline

import (
	"net/http"
	"sync"
	"yuuki798/env-king/biz/pipeline"

	"github.com/gin-gonic/gin"
)

var (
	svc   *pipeline.Service
	svcMu sync.Mutex
)

func initSvc() *pipeline.Service {
	svcMu.Lock()
	defer svcMu.Unlock()
	if svc == nil {
		svc = pipeline.NewService()
	}
	return svc
}

func Register(r *gin.RouterGroup) {
	g := r.Group("/pipeline")
	g.GET("/jobs", listJobs)
	g.POST("/trigger", triggerBuild)
}

func listJobs(c *gin.Context) {
	s := initSvc()
	jobs := s.ListJobs()
	c.JSON(http.StatusOK, jobs)
}

func triggerBuild(c *gin.Context) {
	var req struct {
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo required"})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	s := initSvc()
	job, err := s.Trigger(req.Repo, req.Branch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}
