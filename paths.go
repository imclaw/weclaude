package main

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	_dataDir     string
	_dataDirOnce sync.Once
)

func getDataDir() string {
	_dataDirOnce.Do(func() {
		home, _ := os.UserHomeDir()
		_dataDir = filepath.Join(home, ".weclaude")
		os.MkdirAll(_dataDir, 0700)
	})
	return _dataDir
}

func authFilePath() string {
	return filepath.Join(getDataDir(), "auth.json")
}

func sessionsFilePath() string {
	return filepath.Join(getDataDir(), "sessions.json")
}
