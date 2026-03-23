package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const maxMsgLen = 4000

type sender struct {
	c *client
}

func newSender(c *client) *sender {
	return &sender{c: c}
}

func (s *sender) sendText(toUserID, text, contextToken, fromUserID string) error {
	chunks := splitText(text, maxMsgLen)
	for _, chunk := range chunks {
		payload := map[string]any{
			"msg": map[string]any{
				"from_user_id":  fromUserID,
				"to_user_id":    toUserID,
				"client_id":     fmt.Sprintf("weclaude-%d-%s", time.Now().UnixMilli(), randStr(8)),
				"message_type":  2,
				"message_state": 2,
				"context_token": contextToken,
				"item_list": []map[string]any{
					{"type": 1, "text_item": map[string]any{"text": chunk}},
				},
			},
			"base_info": map[string]any{"channel_version": "1.0.2"},
		}
		res, err := s.c.post("/ilink/bot/sendmessage", payload)
		if err != nil {
			return err
		}
		fmt.Printf("[sendmessage res] %v\n", res)
	}
	return nil
}

// splitText 按最大长度切分，尽量在换行处截断
func splitText(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + maxLen
		if end >= len(text) {
			chunks = append(chunks, text[start:])
			break
		}
		// 找最近的换行符
		if idx := strings.LastIndex(text[start:end], "\n"); idx > 0 {
			end = start + idx + 1
		}
		chunks = append(chunks, text[start:end])
		start = end
	}
	return chunks
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
