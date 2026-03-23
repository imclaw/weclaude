package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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
	case "--help", "-h", "help":
		printHelp()
	default:
		runServer()
	}
}

func printHelp() {
	fmt.Print(`weclaude - 微信 iLink Bot → Claude Code 中间层

用法:
  weclaude            启动服务
  weclaude login      扫码登录
  weclaude status     查看登录状态
  weclaude reset      清除所有会话
  weclaude logout     退出登录
`)
}

func runServer() {
	auth, err := loadAuth()
	if err != nil {
		log.Fatalf("读取登录信息失败: %v", err)
	}
	if auth == nil {
		log.Fatal("未登录，请先运行: weclaude login")
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
