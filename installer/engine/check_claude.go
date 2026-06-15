package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// ClaudeCheck 安装 Claude Code 原生二进制(无需 Node)。
// 走 npmmirror 的官方平台包 @anthropic-ai/claude-code-<平台>:取 latest →
// 下 tarball(gzip,约官方裸二进制的 1/3、国内更快)→ npm integrity 校验 → 解出二进制。
// 二进制与官方逐字节相同(SHA-256 一致),且全程只需 HTTP + 解压,无需 npm/Node。
type ClaudeCheck struct{}

// npmPackageMeta 是 npm registry 包元数据的精简结构,只取版本与 tarball 完整性。
type npmPackageMeta struct {
	DistTags map[string]string `json:"dist-tags"`
	Versions map[string]struct {
		Dist struct {
			Tarball   string `json:"tarball"`
			Integrity string `json:"integrity"`
		} `json:"dist"`
	} `json:"versions"`
}

var verRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+`)

func (ClaudeCheck) ID() string       { return "claude-binary" }
func (ClaudeCheck) Name() string     { return "Claude Code 原生二进制" }
func (ClaudeCheck) NeedsAdmin() bool { return false }

func (c ClaudeCheck) Detect(ctx *Context) (Status, string) {
	bin := filepath.Join(ctx.LocalBin, ctx.claudeBin())
	if _, err := os.Stat(bin); err == nil {
		return StatusOK, "已安装于 " + bin
	}
	// 在 PATH 里找到了但不在预期目录，触发重装迁移到新路径。
	// 这确保 claude.exe 真正落在 LocalBin，PATH 注册后 CMD/PowerShell 都能找到。
	if p, err := exec.LookPath("claude"); err == nil {
		return StatusFixable, "已在 " + p + "，迁移至 " + bin
	}
	return StatusFixable, "未检测到 claude"
}

func (c ClaudeCheck) Fix(ctx *Context) error {
	plat := ctx.claudePlatform() // darwin-arm64 / win32-x64 …… 与 npm 平台包后缀一致
	pkg := "@anthropic-ai/claude-code-" + plat

	// 1. 取平台包元数据(latest 版本 + tarball 地址 + integrity)。
	metaRaw, err := httpGetString(ctx.NpmRegistry + "/" + pkg)
	if err != nil {
		return fmt.Errorf("获取平台包信息失败: %v", err)
	}
	var meta npmPackageMeta
	if err := json.Unmarshal([]byte(metaRaw), &meta); err != nil {
		return fmt.Errorf("解析平台包信息失败: %v", err)
	}
	ver := meta.DistTags["latest"]
	if !verRe.MatchString(ver) {
		return fmt.Errorf("最新版本号异常: %q", ver)
	}
	dist := meta.Versions[ver].Dist
	if dist.Tarball == "" {
		return fmt.Errorf("平台包 %s@%s 缺少 tarball", pkg, ver)
	}

	// 2. 下载 tarball(gzip,断点续传)。
	tgz := filepath.Join(os.TempDir(), "claude-"+plat+"-"+ver+".tgz")
	if err := Download(dist.Tarball, tgz); err != nil {
		return err
	}

	// 3. npm integrity 校验(防半截/被篡改),不过则删包。
	if dist.Integrity != "" {
		if err := VerifyIntegrity(tgz, dist.Integrity); err != nil {
			os.Remove(tgz)
			return err
		}
	}

	// 4. 从包内取出 package/claude[.exe] → 目标目录,删掉压缩包。
	dest := filepath.Join(ctx.LocalBin, ctx.claudeBin())
	if err := untarGzExtractFile(tgz, ctx.claudeBin(), dest); err != nil {
		return err
	}
	os.Remove(tgz)

	if ctx.OS != "windows" {
		return os.Chmod(dest, 0o755)
	}
	return nil
}

func (c ClaudeCheck) Verify(ctx *Context) error {
	bin := filepath.Join(ctx.LocalBin, ctx.claudeBin())
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}
