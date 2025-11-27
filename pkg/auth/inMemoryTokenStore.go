package auth

import (
	"context"
	"sync"
	"time"
)

var _ SessionStore = &InMemorySessionStore{}

type InMemorySessionStore struct {
	lock         sync.Mutex
	sessions     map[string]*Session
	userSessions map[string][]*Session
}

func NewInMemoryTokenStore() *InMemorySessionStore {
	return &InMemorySessionStore{sessions: make(map[string]*Session), userSessions: make(map[string][]*Session), lock: sync.Mutex{}}
}

func (s *InMemorySessionStore) StoreSession(ctx context.Context, session Session) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.userSessions[session.UserID] = append(s.userSessions[session.UserID], &session)
	s.sessions[session.Token] = &session
	return nil
}

func (s *InMemorySessionStore) DeleteSessions(ctx context.Context, userID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	sessions, ok := s.userSessions[userID]
	if !ok {
		return nil
	}
	for _, session := range sessions {
		delete(s.sessions, session.Token)
	}
	delete(s.userSessions, userID)
	return nil
}

func (s *InMemorySessionStore) VerifySession(ctx context.Context, token string, now time.Time) (Session, bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	session, ok := s.sessions[token]
	if !ok {
		return Session{}, false, nil
	}
	return *session, session.ExpiresAt.After(now), nil
}
