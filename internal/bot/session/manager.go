package session

import (
	"sync"
	"time"
)

type Manager struct {
	mu       sync.Mutex
	sessions map[int64]*Session
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[int64]*Session)}
}

func (m *Manager) Get(userID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[userID]
	if !ok {
		s = &Session{UpdatedAt: time.Now()}
		m.sessions[userID] = s
	}
	return s
}

func (m *Manager) Reset(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[userID] = &Session{UpdatedAt: time.Now()}
}
