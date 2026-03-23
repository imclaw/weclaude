package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const pollTimeoutMS = 30000

// Message 对应 iLink Bot 推送的消息结构
type Message struct {
	MessageType  int        `json:"message_type"`
	FromUserID   string     `json:"from_user_id"`
	ToUserID     string     `json:"to_user_id"`
	ContextToken string     `json:"context_token"`
	ItemList     []ItemList `json:"item_list"`
}

type ItemList struct {
	Type     int      `json:"type"`
	TextItem TextItem `json:"text_item"`
}

type TextItem struct {
	Text string `json:"text"`
}

type poller struct {
	c        *client
	sessions *sessionStore
	sender   *sender
}

func newPoller(c *client, sessions *sessionStore, sender *sender) *poller {
	return &poller{c: c, sessions: sessions, sender: sender}
}

func (p *poller) start(ctx context.Context) {
	var getUpdatesBuf string
	processing := sync.Map{} // userID → struct{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		data, err := p.c.post("/ilink/bot/getupdates", map[string]any{
			"get_updates_buf":       getUpdatesBuf,
			"base_info":             map[string]any{"channel_version": "1.0.2"},
			"longpolling_timeout_ms": pollTimeoutMS,
		})
		if err != nil {
			fmt.Printf("[poll error] %v\n", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		// 更新游标
		if buf, ok := data["get_updates_buf"].(string); ok && buf != "" {
			getUpdatesBuf = buf
		}

		// 解析消息列表
		msgs := parseMsgs(data)
		for _, msg := range msgs {
			go p.handleMessage(ctx, msg, &processing)
		}
	}
}

func (p *poller) handleMessage(ctx context.Context, msg Message, processing *sync.Map) {
	// 只处理用户发来的消息
	if msg.MessageType != 1 {
		return
	}

	userID := msg.FromUserID
	botID := msg.ToUserID
	contextToken := msg.ContextToken

	// 提取文本
	var userText string
	for _, item := range msg.ItemList {
		if item.Type == 1 {
			userText = item.TextItem.Text
			break
		}
	}
	if userText == "" {
		return
	}

	fmt.Printf("[%s] %s: %s\n", time.Now().Format("15:04:05"), userID, truncate(userText, 80))

	// 防并发
	if _, loaded := processing.LoadOrStore(userID, struct{}{}); loaded {
		p.sender.sendText(userID, "上条消息还在处理中，请稍候...", contextToken, botID) //nolint:errcheck
		return
	}
	defer processing.Delete(userID)

	// 重置命令
	if isResetCommand(userText) {
		p.sessions.delete(userID)
		p.sender.sendText(userID, "对话已重置，开始新的会话。", contextToken, botID) //nolint:errcheck
		return
	}

	sessionID := p.sessions.get(userID)
	text, newSessionID, err := askClaude(userText, sessionID)
	if err != nil {
		fmt.Printf("[claude error] %v\n", err)
		p.sender.sendText(userID, fmt.Sprintf("抱歉，处理出错了：%s", truncate(err.Error(), 200)), contextToken, botID) //nolint:errcheck
		return
	}

	if newSessionID != "" {
		p.sessions.set(userID, newSessionID)
	}

	if err := p.sender.sendText(userID, text, contextToken, botID); err != nil {
		fmt.Printf("[send error] %v\n", err)
		return
	}

	fmt.Printf("[%s] → 回复 %s (%d 字)\n", time.Now().Format("15:04:05"), userID, len(text))
}

// parseMsgs 从 getupdates 响应里安全地解析消息列表
func parseMsgs(data map[string]any) []Message {
	raw, ok := data["msgs"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	var msgs []Message
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		msg := Message{
			FromUserID:   strField(m, "from_user_id"),
			ToUserID:     strField(m, "to_user_id"),
			ContextToken: strField(m, "context_token"),
		}
		if mt, ok := m["message_type"].(float64); ok {
			msg.MessageType = int(mt)
		}
		if items, ok := m["item_list"].([]any); ok {
			for _, it := range items {
				itm, ok := it.(map[string]any)
				if !ok {
					continue
				}
				var il ItemList
				if t, ok := itm["type"].(float64); ok {
					il.Type = int(t)
				}
				if ti, ok := itm["text_item"].(map[string]any); ok {
					il.TextItem.Text = strField(ti, "text")
				}
				msg.ItemList = append(msg.ItemList, il)
			}
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func strField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
