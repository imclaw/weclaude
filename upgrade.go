package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

const repo = "imclaw/weclaude"

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
var version = "dev"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func cmdUpgrade() {
	fmt.Printf("当前版本: %s\n", version)
	fmt.Println("正在检查最新版本...")

	release, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取版本信息失败: %v\n", err)
		os.Exit(1)
	}

	latest := release.TagName
	fmt.Printf("最新版本: %s\n", latest)

	if version != "dev" && version == latest {
		fmt.Println("已是最新版本，无需升级。")
		return
	}

	assetName := platformAssetName()
	if assetName == "" {
		fmt.Fprintln(os.Stderr, "当前平台不支持自动升级，请前往 https://github.com/"+repo+"/releases 手动下载。")
		os.Exit(1)
	}

	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		fmt.Fprintf(os.Stderr, "未找到当前平台的发布文件 %s，请手动下载。\n", assetName)
		os.Exit(1)
	}

	fmt.Printf("正在下载 %s ...\n", assetName)
	tmp, err := downloadToTemp(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "下载失败: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmp)

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前可执行文件路径失败: %v\n", err)
		os.Exit(1)
	}

	if err := replaceExecutable(self, tmp); err != nil {
		fmt.Fprintf(os.Stderr, "替换可执行文件失败: %v\n如需权限，请尝试: sudo weclaude upgrade\n", err)
		os.Exit(1)
	}

	fmt.Printf("升级成功！当前版本: %s\n", latest)
}

func fetchLatestRelease() (*githubRelease, error) {
	url := "https://api.github.com/repos/" + repo + "/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("未找到发布版本")
	}
	return &rel, nil
}

func platformAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "darwin":
		switch goarch {
		case "arm64":
			return "weclaude-darwin-arm64"
		case "amd64":
			return "weclaude-darwin-amd64"
		}
	case "linux":
		switch goarch {
		case "amd64":
			return "weclaude-linux-amd64"
		case "arm64":
			return "weclaude-linux-arm64"
		}
	case "windows":
		if goarch == "amd64" {
			return "weclaude-windows-amd64.exe"
		}
	}
	return ""
}

func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "weclaude-upgrade-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func replaceExecutable(dst, src string) error {
	if err := os.Chmod(src, 0755); err != nil {
		return err
	}
	// 先备份旧文件（Windows 不能直接覆盖运行中的文件，所以用 rename）
	backup := dst + ".bak"
	_ = os.Rename(dst, backup)

	if err := os.Rename(src, dst); err != nil {
		// rename 跨设备失败时回退到 copy
		_ = os.Rename(backup, dst)
		return copyFile(src, dst)
	}
	_ = os.Remove(backup)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
