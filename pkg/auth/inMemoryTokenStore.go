package auth

import (
	"context"
	"slices"
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

func (s *InMemorySessionStore) ReplaceSession(ctx context.Context, oldSession, newSession Session) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	// remove old token and set new token
	delete(s.sessions, oldSession.Token)
	s.sessions[newSession.Token] = &newSession

	sessions := s.userSessions[oldSession.UserID]
	for i, session := range sessions {
		if session.Token == oldSession.Token {
			// replace old session with new session in userSessions
			sessions[i] = &newSession
			return nil
		}
	}
	// if old session was not found, add new session
	s.userSessions[oldSession.UserID] = append(sessions, &newSession)
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

func (s *InMemorySessionStore) VerifyCSRFToken(ctx context.Context, userID, csrfToken string, now time.Time) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sessions, ok := s.userSessions[userID]
	if !ok {
		return false, nil
	}
	if len(sessions) == 0 {
		return false, nil
	}
	for _, session := range sessions {
		if session.CSRFToken == csrfToken {
			return session.ExpiresAt.After(now), nil
		}
	}
	return false, nil
}

func (s *InMemorySessionStore) GC(ctx context.Context, now time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for token, session := range s.sessions {
		if session.ExpiresAt.Before(now) {
			delete(s.sessions, token)
		}
	}
	for userID, sessions := range s.userSessions {
		s.userSessions[userID] = slices.DeleteFunc(sessions, func(sess *Session) bool {
			return sess.ExpiresAt.Before(now)
		})
	}
	return nil
}
