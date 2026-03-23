package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// client 是 iLink Bot 的 HTTP 客户端
type client struct {
	botToken string
	baseURL  string
}

func newClient(auth *Auth) *client {
	return &client{
		botToken: auth.BotToken,
		baseURL:  auth.BaseURL,
	}
}

func (c *client) post(path string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("X-WECHAT-UIN", makeUin())
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("iLink API %s → HTTP %d", path, resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
