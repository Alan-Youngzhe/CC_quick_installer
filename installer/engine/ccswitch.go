package engine

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// cc-switch 是可选的「服务商配置切换」桌面应用(farion1231/cc-switch)。
// 它是独立 GUI,不进 PATH、无人依赖其路径,因此安装位置可由用户自选。
// 版本钉死,升级时只改这一行(beta 阶段不做"解析最新版",少一个失败点)。
const CCSwitchVersion = "3.16.3"

// ccSwitchMirrors 是 GitHub release 的国内加速前缀,按序尝试,前一个失败自动换下一个。
// 用法:把前缀直接拼在原始 https://github.com/... 链接前。末尾空串 = 直连(有梯子时兜底)。
var ccSwitchMirrors = []string{
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://ghproxy.net/",
	"",
}

// ccSwitchAsset 返回当前平台的 release 资产名,以及解压后可执行体相对 destDir 的路径。
// Windows 用便携版 zip(解压即用、免安装免提权);macOS 用含 .app 的 zip(已签名公证)。
func ccSwitchAsset(ctx *Context) (asset, exeRel string, ok bool) {
	switch ctx.OS {
	case "windows":
		return fmt.Sprintf("CC-Switch-v%s-Windows-Portable.zip", CCSwitchVersion), "cc-switch.exe", true
	case "darwin":
		return fmt.Sprintf("CC-Switch-v%s-macOS.zip", CCSwitchVersion), "CC Switch.app", true
	}
	return "", "", false
}

// DefaultCCSwitchDir 返回 cc-switch 的默认安装目录(用户可在界面改成任意位置)。
// macOS 放 ~/Applications(出现在「访达 > 应用程序」与启动台);Windows 放 ~/CC-Switch。
func DefaultCCSwitchDir(ctx *Context) string {
	if ctx.OS == "darwin" {
		return filepath.Join(ctx.Home, "Applications")
	}
	return filepath.Join(ctx.Home, "CC-Switch")
}

// InstallCCSwitch 下载并解压 cc-switch 到 destDir(空则用默认目录),返回最终可执行体路径。
// 走国内加速镜像(带断点续传与多镜像兜底),复用现有 Download。
func InstallCCSwitch(ctx *Context, destDir string, log func(string)) (string, error) {
	asset, exeRel, ok := ccSwitchAsset(ctx)
	if !ok {
		return "", fmt.Errorf("cc-switch 暂不支持当前平台 %s", ctx.OS)
	}
	if destDir == "" {
		destDir = DefaultCCSwitchDir(ctx)
	}
	logf := func(format string, a ...any) {
		if log != nil {
			log(fmt.Sprintf(format, a...))
		}
	}

	rawURL := fmt.Sprintf(
		"https://github.com/farion1231/cc-switch/releases/download/v%s/%s",
		CCSwitchVersion, asset)
	tmp := filepath.Join(os.TempDir(), asset)

	var lastErr error
	for _, m := range ccSwitchMirrors {
		label := m
		if label == "" {
			label = "直连 GitHub"
		}
		logf("  [下载] cc-switch %s — %s\n", CCSwitchVersion, label)
		if err := Download(m+rawURL, tmp); err != nil {
			lastErr = err
			os.Remove(tmp + ".part") // 换镜像前清掉半截续传文件,避免污染下一次
			logf("  [重试] %s 失败,切换下一个镜像\n", label)
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return "", fmt.Errorf("所有镜像均下载失败: %v", lastErr)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	if err := unzipInto(tmp, destDir); err != nil {
		return "", err
	}
	os.Remove(tmp)

	exe := filepath.Join(destDir, exeRel)
	logf("  [完成] cc-switch 已安装到 %s\n", exe)
	return exe, nil
}

// unzipInto 解压 zip 到 dst,保留完整目录结构与权限。
// 区别于 check_node.go 的 unzipFlatten(会剥掉顶层目录):cc-switch 的 mac 包顶层就是
// "CC Switch.app",必须原样保留;win 便携版顶层无目录(exe 直接在根),也需原样写出。
func unzipInto(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	for _, zf := range r.File {
		target := filepath.Join(dst, zf.Name)
		// 防 zip-slip:解压后路径必须仍在 dst 之内。
		if abs, err := filepath.Abs(target); err != nil || !strings.HasPrefix(abs, dstAbs) {
			return fmt.Errorf("非法压缩包路径: %s", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, zf.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		w.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
