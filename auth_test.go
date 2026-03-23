package main

import (
	"encoding/base64"
	"strconv"
	"testing"
)

func TestMakeUin_IsBase64(t *testing.T) {
	uin := makeUin()
	decoded, err := base64.StdEncoding.DecodeString(uin)
	if err != nil {
		t.Fatalf("makeUin() 结果不是合法 base64: %v", err)
	}
	// 解码后应是一个十进制数字字符串
	if _, err := strconv.ParseUint(string(decoded), 10, 64); err != nil {
		t.Errorf("makeUin() 解码后应为数字字符串，got %q", string(decoded))
	}
}

func TestMakeUin_Unique(t *testing.T) {
	// 生成 100 个，确保不全相同（防重放依赖随机性）
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uin := makeUin()
		seen[uin] = true
	}
	if len(seen) < 90 {
		t.Errorf("makeUin() 随机性不足，100 次只有 %d 个不同值", len(seen))
	}
}
