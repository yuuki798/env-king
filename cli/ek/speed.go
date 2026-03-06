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
		for i, x := range url2Durations {
			if x.Duration >= 10000 {
				cmd.Printf("%v, %s, Duration: N/A\n", i, x.Url)
			} else {
				cmd.Printf("%v, %s, Duration: %d ms\n", i, x.Url, x.Duration)
			}
		}
	},
}
