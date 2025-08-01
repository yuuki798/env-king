package ek

import (
	"github.com/spf13/cobra"
	"yuuki798/env-king/biz/dockerhelper"
)

var SpeedCmd = &cobra.Command{
	Use:   "speed",
	Short: "speed test for mirrors",
	Run: func(cmd *cobra.Command, args []string) {
		dockerhelper := dockerhelper.NewHub()
		ok, url2Durations := dockerhelper.SpeedTest()
		if !ok {
			cmd.Println("Speed test failed")
			return
		}
		cmd.Println("Speed test completed successfully")
		cmd.Println("Results:")
		for url, duration := range url2Durations {
			cmd.Printf("URL: %s, Duration: %d ms\n", url, duration.Duration)
		}
		cmd.Println("You can use the following command to set the best mirror:")
	},
}
