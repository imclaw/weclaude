package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/mdp/qrterminal/v3"
)

const ilinkBaseURL = "https://ilinkai.weixin.qq.com"

// Auth 保存登录凭证
type Auth struct {
	BotToken string `json:"bot_token"`
	BaseURL  string `json:"base_url"`
	LoggedAt string `json:"logged_at"`
}

func loadAuth() (*Auth, error) {
	data, err := os.ReadFile(authFilePath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, nil
	}
	return &auth, nil
}

func saveAuth(auth *Auth) error {
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(authFilePath(), data, 0600)
}

// makeUin 生成随机 X-WECHAT-UIN：uint32 → 十进制字符串 → base64
func makeUin() string {
	n := rand.Uint32()
	s := fmt.Sprintf("%d", n)
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// authGet 用于登录阶段（无 bot_token）的 GET 请求
func authGet(path string) (map[string]any, error) {
	req, err := http.NewRequest("GET", ilinkBaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("X-WECHAT-UIN", makeUin())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// login 扫码登录，返回 Auth
func login() (*Auth, error) {
	fmt.Println("正在获取微信二维码...")

	res, err := authGet("/ilink/bot/get_bot_qrcode?bot_type=3")
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败: %w", err)
	}

	qrToken, _ := res["qrcode"].(string)
	qrContent, _ := res["qrcode_img_content"].(string)
	if qrToken == "" {
		return nil, fmt.Errorf("获取二维码失败: %v", res)
	}
	if qrContent == "" {
		qrContent = qrToken
	}

	// 生成 PNG 并保存到临时目录
	imgPath := filepath.Join(os.TempDir(), "wechat-qrcode.png")
	if err := qrcode.WriteFile(qrContent, qrcode.Medium, 256, imgPath); err != nil {
		return nil, fmt.Errorf("生成二维码图片失败: %w", err)
	}

	// Windows 直接打开图片；其他平台在终端显示 ASCII 二维码
	if runtime.GOOS == "windows" {
		go func() {
			exec.Command("cmd", "/c", "start", "", imgPath).Start()
		}()
		fmt.Printf("\n二维码已保存至：%s\n", imgPath)
		fmt.Print("请用微信扫描弹出的二维码图片...\n\n")
	} else {
		// 在终端直接显示 ASCII 二维码
		fmt.Print("\n")
		qrterminal.GenerateHalfBlock(qrContent, qrterminal.M, os.Stdout)
		fmt.Print("\n")

		// 尝试打开图片（如果有图形界面）
		go func() {
			var openCmd *exec.Cmd
			if runtime.GOOS == "darwin" {
				openCmd = exec.Command("open", imgPath)
			} else {
				openCmd = exec.Command("xdg-open", imgPath)
			}
			openCmd.Start()
		}()
		fmt.Printf("二维码已保存至：%s\n", imgPath)
		fmt.Print("请用微信扫描上方二维码...\n\n")
	}

	// 轮询扫码状态，最多等 3 分钟
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(1500 * time.Millisecond)

		status, err := authGet(fmt.Sprintf("/ilink/bot/get_qrcode_status?qrcode=%s", qrToken))
		if err != nil {
			continue
		}

		switch status["status"] {
		case "scanned":
			fmt.Print("已扫码，请在手机上确认...\r")
		case "confirmed":
			botToken, _ := status["bot_token"].(string)
			auth := &Auth{
				BotToken: botToken,
				LoggedAt: time.Now().Format(time.RFC3339),
			}
			if u, ok := status["baseurl"].(string); ok && u != "" {
				auth.BaseURL = u
			} else {
				auth.BaseURL = ilinkBaseURL
			}
			if err := saveAuth(auth); err != nil {
				return nil, err
			}
			fmt.Print("\n登录成功！\n\n")
			return auth, nil
		case "expired":
			return nil, fmt.Errorf("二维码已过期，请重新运行登录")
		}
	}

	return nil, fmt.Errorf("登录超时，请重试")
}
