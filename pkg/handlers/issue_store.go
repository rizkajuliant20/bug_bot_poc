package handlers

import (
	"sync"

	"github.com/rizkajuliant20/bug-bot/pkg/services"
	"github.com/slack-go/slack"
)

// IssueData stores the context needed to create tickets from button interactions
type IssueData struct {
	Analysis       *services.MultiIssueAnalysis
	BugDescription string
	ThreadMessages []slack.Message
	Reporter       string
	TeamID         string
	Channel        string
	ThreadTS       string
	TS             string
}

// IssueStore is an in-memory store for multi-issue data
type IssueStore struct {
	mu    sync.RWMutex
	store map[string]*IssueData // key: channel_ts
}

var globalIssueStore = &IssueStore{
	store: make(map[string]*IssueData),
}

func (s *IssueStore) Set(key string, data *IssueData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = data
}

func (s *IssueStore) Get(key string) (*IssueData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.store[key]
	return data, ok
}

func (s *IssueStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, key)
}

func (s *IssueStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = make(map[string]*IssueData)
}

func GetIssueStore() *IssueStore {
	return globalIssueStore
}
