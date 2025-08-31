package server

import (
	"log"
	"yuuki798/env-king/initial"

	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务器",
	Long:  `启动服务器，提供 API 接口供前端调用`,
	Run: func(cmd *cobra.Command, args []string) {
		initial.InitAll()
		log.Println("启动了服务器")
	},
}
