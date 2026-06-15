package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// NpmPrefixCheck 把 npm 全局目录指到用户目录,从根上消除
// `npm i -g` 的 EACCES / 需要 sudo 问题。同时设置国内 registry。
type NpmPrefixCheck struct{}

func (NpmPrefixCheck) ID() string       { return "npm-prefix" }
func (NpmPrefixCheck) Name() string     { return "npm 全局目录(免 sudo)" }
func (NpmPrefixCheck) NeedsAdmin() bool { return false }

func (n NpmPrefixCheck) npmrc(ctx *Context) string {
	return filepath.Join(ctx.Home, ".npmrc")
}

func (n NpmPrefixCheck) Detect(ctx *Context) (Status, string) {
	data, err := os.ReadFile(n.npmrc(ctx))
	if err == nil &&
		strings.Contains(string(data), "prefix=") &&
		strings.Contains(string(data), ctx.NpmGlobal) {
		return StatusOK, "npm 全局目录已指向用户态"
	}
	return StatusFixable, "npm prefix 未配置(可能触发 sudo / EACCES)"
}

func (n NpmPrefixCheck) Fix(ctx *Context) error {
	if err := os.MkdirAll(ctx.NpmGlobal, 0o755); err != nil {
		return err
	}
	existing, _ := os.ReadFile(n.npmrc(ctx))
	// 如果已正确指向目标目录，无需修改
	if strings.Contains(string(existing), "prefix="+ctx.NpmGlobal) {
		return nil
	}
	// 过滤掉旧的 prefix 行（路径可能已变），重写为新路径
	var kept []string
	for _, line := range strings.Split(string(existing), "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "prefix=") {
			kept = append(kept, line)
		}
	}
	kept = append(kept, "prefix="+ctx.NpmGlobal, "registry=https://registry.npmmirror.com/")
	return os.WriteFile(n.npmrc(ctx), []byte(strings.Join(kept, "\n")+"\n"), 0o644)
}

func (n NpmPrefixCheck) Verify(ctx *Context) error {
	if st, msg := n.Detect(ctx); st != StatusOK {
		return errors.New(msg)
	}
	return nil
}
