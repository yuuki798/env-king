package cli

import (
	"os"
	"yuuki798/env-king/cli/ek"
	"yuuki798/env-king/cli/server"

	"github.com/spf13/cobra"
)

var cfgFile string

var RootCmd = &cobra.Command{}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func InitCli() {
	//cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	// router
	RootCmd.AddCommand(server.ServerCmd)
	RootCmd.AddCommand(ek.EkCmd)
	RootCmd.AddCommand(ek.BlogCmd)
	
	ek.EkCmd.AddCommand(ek.SpeedCmd)
	ek.EkCmd.AddCommand(ek.MirrorModeCommand)

	ek.EkCmd.AddCommand()

	// flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ek.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	ek.InitBlogCLi()
}
