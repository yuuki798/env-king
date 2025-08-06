package ek

import (
	"github.com/spf13/cobra"
	"yuuki798/env-king/biz/dockerhelper"
)

var MirrorModeCommand = &cobra.Command{
	Use:   "mirrormode",
	Short: "Mirror mode for Docker, includes speed test for mirrors",
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
		ok = dockerhelper.DoMirrorMode()
		if ok {
			cmd.Println("Mirror mode enabled successfully")
		} else {
			cmd.Println("Failed to enable mirror mode")
		}
	},
}
