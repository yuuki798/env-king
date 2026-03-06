package ek

import (
	"github.com/spf13/cobra"
)

var EkCmd = &cobra.Command{
	Use:   "ek",
	Short: "Env King CLI",
	Long:  `Env King CLI is a command line interface for managing environment variables and configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 帮助
		cmd.Help() // 如果没有提供子命令，则显示帮助信息
	},
}
