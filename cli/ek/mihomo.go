package ek

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var MihomoCmd = &cobra.Command{
	Use:   "mihomo",
	Short: "Mihomo 代理管理（git clone 加速）",
	Long:  `嵌入 mihomo 内核，提供 git clone 等操作的代理加速`,
}

var mihomoInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 运行环境（仅 Linux，需在 clash-for-linux-install 目录执行）",
	RunE:  runMihomoInstall,
}

var mihomoOnCmd = &cobra.Command{
	Use:   "on",
	Short: "启动 Mihomo 代理",
	RunE:  runMihomoOn,
}

var mihomoOffCmd = &cobra.Command{
	Use:   "off",
	Short: "关闭 Mihomo 代理",
	RunE:  runMihomoOff,
}

func init() {
	MihomoCmd.AddCommand(mihomoInstallCmd)
	MihomoCmd.AddCommand(mihomoOnCmd)
	MihomoCmd.AddCommand(mihomoOffCmd)
}

func runMihomoInstall(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		cmd.Println("mihomo install 仅支持 Linux")
		return nil
	}
	cwd, _ := os.Getwd()
	execPath, _ := os.Executable()
	roots := []string{cwd, filepath.Dir(execPath)}
	for _, r := range roots {
		for d := r; d != "/" && d != "."; d = filepath.Dir(d) {
			installDir := filepath.Join(d, "clash-for-linux-install")
			if _, err := os.Stat(installDir); err == nil {
				run := exec.Command("bash", "install.sh")
				run.Dir = installDir
				run.Stdout = os.Stdout
				run.Stderr = os.Stderr
				run.Stdin = os.Stdin
				return run.Run()
			}
		}
	}
	workDir := viper.GetString("mihomo.work_dir")
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "env-king-mihomo")
	}
	cmd.Printf("未找到 clash-for-linux-install 目录。请将 clash-for-linux-install 置于 env-king 项目根目录，或配置 mihomo.work_dir: %s\n", workDir)
	return nil
}

func runMihomoOn(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		cmd.Println("mihomo 仅支持 Linux（当前实现）")
		return nil
	}
	c := exec.Command("clashon")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		cmd.Printf("启动失败，请先执行 env-king mihomo install: %v\n", err)
		return err
	}
	cmd.Println("Mihomo 已启动，git clone 将走代理")
	return nil
}

func runMihomoOff(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	c := exec.Command("clashoff")
	_ = c.Run()
	cmd.Println("Mihomo 已关闭")
	return nil
}
