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
// Windows:直接读写 HKCU\Environment 注册表项,避免 setx 的 1024 字符截断问题。
type PathCheck struct{}

func (PathCheck) ID() string       { return "path-register" }
func (PathCheck) Name() string     { return "PATH 注册(免管理员)" }
func (PathCheck) NeedsAdmin() bool { return false }

func (p PathCheck) dirs(ctx *Context) []string {
	if ctx.OS == "windows" {
		// Windows Node.js 解压后 node.exe/npm.cmd 在 NodeDir 根目录，无 bin 子目录。
		// npm 全局包在 Windows 上也直接落在 NpmGlobal 根目录。
		return []string{
			ctx.LocalBin,
			ctx.NodeDir,
			ctx.NpmGlobal,
		}
	}
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
	return filepath.Join(ctx.Home, ".zshrc")
}

// winUserPath 从注册表读取当前用户的 PATH(HKCU\Environment)。
// 不使用 os.Getenv，因为那是进程启动时已合并系统 PATH 的快照，写回会导致路径膨胀。
func winUserPath() string {
	out, err := exec.Command("reg", "query", `HKCU\Environment`, "/v", "Path").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		for _, t := range []string{"REG_EXPAND_SZ", "REG_SZ"} {
			if idx := strings.Index(line, t); idx != -1 {
				return strings.TrimSpace(line[idx+len(t):])
			}
		}
	}
	return ""
}

func (p PathCheck) Detect(ctx *Context) (Status, string) {
	if ctx.OS == "windows" {
		cur := winUserPath()
		curLower := strings.ToLower(cur)
		for _, d := range p.dirs(ctx) {
			if !strings.Contains(curLower, strings.ToLower(d)) {
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
		curUser := winUserPath()
		curLower := strings.ToLower(curUser)

		// 只追加尚未存在的目录
		toAdd := []string{}
		for _, d := range p.dirs(ctx) {
			if !strings.Contains(curLower, strings.ToLower(d)) {
				toAdd = append(toAdd, d)
			}
		}
		if len(toAdd) == 0 {
			return nil
		}

		newPath := curUser
		if newPath != "" && !strings.HasSuffix(newPath, ";") {
			newPath += ";"
		}
		newPath += strings.Join(toAdd, ";")

		// 用 PowerShell SetEnvironmentVariable 写入用户 PATH：
		// 1. 直接操作 HKCU\Environment，无 1024 字符截断
		// 2. 自动广播 WM_SETTINGCHANGE，Explorer 收到后新开的终端立即继承新 PATH
		script := fmt.Sprintf(
			`[System.Environment]::SetEnvironmentVariable('Path', '%s', 'User')`,
			strings.ReplaceAll(newPath, "'", "''"),
		)
		out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
		if err != nil {
			return fmt.Errorf("写入 PATH 失败: %v\n%s", err, out)
		}
		// 同步更新当前进程的 PATH，让安装器进程衍生的子进程（新终端）直接继承新 PATH
		cur := os.Getenv("PATH")
		if cur != "" && !strings.HasSuffix(cur, ";") {
			cur += ";"
		}
		os.Setenv("PATH", cur+strings.Join(toAdd, ";"))
		return nil
	}

	rc := p.shellRC(ctx)
	if data, _ := os.ReadFile(rc); strings.Contains(string(data), pathBegin) {
		return nil
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
		// 注册表已写入；新 PATH 需重开终端生效，这里不阻塞校验
		return nil
	}
	if st, msg := p.Detect(ctx); st != StatusOK {
		return errors.New(msg)
	}
	return nil
}
