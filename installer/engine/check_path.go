package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const pathBegin = "# >>> claude-toolbox PATH >>>"
const pathEnd = "# <<< claude-toolbox PATH <<<"

// PathCheck 把工具目录写入 PATH。
// macOS/Linux:写入 shell 配置文件(zsh/bash),免管理员。
// Windows:用 setx 写当前用户 PATH(HKCU),免管理员。
type PathCheck struct{}

func (PathCheck) ID() string       { return "path-register" }
func (PathCheck) Name() string     { return "PATH 注册(免管理员)" }
func (PathCheck) NeedsAdmin() bool { return false }

func (p PathCheck) dirs(ctx *Context) []string {
	return []string{
		ctx.LocalBin,
		filepath.Join(ctx.NodeDir, "bin"),
		filepath.Join(ctx.NpmGlobal, "bin"),
	}
}

func (p PathCheck) shellRC(ctx *Context) string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "bash") {
		return filepath.Join(ctx.Home, ".bashrc")
	}
	// 现代 macOS 默认 zsh
	return filepath.Join(ctx.Home, ".zshrc")
}

func (p PathCheck) Detect(ctx *Context) (Status, string) {
	if ctx.OS == "windows" {
		cur := os.Getenv("PATH")
		for _, d := range p.dirs(ctx) {
			if !strings.Contains(cur, d) {
				return StatusFixable, "用户 PATH 缺少 " + d
			}
		}
		return StatusOK, "PATH 已包含工具目录"
	}
	data, err := os.ReadFile(p.shellRC(ctx))
	if err == nil && strings.Contains(string(data), pathBegin) {
		return StatusOK, "已写入 " + p.shellRC(ctx)
	}
	return StatusFixable, "shell 配置未写入 PATH"
}

func (p PathCheck) Fix(ctx *Context) error {
	if ctx.OS == "windows" {
		// 注意:setx 有 1024 字符上限,生产环境建议直接读改 HKCU\Environment 注册表项。
		add := strings.Join(p.dirs(ctx), ";")
		cur := os.Getenv("PATH")
		out, err := exec.Command("setx", "PATH", cur+";"+add).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", err, string(out))
		}
		return nil
	}
	rc := p.shellRC(ctx)
	if data, _ := os.ReadFile(rc); strings.Contains(string(data), pathBegin) {
		return nil // 幂等:已写入则跳过
	}
	var b strings.Builder
	b.WriteString("\n" + pathBegin + "\n")
	for _, d := range p.dirs(ctx) {
		fmt.Fprintf(&b, "export PATH=\"%s:$PATH\"\n", d)
	}
	b.WriteString(pathEnd + "\n")

	f, err := os.OpenFile(rc, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(b.String())
	return err
}

func (p PathCheck) Verify(ctx *Context) error {
	if ctx.OS == "windows" {
		return nil // setx 已成功;新 PATH 需重开终端生效,这里不阻塞
	}
	if st, msg := p.Detect(ctx); st != StatusOK {
		return errors.New(msg)
	}
	return nil
}
