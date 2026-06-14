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
	if strings.Contains(string(existing), "prefix=") {
		return nil // 已有 prefix,尊重用户设置,不覆盖
	}
	line := "prefix=" + ctx.NpmGlobal + "\nregistry=https://registry.npmmirror.com/\n"
	f, err := os.OpenFile(n.npmrc(ctx), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func (n NpmPrefixCheck) Verify(ctx *Context) error {
	if st, msg := n.Detect(ctx); st != StatusOK {
		return errors.New(msg)
	}
	return nil
}
