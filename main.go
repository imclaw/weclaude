package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sevlyar/go-daemon"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "login":
		if _, err := login(); err != nil {
			log.Fatalf("登录失败: %v", err)
		}
	case "status":
		auth, err := loadAuth()
		if err != nil || auth == nil {
			fmt.Println("未登录")
			os.Exit(1)
		}
		fmt.Printf("已登录\n登录时间: %s\n", auth.LoggedAt)

		// 检查守护进程状态
		pidData, err := os.ReadFile(pidFilePath())
		if err == nil {
			if pid, err := strconv.Atoi(string(pidData)); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						fmt.Printf("守护进程: 运行中 (PID: %d)\n", pid)
					} else {
						fmt.Println("守护进程: 未运行 (PID 文件过期)")
					}
				}
			}
		} else {
			fmt.Println("守护进程: 未启动")
		}
	case "logout":
		if err := os.Remove(authFilePath()); err != nil && !os.IsNotExist(err) {
			log.Fatalf("退出登录失败: %v", err)
		}
		fmt.Println("已退出登录")
	case "reset":
		if err := os.Remove(sessionsFilePath()); err != nil && !os.IsNotExist(err) {
			log.Fatalf("重置会话失败: %v", err)
		}
		fmt.Println("所有会话已重置")
	case "send":
		cmdSend(os.Args[2:])
	case "contacts":
		cmdContacts()
	case "--help", "-h", "help":
		printHelp()
	case "daemon":
		runServerDaemon()
	case "stop":
		stopDaemon()
	default:
		runServer()
	}
}

func printHelp() {
	fmt.Print(`weclaude - 微信 iLink Bot → Claude Code 中间层

用法:
  weclaude                        启动服务（前台）
  weclaude daemon                 启动守护进程（后台）
  weclaude stop                   停止守护进程
  weclaude login                  扫码登录
  weclaude status                 查看登录状态和守护进程信息
  weclaude contacts               列出所有已知联系人 ID
  weclaude send <text>            主动发送消息给默认用户（登录用户）
  weclaude send <userID> <text>   主动发送消息给指定联系人
  weclaude reset                  清除所有会话
  weclaude logout                 退出登录
`)
}

func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".weclaude", "daemon.pid")
}

func cmdContacts() {
	sessions := newSessionStore()
	ids := sessions.list()
	if len(ids) == 0 {
		fmt.Println("暂无联系人（尚未收到任何微信消息）")
		return
	}
	fmt.Printf("已知联系人（共 %d 个）:\n", len(ids))
	for _, id := range ids {
		fmt.Println(" ", id)
	}
}

func cmdSend(args []string) {
	auth, err := loadAuth()
	if err != nil || auth == nil {
		log.Fatal("未登录，请先运行: weclaude login")
	}
	if auth.BotID == "" {
		log.Fatal("未存储 Bot ID，请重新运行 weclaude login 后再试")
	}

	var toUserID, text string
	switch len(args) {
	case 1:
		if auth.UserID == "" {
			log.Fatal("未存储默认用户 ID，请重新运行 weclaude login 或指定用户: weclaude send <userID> <text>")
		}
		toUserID = auth.UserID
		text = args[0]
	case 2:
		toUserID = args[0]
		text = args[1]
	default:
		fmt.Fprintln(os.Stderr, "用法: weclaude send <text>             # 发送给默认用户")
		fmt.Fprintln(os.Stderr, "      weclaude send <userID> <text>   # 发送给指定用户")
		os.Exit(1)
	}

	client := newClient(auth)
	s := newSender(client)
	if err := s.sendText(toUserID, text, "", auth.BotID); err != nil {
		log.Fatalf("发送失败: %v", err)
	}
	fmt.Printf("已发送给 %s\n", toUserID)
}

func runServerDaemon() {
	auth, err := loadAuth()
	if err != nil || auth == nil {
		log.Fatal("未登录，请先运行: weclaude login")
	}

	// 生成日期日志文件名
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(getDataDir(), "daemon-"+today+".log")

	cntxt := &daemon.Context{
		PidFileName: pidFilePath(),
		PidFilePerm: 0644,
		LogFileName: logFile,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Chroot:      "",
		Umask:       027,
	}

	if child, _ := cntxt.Reborn(); child != nil {
		fmt.Printf("守护进程已启动，PID: %d\n", child.Pid)
		return
	}
	defer cntxt.Release()

	runServer()
}

func stopDaemon() {
	pidData, err := os.ReadFile(pidFilePath())
	if err != nil {
		fmt.Println("守护进程未运行")
		os.Exit(1)
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		fmt.Println("无效的 PID 文件")
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("无法找到进程:", err)
		os.Exit(1)
	}

	if err := proc.Kill(); err != nil {
		fmt.Println("停止守护进程失败:", err)
		os.Exit(1)
	}

	os.Remove(pidFilePath())
	fmt.Println("守护进程已停止")
}

func runServer() {
	auth, err := loadAuth()
	if err != nil {
		log.Fatalf("读取登录信息失败: %v", err)
	}
	if auth == nil {
		fmt.Println("尚未登录，开始扫码登录...")
		auth, err = login()
		if err != nil {
			log.Fatalf("登录失败: %v", err)
		}
	}

	fmt.Printf("已登录（%s），正在启动...\n", auth.LoggedAt)

	client := newClient(auth)
	sessions := newSessionStore()
	sender := newSender(client)
	poller := newPoller(client, sessions, sender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\n正在退出...")
		cancel()
	}()

	fmt.Print("开始监听微信消息...\n\n")
	poller.start(ctx)
}
