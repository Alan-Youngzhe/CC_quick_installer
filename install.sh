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
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"

printf '→ 下载安装器（Universal Binary）...\n'
curl -fsSL --progress-bar "${DOWNLOAD_URL}" -o "${INSTALL_PATH}"

printf '→ 解除 Gatekeeper 隔离...\n'
xattr -dr com.apple.quarantine "${INSTALL_PATH}" 2>/dev/null || true
chmod +x "${INSTALL_PATH}"

printf '\n'
printf '\033[32m✓ 准备完成，正在启动界面...\033[0m\n'
printf '  浏览器将自动打开安装界面\n'
printf '  界面关闭后按 \033[33mCtrl+C\033[0m 退出\n'
printf '\n'

exec "${INSTALL_PATH}"
