package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var claudeBin = func() string {
	if v := os.Getenv("CLAUDE_BIN"); v != "" {
		return v
	}
	return "claude"
}()

var resetCommands = map[string]bool{
	"/reset": true,
	"重置":    true,
	"reset":  true,
	"/new":   true,
	"新对话":   true,
}

// claudeOutput 对应 claude --output-format json 的输出结构
type claudeOutput struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
	IsError   bool   `json:"is_error"`
}

func isResetCommand(text string) bool {
	return resetCommands[strings.TrimSpace(strings.ToLower(text))]
}

// askClaude 调用本地 claude CLI，返回回复文本和新的 sessionID
func askClaude(message, sessionID string) (text, newSessionID string, err error) {
	result, err := spawnClaude(buildArgs(message, sessionID))
	if err != nil {
		// session 过期时自动降级为新对话
		if sessionID != "" && strings.Contains(err.Error(), "No conversation found") {
			fmt.Printf("[claude] session %s 已失效，降级为新对话\n", sessionID)
			result, err = spawnClaude(buildArgs(message, ""))
			if err != nil {
				return "", "", err
			}
		} else {
			return "", "", err
		}
	}

	var out claudeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		return "", "", fmt.Errorf("Claude 输出解析失败: %s", result[:min(len(result), 300)])
	}
	if out.IsError {
		return "", "", fmt.Errorf("Claude 返回错误: %s", out.Result)
	}
	return out.Result, out.SessionID, nil
}

func buildArgs(message, sessionID string) []string {
	args := []string{"--dangerously-skip-permissions", "--output-format", "json", "-p", message}
	if sessionID != "" {
		args = append([]string{"--resume", sessionID}, args...)
	}
	return args
}

func spawnClaude(args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, claudeBin, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = nil

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if stderr.Len() > 0 {
		fmt.Printf("[claude stderr] %s\n", stderr.String()[:min(stderr.Len(), 200)])
	}

	if err != nil && stdout.Len() == 0 {
		return "", fmt.Errorf("claude 启动失败 (exit %v): %s", err, stderr.String()[:min(stderr.Len(), 300)])
	}
	return stdout.String(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
