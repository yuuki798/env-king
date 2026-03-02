package initial

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"yuuki798/env-king/internal/api"
)

func InitRouter(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	api.Setup(r)

	// SPA 静态托管：若有 web/dist，则未匹配路由返回 index.html
	dist := findDistDir()
	if dist != "" {
		r.Static("/assets", filepath.Join(dist, "assets"))
		r.NoRoute(func(c *gin.Context) {
			// SPA: 所有未匹配路由返回 index.html
			c.File(filepath.Join(dist, "index.html"))
		})
	}
}

func findDistDir() string {
	roots := []string{"."}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	for _, root := range roots {
		for _, d := range []string{"web/dist", "dist"} {
			p := filepath.Join(root, d)
			if f, err := os.Stat(p); err == nil && f.IsDir() {
				return p
			}
		}
	}
	return ""
}
