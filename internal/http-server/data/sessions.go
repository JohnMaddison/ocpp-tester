package data

import (
	"sort"
	"sync"
	"time"
)

type Session struct {
	ChargePointID string    `json:"chargePointId"`
	Protocol      string    `json:"protocol"`
	RemoteAddr    string    `json:"remoteAddr"`
	LocalAddr     string    `json:"localAddr"`
	ConnectedAt   time.Time `json:"connectedAt"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]Session),
	}
}

func (s *SessionStore) Upsert(session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ChargePointID] = session
}

func (s *SessionStore) Delete(chargePointID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, chargePointID)
}

func (s *SessionStore) Get(chargePointID string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[chargePointID]
	return session, ok
}

func (s *SessionStore) List() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ChargePointID < sessions[j].ChargePointID
	})

	return sessions
}
