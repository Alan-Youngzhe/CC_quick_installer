package engine

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Download 从镜像 url 下载到 dest,支持断点续传(HTTP Range)。
// 弱网现场反复中断后再次运行可从上次进度继续。
func Download(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"

	var existing int64
	if fi, err := os.Stat(part); err == nil {
		existing = fi.Size()
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("下载失败 HTTP %d: %s", resp.StatusCode, url)
	}

	flag := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flag, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, dest)
}

// httpGetString 拉取一个小文本端点(如版本号、manifest),返回去空白后的内容。
func httpGetString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("请求失败 HTTP %d: %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB 上限,防异常大响应
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// VerifyIntegrity 校验文件是否匹配 npm 的 dist.integrity(形如 "sha512-<base64>")。
// 这是 npm 官方的包完整性机制,防止半截下载/被篡改。
func VerifyIntegrity(path, integrity string) error {
	algo, want, ok := strings.Cut(integrity, "-")
	if !ok {
		return fmt.Errorf("integrity 格式异常: %q", integrity)
	}
	var h hash.Hash
	switch algo {
	case "sha512":
		h = sha512.New()
	case "sha256":
		h = sha256.New()
	default:
		return fmt.Errorf("不支持的 integrity 算法: %s", algo)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := base64.StdEncoding.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("%s 校验失败: 期望 %s 实得 %s", algo, want, got)
	}
	return nil
}

// Sha256File 计算文件 SHA-256,用于校验负载完整性(防止半截下载/被篡改)。
func Sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
