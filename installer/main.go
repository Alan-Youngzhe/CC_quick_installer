package main

import (
	"fmt"
	"os"

	"claude-toolbox-installer/engine"
)

func main() {
	ctx, err := engine.NewContext()
	if err != nil {
		fmt.Println("初始化失败:", err)
		os.Exit(1)
	}

	// 子命令:配置导入/移除。GUI 调用的是同一套 engine 逻辑。
	if len(os.Args) > 1 {
		if handleConfigCmd(ctx, os.Args[1:]) {
			return
		}
	}

	runDoctor(ctx)
}

// runDoctor 执行体检并自动修复。
func runDoctor(ctx *engine.Context) {
	fmt.Println("== Claude Code 一键安装器 / 环境医生 ==")
	fmt.Printf("系统: %s/%s   用户目录: %s\n", ctx.OS, ctx.Arch, ctx.Home)
	if _, ok := engine.ReadSettings(ctx.Home); ok {
		fmt.Println("配置: 已导入 settings.json")
	} else {
		fmt.Println("配置: 尚未导入 settings.json(可在 GUI 导入,或 doctor import <文件>)")
	}
	fmt.Println("开始体检并自动修复(全程用户态,无需管理员)...")
	fmt.Println()

	doc := &engine.Doctor{Checks: engine.DefaultChecks()}
	results := doc.Run(ctx)

	failed := 0
	for _, r := range results {
		if r.Status == engine.StatusFailed {
			failed++
		}
	}
	fmt.Println()
	if failed == 0 {
		fmt.Println("✅ 全部就绪。请打开一个【新】终端,输入  claude  即可开始 vibe coding。")
		return
	}
	fmt.Printf("⚠️  有 %d 项未完成。\n", failed)
	os.Exit(1)
}

// handleConfigCmd 处理配置子命令,返回是否已处理。
//
//	doctor import <settings.json 路径>   导入配置
//	doctor remove                        移除当前配置
func handleConfigCmd(ctx *engine.Context, args []string) bool {
	switch args[0] {
	case "import":
		if len(args) < 2 {
			fmt.Println("用法: doctor import <settings.json 路径>")
			return true
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Println("读取失败:", err)
			return true
		}
		if err := engine.ImportSettings(ctx.Home, string(data)); err != nil {
			fmt.Println("导入失败:", err)
			return true
		}
		fmt.Println("已导入到", engine.SettingsPath(ctx.Home))
		return true

	case "remove":
		if err := engine.RemoveSettings(ctx.Home); err != nil {
			fmt.Println("移除失败:", err)
			return true
		}
		fmt.Println("已移除配置。")
		return true
	}
	return false
}
