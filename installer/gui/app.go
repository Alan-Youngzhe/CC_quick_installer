package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"

	"claude-toolbox-installer/engine"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定层:只负责把前端调用转给现有 engine,不实现任何业务逻辑。
type App struct {
	ctx context.Context
	ec  *engine.Context // 引擎上下文(平台/路径),启动时构造一次
}

func NewApp() *App { return &App{} }

// startup 在窗口创建后调用:构造引擎上下文。失败也不 panic,留给前端展示。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if ec, err := engine.NewContext(); err == nil {
		a.ec = ec
	} else {
		wruntime.LogError(ctx, "引擎初始化失败: "+err.Error())
	}
}

// --- 配置导入/移除(像梯子导入订阅:直接粘贴一份 settings.json) ---

// ConfigView 给前端展示当前配置状态。
type ConfigView struct {
	Configured bool   `json:"configured"`
	Content    string `json:"content"`
}

// GetConfig 返回当前 ~/.claude/settings.json 内容与是否已配置。
func (a *App) GetConfig() ConfigView {
	if a.ec == nil {
		return ConfigView{}
	}
	content, ok := engine.ReadSettings(a.ec.Home)
	return ConfigView{Configured: ok, Content: content}
}

// ImportConfig 把用户粘贴的 settings.json 导入(校验 JSON 合法后写入)。
func (a *App) ImportConfig(content string) error {
	if a.ec == nil {
		return errEngine
	}
	return engine.ImportSettings(a.ec.Home, content)
}

// RemoveConfig 移除当前配置(想换配置时先移除再导入)。
func (a *App) RemoveConfig() error {
	if a.ec == nil {
		return errEngine
	}
	return engine.RemoveSettings(a.ec.Home)
}

// --- 一键安装(直接复用 engine.Doctor + DefaultChecks) ---

// InstallResult 是一次体检的汇总结果。
type InstallResult struct {
	Failed int  `json:"failed"`
	OK     bool `json:"ok"`
}

// RunInstall 执行「一键安装」。逐行进度通过 "doctor:log" 事件实时推给前端。
func (a *App) RunInstall() InstallResult {
	if a.ec == nil {
		return InstallResult{Failed: 1, OK: false}
	}
	doc := &engine.Doctor{
		Checks: engine.DefaultChecks(),
		Log: func(line string) {
			if a.ctx != nil { // 无窗口上下文(如单测)时跳过事件,不影响检测逻辑
				wruntime.EventsEmit(a.ctx, "doctor:log", line)
			}
		},
	}
	results := doc.Run(a.ec)
	failed := 0
	for _, r := range results {
		if r.Status == engine.StatusFailed {
			failed++
		}
	}
	// 安装成功后在 Windows 上自动弹出一个新 CMD 窗口。
	// 不依赖 Fix() 是否运行过 os.Setenv（PATH 检查"已就绪"时 Fix 会跳过），
	// 在这里强制把工具目录写入当前进程 PATH，确保弹出的新 CMD 一定能找到 claude。
	if failed == 0 && runtime.GOOS == "windows" {
		toolDirs := a.ec.LocalBin + ";" + a.ec.NodeDir + ";" + a.ec.NpmGlobal
		existing := os.Getenv("PATH")
		os.Setenv("PATH", toolDirs+";"+existing)
		exec.Command("cmd", "/c", "start", "cmd", "/k",
			`echo.&echo   ================================&echo   Claude 已就绪！输入 claude 开始使用&echo   ================================&echo.`).Start()
	}
	return InstallResult{Failed: failed, OK: failed == 0}
}

// SystemInfo 返回平台/路径概览,GUI 顶部展示。
func (a *App) SystemInfo() map[string]string {
	if a.ec == nil {
		return map[string]string{"error": "engine not ready"}
	}
	return map[string]string{
		"os":   a.ec.OS,
		"arch": a.ec.Arch,
		"home": a.ec.Home,
	}
}
