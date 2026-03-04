package infra

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Shell 负责在 shell 中执行命令，类似 CLI 执行器。
type Shell struct {
	// Path 使用的 shell 路径，空则根据系统自动选择（Unix: /bin/sh, Windows: cmd）
	Path string
	// Dir 执行命令时的工作目录，空表示当前进程工作目录
	Dir string
	// Env 可选的额外环境变量，格式为 "KEY=value"
	Env []string
}

// Result 单次命令执行结果。
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error // 启动/等待失败时的错误，非进程退出码
}

// NewShell 创建一个使用系统默认 shell 的 Shell。
func NewShell() *Shell {
	path := "/bin/sh"
	if runtime.GOOS == "windows" {
		path = "cmd"
	}
	return &Shell{Path: path}
}

// NewShellWithPath 指定 shell 路径创建 Shell（如 "/bin/bash"）。
func NewShellWithPath(path string) *Shell {
	return &Shell{Path: path}
}

// Run 执行一条 shell 命令字符串（会通过 shell 解析），返回标准输出、标准错误和退出码。
// 若 Path 为 cmd，在 Windows 上会使用 /c 传参。
func (s *Shell) Run(ctx context.Context, cmd string) Result {
	return s.run(ctx, cmd, false)
}

// RunQuiet 与 Run 相同，但不在 Result 中保留 stdout/stderr，适合只关心成功与否的场景。
func (s *Shell) RunQuiet(ctx context.Context, cmd string) Result {
	return s.run(ctx, cmd, true)
}

func (s *Shell) run(ctx context.Context, cmd string, quiet bool) Result {
	var stdout, stderr bytes.Buffer
	shell := s.Path
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "/bin/sh"
		}
	}

	var c *exec.Cmd
	if runtime.GOOS == "windows" && shell == "cmd" {
		c = exec.CommandContext(ctx, shell, "/c", cmd)
	} else {
		c = exec.CommandContext(ctx, shell, "-c", cmd)
	}

	if s.Dir != "" {
		c.Dir = s.Dir
	}
	if len(s.Env) > 0 {
		c.Env = append(c.Env, s.Env...)
	}
	if !quiet {
		c.Stdout = &stdout
		c.Stderr = &stderr
	}

	err := c.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			// 启动失败或 context 取消等
			return Result{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: -1,
				Err:      err,
			}
		}
	}

	return Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: code,
	}
}

// RunCombined 执行命令并将 stdout 与 stderr 合并到 Result.Stdout，Result.Stderr 为空。
func (s *Shell) RunCombined(ctx context.Context, cmd string) Result {
	var combined bytes.Buffer
	shell := s.Path
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "/bin/sh"
		}
	}

	var c *exec.Cmd
	if runtime.GOOS == "windows" && shell == "cmd" {
		c = exec.CommandContext(ctx, shell, "/c", cmd)
	} else {
		c = exec.CommandContext(ctx, shell, "-c", cmd)
	}

	if s.Dir != "" {
		c.Dir = s.Dir
	}
	if len(s.Env) > 0 {
		c.Env = append(c.Env, s.Env...)
	}
	c.Stdout = &combined
	c.Stderr = &combined

	err := c.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			return Result{
				Stdout:   combined.String(),
				Stderr:   "",
				ExitCode: -1,
				Err:      err,
			}
		}
	}

	return Result{
		Stdout:   combined.String(),
		Stderr:   "",
		ExitCode: code,
	}
}

// WithDir 返回工作目录为 dir 的副本，不修改原 Shell。
func (s *Shell) WithDir(dir string) *Shell {
	out := *s
	out.Dir = dir
	return &out
}

// WithEnv 返回追加了 env 的副本。env 格式为 "KEY=value"。
func (s *Shell) WithEnv(env ...string) *Shell {
	out := *s
	out.Env = append([]string(nil), s.Env...)
	out.Env = append(out.Env, env...)
	return &out
}

// Success 表示命令是否成功退出（ExitCode == 0 且无 Err）。
func (r Result) Success() bool {
	return r.Err == nil && r.ExitCode == 0
}

// String 便于日志输出。
func (r Result) String() string {
	if r.Err != nil {
		return fmt.Sprintf("exit=%d err=%v stdout=%q stderr=%q", r.ExitCode, r.Err, r.Stdout, r.Stderr)
	}
	return fmt.Sprintf("exit=%d stdout=%q stderr=%q", r.ExitCode, r.Stdout, r.Stderr)
}
