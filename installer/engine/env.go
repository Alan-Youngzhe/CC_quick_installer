package engine

import (
	"os"
	"path/filepath"
	"runtime"
)

// Context 贯穿整个安装流程,保存平台信息、目标路径、下载源。
// 所有路径都落在「用户可写目录」内,因此整套流程零管理员权限。
type Context struct {
	OS        string // darwin / windows / linux
	Arch      string // amd64 / arm64
	Home      string
	LocalBin  string // claude 等二进制安装目录
	NodeDir   string // 用户态 Node.js 解压目录
	NpmGlobal string // npm 全局包目录(免 sudo)

	// 下载源:默认指向真实、国内可达的镜像。上线时可整体替换为你自建 CDN。
	NodeMirror  string // Node.js 包根:{base}/v<版本>/<asset>
	NpmRegistry string // npm registry 根:Claude 原生二进制走 @anthropic-ai/claude-code-<平台> 平台包(gzip,比官方裸二进制小 3 倍、国内快)
}

// NewContext 探测当前机器,构造上下文。
func NewContext() (*Context, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	c := &Context{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Home: home,
		// npmmirror 的 Node 镜像(国内可达);cdn 子域是其 302 后的最终地址,直连省一跳。
		NodeMirror: "https://cdn.npmmirror.com/binaries/node",
		// npmmirror 的 npm registry(国内可达)。Claude 二进制即官方平台包的镜像,SHA-256 与官方一致。
		NpmRegistry: "https://registry.npmmirror.com",
	}
	if c.OS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		root := filepath.Join(base, "Programs", "claude-toolbox")
		c.LocalBin = filepath.Join(root, "bin")
		c.NodeDir = filepath.Join(root, "node")
		c.NpmGlobal = filepath.Join(root, "npm-global")
	} else {
		c.LocalBin = filepath.Join(home, ".local", "bin")
		c.NodeDir = filepath.Join(home, ".local", "node")
		c.NpmGlobal = filepath.Join(home, ".npm-global")
	}
	return c, nil
}

// claudeBin 返回当前平台的 claude 可执行文件名。
func (c *Context) claudeBin() string {
	if c.OS == "windows" {
		return "claude.exe"
	}
	return "claude"
}

// pkgArch 返回 Node / Claude 包命名用的架构名(Go 的 amd64 → 上游的 x64)。
func (c *Context) pkgArch() string {
	if c.Arch == "amd64" {
		return "x64"
	}
	return c.Arch
}

// claudePlatform 返回 Claude manifest/下载用的平台名,如 darwin-arm64 / win32-x64。
func (c *Context) claudePlatform() string {
	osName := c.OS
	if osName == "windows" {
		osName = "win32"
	}
	return osName + "-" + c.pkgArch()
}
