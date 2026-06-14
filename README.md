# Claude Code Quick Installer

> 一键安装 Claude Code — 用户态、零依赖、国内直连，无需管理员权限、无需科学上网。

English description: A zero-dependency, user-space Claude Code installer written in Go. No admin rights, no VPN, works out of the box in mainland China via npmmirror.

---

## 功能

- **全程用户态**：所有写入落在 `~/.local`、`~/.npm-global`、`~/.claude`，绝不调用 `sudo`
- **国内直连**：Node.js 走 npmmirror，Claude Code 原生二进制走 `@anthropic-ai/claude-code-<平台>` npm 平台包（npmmirror 同步），体积比官方 `downloads.claude.ai` 小约 3×，国内速度快约 4×
- **幂等**：重复运行自动跳过已就绪项，中途中断再跑也能续完
- **CLI + GUI 两用**：同一套 `engine/` 逻辑，CLI 用于脚本/CI，GUI（Wails）用于普通用户
- **配置导入/移除**：粘贴一份 `settings.json`（模型、Key、Base URL）即可切换国内模型，像订阅导入一样简单

---

## 快速开始

### GUI（推荐普通用户）

下载 [Releases](https://github.com/Alan-Youngzhe/CC_quick_installer/releases) 中对应平台的包，打开后：

1. 点击 **RUN INSTALL** — 自动检测并安装 Node.js、Claude Code，修复 PATH
2. 点击 **IMPORT** — 粘贴你的 `settings.json`，配置模型与 API Key
3. 打开一个**新终端**，输入 `claude` 开始使用

### CLI（开发者 / 脚本）

```bash
cd installer
go build -o doctor
./doctor                              # 体检 + 自动修复
./doctor import ~/path/settings.json  # 导入配置到 ~/.claude/settings.json
./doctor remove                       # 移除当前配置
```

完整安装（干净 Mac）约 **47 秒**。

---

## settings.json 示例

使用国内模型（以 Kimi K2 为例）：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://api.moonshot.cn/anthropic",
    "ANTHROPIC_AUTH_TOKEN": "sk-...",
    "ANTHROPIC_MODEL": "kimi-k2-turbo-preview"
  }
}
```

在 GUI 的 **import_config** 面板粘贴后点 IMPORT，也可以直接用 CLI：

```bash
./doctor import settings.json
```

---

## 项目结构

```
installer/
├── main.go              # CLI 入口
├── engine/              # 核心逻辑（CLI 与 GUI 共用）
│   ├── doctor.go        # 检查接口 + 三段式执行
│   ├── env.go           # 平台探测 + 镜像地址配置
│   ├── mirror.go        # 下载（断点续传）+ SHA-256 校验
│   ├── config.go        # settings.json 导入/移除
│   └── check_*.go       # node / claude / npm-prefix / PATH
└── gui/                 # Wails GUI 外壳
    ├── main.go
    ├── app.go           # 绑定层
    └── frontend/dist/index.html
```

---

## 构建

**CLI：**

```bash
cd installer

# macOS
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o dist/doctor-darwin-arm64
GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o dist/doctor-darwin-x64

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/doctor-win-x64.exe
```

**GUI（需要 [Wails](https://wails.io/)）：**

```bash
cd installer/gui
wails dev        # 开发热重载
wails build      # 正式包（macOS）
wails build -platform windows/amd64 -nsis   # Windows 安装包
```

> Wails 依赖平台原生 WebView（macOS WKWebView / Windows WebView2），不支持跨平台交叉编译，需在目标平台分别构建。

---

## 换镜像源 / 自建 CDN

镜像地址集中在 `engine/env.go`：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NodeMirror` | `cdn.npmmirror.com/binaries/node` | Node.js 下载源 |
| `NpmRegistry` | `registry.npmmirror.com` | Claude Code 平台包注册表 |

保持相同 URL 结构即可整体切换到自建 CDN，无需改业务代码。

---

## 要求

- macOS 12+ 或 Windows 10/11（amd64）
- 构建 GUI 需要 Go ≥ 1.22 + Wails CLI
- **运行时无需 Node.js / npm**（安装器自行引导）

---

## License

MIT
