package api

import (
	"github.com/gin-gonic/gin"
	"yuuki798/env-king/internal/api/agent"
	"yuuki798/env-king/internal/api/clash"
	"yuuki798/env-king/internal/api/pipeline"
	"yuuki798/env-king/internal/api/skills"
)

func Setup(r *gin.Engine) {
	r.Use(corsMiddleware())
	api := r.Group("/api")
	{
		pipeline.Register(api)
		clash.Register(api)
		agent.Register(api)
		skills.Register(api)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
