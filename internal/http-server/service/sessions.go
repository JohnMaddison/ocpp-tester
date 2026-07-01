package service

import (
	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
)

type SessionsService struct {
	store *data.SessionStore
}

func NewSessionsService(store *data.SessionStore) *SessionsService {
	return &SessionsService{store: store}
}

func (s *SessionsService) ListSessions() []data.Session {
	return s.store.List()
}

func (s *SessionsService) GetSession(chargePointID string) (data.Session, bool) {
	return s.store.Get(chargePointID)
}
