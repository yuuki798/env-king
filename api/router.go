package api

import (
	"yuuki798/env-king/api/agent"
	"yuuki798/env-king/api/config"
	apifeishu "yuuki798/env-king/api/feishu"
	"yuuki798/env-king/api/mihomo"
	"yuuki798/env-king/api/script"
	"yuuki798/env-king/api/skills"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	r.Use(corsMiddleware())
	api := r.Group("/api")
	{
		config.Register(api)
		script.Register(api)
		mihomo.Register(api)
		agent.Register(api)
		skills.Register(api)
		apifeishu.Register(api)
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
