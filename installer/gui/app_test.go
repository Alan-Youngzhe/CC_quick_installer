package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBindingSmoke 冒烟测试 GUI 绑定层:在沙箱 HOME 下走一遍前端会触发的调用,
// 确认绑定正确转发到 engine,且写入只落在沙箱目录(绝不碰真实 ~/.claude)。
func TestBindingSmoke(t *testing.T) {
	sb := t.TempDir()
	t.Setenv("HOME", sb)        // 引擎用 os.UserHomeDir(),沙箱化全部写入
	t.Setenv("USERPROFILE", sb) // Windows 兜底

	app := NewApp()
	app.startup(nil) // nil ctx:无窗口,事件发送被跳过
	if app.ec == nil {
		t.Fatal("startup 后引擎上下文为空")
	}

	// 1. 初始未配置。
	if c := app.GetConfig(); c.Configured {
		t.Error("全新沙箱不应已配置")
	}

	// 2. 导入一份 settings.json → 落在沙箱内。
	cfg := `{"env":{"ANTHROPIC_BASE_URL":"https://api.moonshot.cn/anthropic","ANTHROPIC_AUTH_TOKEN":"sk-secret-1"}}`
	if err := app.ImportConfig(cfg); err != nil {
		t.Fatal("ImportConfig:", err)
	}
	data, err := os.ReadFile(filepath.Join(sb, ".claude", "settings.json"))
	if err != nil {
		t.Fatal("读取沙箱 settings.json:", err)
	}
	if !strings.Contains(string(data), "sk-secret-1") {
		t.Errorf("settings.json 未正确写入: %s", data)
	}
	if c := app.GetConfig(); !c.Configured {
		t.Error("导入后应为已配置")
	}

	// 3. 非法 JSON 应被拒绝。
	if err := app.ImportConfig("{not json"); err == nil {
		t.Error("非法 JSON 应导入失败")
	}

	// 4. 移除后回到未配置。
	if err := app.RemoveConfig(); err != nil {
		t.Fatal("RemoveConfig:", err)
	}
	if c := app.GetConfig(); c.Configured {
		t.Error("移除后不应再是已配置")
	}
}
