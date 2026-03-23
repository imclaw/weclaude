package main

import "testing"

func TestIsResetCommand(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"/reset", true},
		{"重置", true},
		{"reset", true},
		{"/new", true},
		{"新对话", true},
		// 大小写不敏感
		{"RESET", true},
		{"Reset", true},
		// 前后空格
		{"  /reset  ", true},
		// 非重置命令
		{"hello", false},
		{"", false},
		{"重置一下", false},
		{"/reset now", false},
	}

	for _, c := range cases {
		got := isResetCommand(c.input)
		if got != c.expected {
			t.Errorf("isResetCommand(%q) = %v, want %v", c.input, got, c.expected)
		}
	}
}

func TestBuildArgs_NoSession(t *testing.T) {
	args := buildArgs("hello", "")
	// 不应包含 --resume
	for _, a := range args {
		if a == "--resume" {
			t.Error("无 sessionID 时不应包含 --resume")
		}
	}
	// 应包含 -p 和消息内容
	found := false
	for i, a := range args {
		if a == "-p" && i+1 < len(args) && args[i+1] == "hello" {
			found = true
		}
	}
	if !found {
		t.Errorf("应包含 -p hello，got %v", args)
	}
}

func TestBuildArgs_WithSession(t *testing.T) {
	args := buildArgs("hello", "sess-123")
	// 应包含 --resume sess-123
	foundResume := false
	for i, a := range args {
		if a == "--resume" && i+1 < len(args) && args[i+1] == "sess-123" {
			foundResume = true
		}
	}
	if !foundResume {
		t.Errorf("有 sessionID 时应包含 --resume sess-123，got %v", args)
	}
}

func TestBuildArgs_AlwaysHasFlags(t *testing.T) {
	args := buildArgs("test", "")
	flags := map[string]bool{
		"--dangerously-skip-permissions": false,
		"--output-format":               false,
		"-p":                            false,
	}
	for _, a := range args {
		if _, ok := flags[a]; ok {
			flags[a] = true
		}
	}
	for flag, found := range flags {
		if !found {
			t.Errorf("缺少必要参数: %s，args = %v", flag, args)
		}
	}
}
