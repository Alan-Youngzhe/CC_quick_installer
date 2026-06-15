package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// SettingsPath 返回 Claude Code 的配置文件路径。
func SettingsPath(home string) string {
	return filepath.Join(home, ".claude", "settings.json")
}

// ImportSettings 校验内容为合法 JSON 后,整份写入 ~/.claude/settings.json。
// 用户从别处(团队分发、文档)复制一份 settings.json 直接粘贴导入即可,无需逐项填写。
// Windows 上自动剔除 hooks 字段：hooks 通常调用 /bin/sh，在 Windows 上无法运行。
func ImportSettings(home, content string) error {
	var probe map[string]any
	if err := json.Unmarshal([]byte(content), &probe); err != nil {
		return fmt.Errorf("不是合法的 JSON:%v", err)
	}
	if runtime.GOOS == "windows" {
		delete(probe, "hooks")
	}
	p := SettingsPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(probe, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, out, 0o600) // 0600:含 Key,仅本人可读
}

// RemoveSettings 删除 ~/.claude/settings.json(想换配置时先移除再导入)。
func RemoveSettings(home string) error {
	err := os.Remove(SettingsPath(home))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ReadSettings 读取当前配置内容;不存在返回 false。
func ReadSettings(home string) (string, bool) {
	data, err := os.ReadFile(SettingsPath(home))
	if err != nil {
		return "", false
	}
	return string(data), true
}
