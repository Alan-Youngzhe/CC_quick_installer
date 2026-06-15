#!/usr/bin/env bash
set -euo pipefail

REPO="Alan-Youngzhe/CC_quick_installer"
BINARY_NAME="CCQuickInstaller-mac"
INSTALL_PATH="/tmp/CCQuickInstaller-mac"

printf '\n'
printf '  \033[32m ██████╗  ██████╗ ██████╗ ███████╗      ███████╗ ██████╗ ██████╗  ██████╗ ███████╗\033[0m\n'
printf '  \033[32m██╔════╝ ██╔═══██╗██╔══██╗██╔════╝      ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔════╝\033[0m\n'
printf '  \033[32m██║      ██║   ██║██║  ██║█████╗   ─── █████╗  ██║   ██║██████╔╝██║  ███╗█████╗\033[0m\n'
printf '  \033[32m██║      ██║   ██║██║  ██║██╔══╝   ─── ██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝\033[0m\n'
printf '  \033[32m╚██████╗ ╚██████╔╝██████╔╝███████╗     ██║     ╚██████╔╝██║  ██╗╚██████╔╝███████╗\033[0m\n'
printf '  \033[32m ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝     ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝\033[0m\n'
printf '\n'
printf '  \033[33m▌ QUICK INSTALLER\033[0m\n'
printf '  \033[2m// CODE-FORGE · 一键安装 Claude Code · 用户态零依赖 · 国内直连\033[0m\n'
printf '\n'

# 系统检查
if [[ "$(uname)" != "Darwin" ]]; then
  printf '\033[31m✗ 此脚本仅支持 macOS\033[0m\n'
  exit 1
fi

printf '→ 获取最新版本...\n'
GITHUB_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"

# 安装器二进制托管在 GitHub Releases，而 release-assets.githubusercontent.com 在国内被墙，
# 直连会 curl:(7) 连接失败。这里优先走 GitHub 国内加速镜像，逐个尝试，最后才直连。
# （npmmirror 国内源只负责安装器运行后下载 Node/Claude，管不到"下载安装器自身"这一步。）
MIRRORS=(
  "https://ghfast.top/"
  "https://gh-proxy.com/"
  ""
)

downloaded=0
for m in "${MIRRORS[@]}"; do
  label="${m:-直连 GitHub}"
  printf '→ 下载安装器（Universal Binary）· %s\n' "${label}"
  if curl -fsSL --progress-bar "${m}${GITHUB_URL}" -o "${INSTALL_PATH}"; then
    downloaded=1
    break
  fi
  printf '\033[33m  此源失败，尝试下一个...\033[0m\n'
done

if [[ "${downloaded}" != "1" ]]; then
  printf '\033[31m✗ 所有下载源均连接失败，请检查网络后重试，或手动下载安装器\033[0m\n'
  exit 1
fi

printf '→ 解除 Gatekeeper 隔离...\n'
xattr -dr com.apple.quarantine "${INSTALL_PATH}" 2>/dev/null || true
chmod +x "${INSTALL_PATH}"

printf '\n'
printf '\033[32m✓ 准备完成，正在启动界面...\033[0m\n'
printf '  浏览器将自动打开安装界面\n'
printf '  界面关闭后按 \033[33mCtrl+C\033[0m 退出\n'
printf '\n'

exec "${INSTALL_PATH}"
