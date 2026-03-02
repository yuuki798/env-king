package ek

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ClashCmd = &cobra.Command{
	Use:   "clash",
	Short: "Clash 代理管理（git clone 加速）",
	Long:  `集成 clash-for-linux-install，提供 git clone 等操作的代理加速`,
}

var clashInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Clash（仅 Linux，需在 clash-for-linux-install 目录执行）",
	RunE:  runClashInstall,
}

var clashOnCmd = &cobra.Command{
	Use:   "on",
	Short: "启动 Clash 代理",
	RunE:  runClashOn,
}

var clashOffCmd = &cobra.Command{
	Use:   "off",
	Short: "关闭 Clash 代理",
	RunE:  runClashOff,
}

func init() {
	ClashCmd.AddCommand(clashInstallCmd)
	ClashCmd.AddCommand(clashOnCmd)
	ClashCmd.AddCommand(clashOffCmd)
}

func runClashInstall(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		cmd.Println("clash install 仅支持 Linux")
		return nil
	}
	// 从当前目录或可执行文件所在目录向上查找 clash-for-linux-install
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
	installDir := viper.GetString("clash.install_dir")
	if installDir == "" {
		installDir = filepath.Join(os.TempDir(), "env-king-clash-install")
	}
	cmd.Printf("未找到 clash-for-linux-install 目录。请将 clash-for-linux-install 置于 env-king 项目根目录，或配置 clash.install_dir: %s\n", installDir)
	return nil
}

func runClashOn(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		cmd.Println("clash 仅支持 Linux")
		return nil
	}
	c := exec.Command("clashon")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		cmd.Printf("启动失败，请先执行 env-king clash install: %v\n", err)
		return err
	}
	cmd.Println("Clash 已启动，git clone 将走代理")
	return nil
}

func runClashOff(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	c := exec.Command("clashoff")
	_ = c.Run()
	cmd.Println("Clash 已关闭")
	return nil
}
