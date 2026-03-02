package clash

import (
	"os/exec"
	"runtime"
	"strings"
)

func IsRunning() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	cmd := exec.Command("pgrep", "-x", "clash")
	out, err := cmd.CombinedOutput()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func ProxyPort() int {
	// 默认 mixed-port 7890
	return 7890
}

func Start() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	// 若已配置 clashctl，则: clashctl on 或 clashon
	// 否则尝试直接启动 clash（若已安装）
	cmd := exec.Command("bash", "-c", "clashon 2>/dev/null || clash 2>/dev/null &")
	return cmd.Run()
}

func Stop() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	cmd := exec.Command("pkill", "-x", "clash")
	_ = cmd.Run()
	return nil
}
