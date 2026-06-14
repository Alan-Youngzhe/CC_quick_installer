package engine

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// NodeCheck 在「用户态」安装 Node.js(免 sudo / 免管理员)。
// 不走 Homebrew(在 Intel Mac 上可能要权限、且慢),直接从国内镜像拉官方包解压到用户目录。
type NodeCheck struct {
	// Version 为空 = 运行时自动解析 npmmirror 上的最新 LTS;填具体版本(如 "24.16.0")则钉死。
	Version string
}

// fallbackNodeLTS 是解析最新 LTS 失败时的兜底版本(验证过、三平台齐全)。
const fallbackNodeLTS = "24.16.0"

func (NodeCheck) ID() string       { return "node-runtime" }
func (NodeCheck) Name() string     { return "Node.js 运行时(用户态)" }
func (NodeCheck) NeedsAdmin() bool { return false }

func (n NodeCheck) localNode(ctx *Context) string {
	if ctx.OS == "windows" {
		return filepath.Join(ctx.NodeDir, "node.exe")
	}
	return filepath.Join(ctx.NodeDir, "bin", "node")
}

func (n NodeCheck) Detect(ctx *Context) (Status, string) {
	for _, c := range []string{n.localNode(ctx), "node"} {
		out, err := exec.Command(c, "--version").Output()
		if err == nil && nodeMajor(strings.TrimSpace(string(out))) >= 18 {
			return StatusOK, "已安装 " + strings.TrimSpace(string(out))
		}
	}
	return StatusFixable, "未检测到 Node >= 18"
}

func (n NodeCheck) Fix(ctx *Context) error {
	ver := n.Version
	if ver == "" {
		ver = resolveNodeLTS(ctx) // 自动跟随最新 LTS,失败回退兜底版本
	}
	arch := ctx.pkgArch() // Node 官方包用 x64 命名
	var asset string
	if ctx.OS == "windows" {
		asset = fmt.Sprintf("node-v%s-win-%s.zip", ver, arch)
	} else {
		asset = fmt.Sprintf("node-v%s-%s-%s.tar.gz", ver, ctx.OS, arch)
	}
	// npmmirror 结构: {NodeMirror}/v<版本>/<asset>
	url := ctx.NodeMirror + "/v" + ver + "/" + asset
	tmp := filepath.Join(os.TempDir(), asset)
	if err := Download(url, tmp); err != nil {
		return err
	}
	if err := os.MkdirAll(ctx.NodeDir, 0o755); err != nil {
		return err
	}
	if ctx.OS == "windows" {
		return unzipFlatten(tmp, ctx.NodeDir)
	}
	return untarGzFlatten(tmp, ctx.NodeDir)
}

func (n NodeCheck) Verify(ctx *Context) error {
	out, err := exec.Command(n.localNode(ctx), "--version").Output()
	if err != nil {
		return fmt.Errorf("node 校验失败: %v", err)
	}
	if nodeMajor(strings.TrimSpace(string(out))) < 18 {
		return errors.New("node 版本过低")
	}
	return nil
}

// nodeMajor 从 "v20.15.0" 解析主版本号。
func nodeMajor(v string) int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 2)
	n, _ := strconv.Atoi(parts[0])
	return n
}

// resolveNodeLTS 从 npmmirror 的 index.json 取最新 LTS 版本(返回不带 v 前缀,如 "24.16.0")。
// index.json 按版本倒序,第一个 lts 非 false 的即最新 LTS;任何异常都回退 fallbackNodeLTS,
// 保证镜像抽风/字段变更时仍能装上一个验证过的版本。
func resolveNodeLTS(ctx *Context) string {
	raw, err := httpGetString(ctx.NodeMirror + "/index.json")
	if err != nil {
		return fallbackNodeLTS
	}
	var items []struct {
		Version string          `json:"version"` // "v24.16.0"
		LTS     json.RawMessage `json:"lts"`     // 字符串(代号)或 false
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return fallbackNodeLTS
	}
	for _, it := range items {
		if len(it.LTS) > 0 && string(it.LTS) != "false" { // 是 LTS
			if v := strings.TrimPrefix(it.Version, "v"); v != "" {
				return v
			}
		}
	}
	return fallbackNodeLTS
}

// untarGzExtractFile 从 tar.gz 中只取出顶层目录下指定相对路径的单个文件,写到 dst。
// 用于 npm 平台包(package/claude)只提取那个二进制,忽略 package.json/LICENSE 等。
func untarGzExtractFile(src, want, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if stripTop(hdr.Name) != want {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		w, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, tr)
		w.Close()
		return err
	}
	return fmt.Errorf("包内未找到 %s", want)
}

// untarGzFlatten 解压 tar.gz 并剥掉顶层目录(node-vX.../bin/node → bin/node)。
func untarGzFlatten(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rel := stripTop(hdr.Name)
		if rel == "" {
			continue
		}
		target := filepath.Join(dst, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr); err != nil {
				w.Close()
				return err
			}
			w.Close()
		case tar.TypeSymlink:
			_ = os.Symlink(hdr.Linkname, target)
		}
	}
	return nil
}

// unzipFlatten 解压 zip 并剥掉顶层目录。
func unzipFlatten(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, zf := range r.File {
		rel := stripTop(zf.Name)
		if rel == "" {
			continue
		}
		target := filepath.Join(dst, rel)
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

// stripTop 去掉路径的第一段(顶层目录)。
func stripTop(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	idx := strings.Index(name, "/")
	if idx < 0 {
		return ""
	}
	return name[idx+1:]
}
