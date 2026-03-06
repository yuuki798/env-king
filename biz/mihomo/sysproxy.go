package mihomo

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// SystemProxyStatus 系统代理状态
type SystemProxyStatus struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

// SetSystemProxy 开启或关闭系统代理（指向 mixed-port）
func SetSystemProxy(enable bool) error {
	port := ProxyPort()
	switch runtime.GOOS {
	case "darwin":
		return setSystemProxyMacOS(enable, "127.0.0.1", port)
	case "linux":
		return setSystemProxyLinux(enable, "127.0.0.1", port)
	default:
		return fmt.Errorf("系统代理设置暂不支持 %s 平台，请手动设置 HTTP_PROXY=http://127.0.0.1:%d", runtime.GOOS, port)
	}
}

// GetSystemProxyStatus 查询当前系统代理状态
func GetSystemProxyStatus() SystemProxyStatus {
	port := ProxyPort()
	switch runtime.GOOS {
	case "darwin":
		return getSystemProxyStatusMacOS(port)
	default:
		return SystemProxyStatus{Host: "127.0.0.1", Port: port}
	}
}

// --- macOS ---

func macOSNetworkServices() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil
	}
	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") || line == "Thunderbolt Bridge" {
			continue
		}
		line = strings.TrimPrefix(line, "* ")
		services = append(services, line)
	}
	return services
}

func setSystemProxyMacOS(enable bool, host string, port int) error {
	services := macOSNetworkServices()
	if len(services) == 0 {
		return fmt.Errorf("未找到网络服务，请手动设置")
	}
	portStr := strconv.Itoa(port)
	var errs []string
	for _, svc := range services {
		var cmds [][]string
		if enable {
			cmds = [][]string{
				{"networksetup", "-setwebproxy", svc, host, portStr},
				{"networksetup", "-setwebproxystate", svc, "on"},
				{"networksetup", "-setsecurewebproxy", svc, host, portStr},
				{"networksetup", "-setsecurewebproxystate", svc, "on"},
				{"networksetup", "-setsocksfirewallproxy", svc, host, portStr},
				{"networksetup", "-setsocksfirewallproxystate", svc, "on"},
			}
		} else {
			cmds = [][]string{
				{"networksetup", "-setwebproxystate", svc, "off"},
				{"networksetup", "-setsecurewebproxystate", svc, "off"},
				{"networksetup", "-setsocksfirewallproxystate", svc, "off"},
			}
		}
		for _, args := range cmds {
			if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
				errs = append(errs, fmt.Sprintf("[%s] %s: %v", svc, args[1], err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分网络服务设置失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

func getSystemProxyStatusMacOS(port int) SystemProxyStatus {
	services := macOSNetworkServices()
	if len(services) == 0 {
		return SystemProxyStatus{Host: "127.0.0.1", Port: port}
	}
	out, err := exec.Command("networksetup", "-getwebproxy", services[0]).Output()
	if err != nil {
		return SystemProxyStatus{Host: "127.0.0.1", Port: port}
	}
	lines := strings.Split(string(out), "\n")
	enabled := false
	server := ""
	proxyPort := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Enabled: Yes") {
			enabled = true
		}
		if strings.HasPrefix(line, "Server: ") {
			server = strings.TrimPrefix(line, "Server: ")
		}
		if strings.HasPrefix(line, "Port: ") {
			p, _ := strconv.Atoi(strings.TrimPrefix(line, "Port: "))
			proxyPort = p
		}
	}
	// 只有指向我们的端口才算"已开启"
	if server == "127.0.0.1" && proxyPort == port {
		return SystemProxyStatus{Enabled: enabled, Host: server, Port: proxyPort}
	}
	return SystemProxyStatus{Enabled: false, Host: "127.0.0.1", Port: port}
}

// --- Linux ---

func setSystemProxyLinux(enable bool, host string, port int) error {
	portStr := strconv.Itoa(port)
	proxyURL := fmt.Sprintf("http://%s:%s", host, portStr)
	// 尝试 gsettings（GNOME）
	if _, err := exec.LookPath("gsettings"); err == nil {
		if enable {
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", host).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", host).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()
		} else {
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
		}
		return nil
	}
	// 非 GNOME 提示手动设置环境变量
	if enable {
		return fmt.Errorf("请手动设置环境变量：export HTTP_PROXY=%s HTTPS_PROXY=%s ALL_PROXY=socks5://%s:%s", proxyURL, proxyURL, host, portStr)
	}
	return fmt.Errorf("请手动取消：unset HTTP_PROXY HTTPS_PROXY ALL_PROXY")
}
