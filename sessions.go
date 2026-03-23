package main

import (
	"encoding/json"
	"os"
	"sync"
)

// sessionStore 维护 微信用户ID → Claude session_id 的映射，线程安全，持久化到磁盘
type sessionStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newSessionStore() *sessionStore {
	s := &sessionStore{data: make(map[string]string)}
	s.load()
	return s
}

func (s *sessionStore) get(userID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[userID]
}

func (s *sessionStore) set(userID, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[userID] = sessionID
	s.save()
}

func (s *sessionStore) delete(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, userID)
	s.save()
}

func (s *sessionStore) load() {
	data, err := os.ReadFile(sessionsFilePath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.data) //nolint:errcheck
}

func (s *sessionStore) save() {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(sessionsFilePath(), data, 0600) //nolint:errcheck
}
