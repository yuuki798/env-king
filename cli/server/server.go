package server

import (
	"log"
	"yuuki798/env-king/initial"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务器",
	Long:  `启动服务器，提供 API 接口供前端调用，同时可托管 web 静态资源`,
	Run: func(cmd *cobra.Command, args []string) {
		initial.InitAll()
		r := gin.Default()
		initial.InitRouter(r)
		log.Println("Env King server starting on :28889")
		if err := r.Run("127.0.0.1:28889"); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	},
}
