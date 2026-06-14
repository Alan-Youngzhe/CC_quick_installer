package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

//go:embed static
var staticFS embed.FS

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   Claude Code Quick Installer            ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n服务已启动 → %s\n", url)
	fmt.Println("正在打开浏览器，稍候...")
	fmt.Println("按 Ctrl+C 退出\n")

	// 略等一拍再开浏览器，让 server 先就绪
	go func() {
		time.Sleep(200 * time.Millisecond)
		exec.Command("open", url).Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		fmt.Println("\n已退出。")
		os.Exit(0)
	}()

	if err := http.Serve(ln, newMux()); err != nil {
		fmt.Fprintln(os.Stderr, "服务异常:", err)
		os.Exit(1)
	}
}

// staticFile 给 serveIndex 使用，避免 embed.FS 路径问题。
func staticFile(name string) (fs.File, error) {
	return staticFS.Open("static/" + name)
}
