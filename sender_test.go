package main

import (
	"strings"
	"testing"
)

func TestSplitText_ShortText(t *testing.T) {
	chunks := splitText("hello world", 4000)
	if len(chunks) != 1 || chunks[0] != "hello world" {
		t.Errorf("短文本不应分片，got %v", chunks)
	}
}

func TestSplitText_ExactLimit(t *testing.T) {
	text := strings.Repeat("a", 4000)
	chunks := splitText(text, 4000)
	if len(chunks) != 1 {
		t.Errorf("恰好等于 maxLen 不应分片，got %d 片", len(chunks))
	}
}

func TestSplitText_SplitOnNewline(t *testing.T) {
	// 构造一段在 4000 字内有换行符的文本
	part1 := strings.Repeat("a", 3990) + "\n"
	part2 := strings.Repeat("b", 100)
	text := part1 + part2

	chunks := splitText(text, 4000)
	if len(chunks) != 2 {
		t.Fatalf("应分为 2 片，got %d 片", len(chunks))
	}
	// 第一片应在换行处截断，包含换行符
	if !strings.HasSuffix(chunks[0], "\n") {
		t.Errorf("第一片应以换行结尾，got: %q", chunks[0][len(chunks[0])-5:])
	}
	if chunks[1] != part2 {
		t.Errorf("第二片内容不对，got %q", chunks[1])
	}
}

func TestSplitText_NoNewline(t *testing.T) {
	// 超长但没有换行，强制按 maxLen 截断
	text := strings.Repeat("x", 9000)
	chunks := splitText(text, 4000)
	if len(chunks) != 3 {
		t.Fatalf("应分为 3 片，got %d 片", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != 9000 {
		t.Errorf("分片后总长度应为 9000，got %d", total)
	}
}

func TestSplitText_Empty(t *testing.T) {
	chunks := splitText("", 4000)
	if len(chunks) != 1 || chunks[0] != "" {
		t.Errorf("空字符串应返回 [\"\"]，got %v", chunks)
	}
}
