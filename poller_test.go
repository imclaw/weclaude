package main

import "testing"

func TestParseMsgs_Empty(t *testing.T) {
	msgs := parseMsgs(map[string]any{})
	if len(msgs) != 0 {
		t.Errorf("无 msgs 字段应返回空，got %v", msgs)
	}
}

func TestParseMsgs_TextMessage(t *testing.T) {
	data := map[string]any{
		"msgs": []any{
			map[string]any{
				"message_type":  float64(1),
				"from_user_id":  "user_abc",
				"to_user_id":    "bot_xyz",
				"context_token": "ctx_token_123",
				"item_list": []any{
					map[string]any{
						"type": float64(1),
						"text_item": map[string]any{
							"text": "你好",
						},
					},
				},
			},
		},
	}

	msgs := parseMsgs(data)
	if len(msgs) != 1 {
		t.Fatalf("应解析出 1 条消息，got %d", len(msgs))
	}

	msg := msgs[0]
	if msg.MessageType != 1 {
		t.Errorf("MessageType 应为 1，got %d", msg.MessageType)
	}
	if msg.FromUserID != "user_abc" {
		t.Errorf("FromUserID 应为 user_abc，got %s", msg.FromUserID)
	}
	if msg.ToUserID != "bot_xyz" {
		t.Errorf("ToUserID 应为 bot_xyz，got %s", msg.ToUserID)
	}
	if msg.ContextToken != "ctx_token_123" {
		t.Errorf("ContextToken 应为 ctx_token_123，got %s", msg.ContextToken)
	}
	if len(msg.ItemList) != 1 {
		t.Fatalf("ItemList 应有 1 项，got %d", len(msg.ItemList))
	}
	if msg.ItemList[0].TextItem.Text != "你好" {
		t.Errorf("文本内容应为 '你好'，got %s", msg.ItemList[0].TextItem.Text)
	}
}

func TestParseMsgs_MultipleMessages(t *testing.T) {
	data := map[string]any{
		"msgs": []any{
			map[string]any{
				"message_type": float64(1),
				"from_user_id": "user_1",
				"item_list":    []any{},
			},
			map[string]any{
				"message_type": float64(2), // bot 发出的，不是用户消息
				"from_user_id": "bot_1",
				"item_list":    []any{},
			},
		},
	}

	msgs := parseMsgs(data)
	if len(msgs) != 2 {
		t.Fatalf("应解析出 2 条消息，got %d", len(msgs))
	}
	if msgs[0].FromUserID != "user_1" {
		t.Errorf("第一条消息 FromUserID 应为 user_1，got %s", msgs[0].FromUserID)
	}
	if msgs[1].MessageType != 2 {
		t.Errorf("第二条消息 MessageType 应为 2，got %d", msgs[1].MessageType)
	}
}

func TestParseMsgs_InvalidType(t *testing.T) {
	// msgs 字段不是数组，应安全返回空
	data := map[string]any{
		"msgs": "not an array",
	}
	msgs := parseMsgs(data)
	if len(msgs) != 0 {
		t.Errorf("非法 msgs 类型应返回空，got %v", msgs)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		n        int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello"},
		{"", 5, ""},
		{"abc", 3, "abc"},
	}
	for _, c := range cases {
		got := truncate(c.input, c.n)
		if got != c.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.input, c.n, got, c.expected)
		}
	}
}
