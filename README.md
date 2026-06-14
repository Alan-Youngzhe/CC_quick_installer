# CODE-FORGE · Claude Code Quick Installer

> 一键安装 Claude Code — 用户态、零依赖、国内直连，无需管理员权限、无需科学上网。

---

## 快速开始

### macOS（一行命令）

打开终端，粘贴以下命令：

```bash
curl -fsSL https://raw.githubusercontent.com/Alan-Youngzhe/CC_quick_installer/main/install.sh | bash
```

脚本自动下载安装器、解除 Gatekeeper 隔离、打开浏览器界面，全程无需手动操作。

### Windows（双击运行）

下载 [Releases](https://github.com/Alan-Youngzhe/CC_quick_installer/releases/latest) 中的 `CCQuickInstaller-win-x64.exe`，双击打开图形界面。

---

## 使用流程

1. 点击 **RUN INSTALL** — 自动检测并安装 Node.js、Claude Code，修复 PATH
2. 在 **import_config** 面板粘贴你的 `settings.json`，配置模型与 API Key
3. 打开一个**新终端**，输入 `claude` 开始使用

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

---

## 功能特性

- **全程用户态**：所有写入落在 `~/.local`、`~/.npm-global`、`~/.claude`，不调用 `sudo`
- **国内直连**：Node.js 走 npmmirror，Claude Code 原生二进制走 npmmirror npm 平台包，体积小 3×，速度快 4×
- **幂等**：重复运行自动跳过已就绪项，中途中断再跑也能续完
- **CLI + GUI 两用**：macOS 浏览器界面，Windows 原生 GUI，视觉完全一致

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
├── webui/               # macOS 浏览器 UI（HTTP server + SSE）
│   ├── main.go
│   ├── server.go
│   └── static/index.html
└── gui/                 # Windows Wails GUI
    ├── main.go
    ├── app.go
    └── frontend/dist/index.html
```

---

## 构建

**macOS（curl 脚本用的二进制）：**

```bash
cd installer
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/CCQuickInstaller-arm64 ./webui/
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/CCQuickInstaller-amd64 ./webui/
lipo -create -output dist/CCQuickInstaller-mac dist/CCQuickInstaller-arm64 dist/CCQuickInstaller-amd64
```

**Windows GUI（需要 [Wails](https://wails.io/)，在 Windows 机器上执行）：**

```bash
cd installer/gui
wails build -platform windows/amd64 -webview2 embed
```

---

## 换镜像源

镜像地址集中在 `engine/env.go`：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NodeMirror` | `cdn.npmmirror.com/binaries/node` | Node.js 下载源 |
| `NpmRegistry` | `registry.npmmirror.com` | Claude Code 平台包注册表 |

---

## License

MIT
