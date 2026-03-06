package mihomo

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/viper"
)

var ErrBinNotFound = errors.New("mihomo 二进制未找到：请将 mihomo 放到 workdir/mihomo/ 目录，或配置 mihomo.bin，或确保 PATH 中有 mihomo")

const defaultConfigName = "config.yaml"

var (
	cmdMu   sync.Mutex
	runCmd  *exec.Cmd
	workDir string
	binPath string
)

func initWorkDir() (string, error) {
	dir := strings.TrimSpace(viper.GetString("mihomo.work_dir"))
	if dir == "" {
		dir = "./workdir/mihomo"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return "", err
	}
	cfgPath := filepath.Join(abs, defaultConfigName)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		tpl := []byte(`mixed-port: 17890
allow-lan: false
mode: rule
log-level: info
external-controller: 127.0.0.1:19090
rules:
  - GEOIP,CN,DIRECT
  - MATCH,DIRECT
`)
		if err := os.WriteFile(cfgPath, tpl, 0644); err != nil {
			return "", err
		}
	}
	return abs, nil
}

func getWorkDir() (string, error) {
	if workDir != "" {
		return workDir, nil
	}
	var err error
	workDir, err = initWorkDir()
	return workDir, err
}

func platformBinName() string {
	os, arch := runtime.GOOS, runtime.GOARCH
	name := "mihomo-" + os + "-" + arch
	if os == "windows" {
		return name + ".exe"
	}
	return name
}

func getBin() (string, error) {
	if binPath != "" {
		return binPath, nil
	}
	bin := strings.TrimSpace(viper.GetString("mihomo.bin"))
	if bin != "" {
		binPath = bin
		return binPath, nil
	}
	wd, err := getWorkDir()
	if err != nil {
		return "", err
	}
	// 1) 优先使用平台专用二进制 mihomo-{os}-{arch}[.exe]
	platformBin := filepath.Join(wd, platformBinName())
	if _, err := os.Stat(platformBin); err == nil {
		binPath = platformBin
		return binPath, nil
	}
	// 2) 回退到通用名 mihomo
	candidate := filepath.Join(wd, "mihomo")
	if runtime.GOOS == "windows" {
		candidate = filepath.Join(wd, "mihomo.exe")
	}
	if _, err := os.Stat(candidate); err == nil {
		binPath = candidate
		return binPath, nil
	}
	// 3) PATH 中的 mihomo
	path, err := exec.LookPath("mihomo")
	if err == nil {
		binPath = path
		return binPath, nil
	}
	return "", nil
}

func configPath() (string, error) {
	wd, err := getWorkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, defaultConfigName), nil
}

func IsRunning() bool {
	cmdMu.Lock()
	c := runCmd
	cmdMu.Unlock()
	if c == nil || c.Process == nil {
		return false
	}
	if runtime.GOOS != "windows" {
		if err := syscall.Kill(c.Process.Pid, 0); err != nil {
			cmdMu.Lock()
			if runCmd == c {
				runCmd = nil
			}
			cmdMu.Unlock()
			return false
		}
	}
	return true
}

func ProxyPort() int {
	return 17890
}

func Start() error {
	wd, err := getWorkDir()
	if err != nil {
		return err
	}
	if err := EnsureProviderFiles(); err != nil {
		// 非致命，仅打日志
		log.Printf("[mihomo] EnsureProviderFiles: %v", err)
	}
	bin, err := getBin()
	if err != nil {
		return err
	}
	if bin == "" {
		return ErrBinNotFound
	}
	cmdMu.Lock()
	if runCmd != nil && runCmd.Process != nil {
		cmdMu.Unlock()
		return nil
	}
	runCmd = exec.Command(bin, "-d", wd)
	var stdoutBuf, stderrBuf bytes.Buffer
	runCmd.Stdout = &stdoutBuf
	runCmd.Stderr = &stderrBuf
	runCmd.Stdin = nil
	if err := runCmd.Start(); err != nil {
		runCmd = nil
		cmdMu.Unlock()
		return err
	}
	cmdMu.Unlock()
	go func() {
		waitErr := runCmd.Wait()
		if stderrBuf.Len() > 0 {
			log.Printf("[mihomo] stderr: %s", stderrBuf.String())
		}
		if waitErr != nil {
			log.Printf("[mihomo] exit: %v", waitErr)
		}
		cmdMu.Lock()
		if runCmd != nil {
			runCmd = nil
		}
		cmdMu.Unlock()
	}()
	return nil
}

func Stop() error {
	cmdMu.Lock()
	c := runCmd
	runCmd = nil
	cmdMu.Unlock()
	if c == nil || c.Process == nil {
		return nil
	}
	return c.Process.Kill()
}

func GetConfig() ([]byte, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func SetConfig(body []byte) error {
	_, err := getWorkDir()
	if err != nil {
		return err
	}
	p, err := configPath()
	if err != nil {
		return err
	}
	return os.WriteFile(p, body, 0644)
}
