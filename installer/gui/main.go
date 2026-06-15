package main

import (
	"embed"
	"errors"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var errEngine = errors.New("引擎未就绪")

// Version 由 CI 通过 -ldflags "-X main.Version=vX.Y.Z" 注入；本地构建显示 dev。
var Version = "dev"

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "CODE-FORGE 一键安装器 " + Version,
		Width:            760,
		Height:           620,
		MinWidth:         640,
		MinHeight:        520,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 17, G: 18, B: 23, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
