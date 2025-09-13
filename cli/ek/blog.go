package ek

import (
	"fmt"
	"yuuki798/env-king/biz/blog"

	"github.com/spf13/cobra"
)

var BlogCmd = &cobra.Command{
	Use:   "blog",
	Short: "blog CLI",
	Long:  `blog Manager - Interactive shell for managing blog articles`,
	Run: func(cmd *cobra.Command, args []string) {
		hub := blog.NewHub()

		// 从存储恢复数据
		err := hub.RecoverFromStorage()
		if err != nil {
			fmt.Printf("Warning: Failed to recover from storage: %v\n", err)
		}

		// 获取用户名参数
		username, _ := cmd.Flags().GetString("user")
		if username == "" {
			username = "default" // 默认用户名
		}

		// 检查用户是否存在，如果不存在则创建
		_, exists := hub.GetUser(username)
		if !exists {
			// 阻塞并让用户输入localPath（博客目录）
			fmt.Printf("用户 %s 不存在，创建新用户，请输入localPath\n", username)
			var localPath string
			fmt.Print("localPath: ")
			_, err := fmt.Scanln(&localPath)
			if err != nil {
				fmt.Printf("Error reading localPath: %v\n", err)
				return
			}
			var hexoPath string
			fmt.Print("HexoPath: (可选，直接回车跳过) ")
			_, err = fmt.Scanln(&hexoPath)
			if err != nil {
				fmt.Printf("Error reading localPath: %v\n", err)
				return
			}
			// 创建默认用户
			user := blog.NewUser(username, []string{"hexo"}, []byte{}, localPath, hexoPath)
			err = hub.AddUser(user)
			if err != nil {
				fmt.Printf("Error creating user: %v\n", err)
				return
			}
		}

		// 创建并启动shell
		shell := blog.NewShell(hub, username)
		err = shell.Start()
		if err != nil {
			fmt.Printf("Error starting shell: %v\n", err)
		}
	},
}

func InitBlogCLi() {
	BlogCmd.Flags().StringP("user", "u", "default", "Username for blog management")
}
