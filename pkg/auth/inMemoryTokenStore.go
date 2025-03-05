package auth

import (
	"context"
	"sync"
	"time"
)

var _ TokenStore = &InMemoryTokenStore{}

type storedToken struct {
	userID    string
	token     string
	expiresAt time.Time
}

type InMemoryTokenStore struct {
	lock       sync.Mutex
	tokens     map[string]storedToken
	userTokens map[string][]string
}

func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{tokens: make(map[string]storedToken), userTokens: make(map[string][]string), lock: sync.Mutex{}}
}

func (s *InMemoryTokenStore) StoreToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.userTokens[userID] = append(s.userTokens[userID], token)
	s.tokens[token] = storedToken{userID, token, expiresAt}
	return nil
}

func (s *InMemoryTokenStore) DeleteTokens(ctx context.Context, userID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	tokens, ok := s.userTokens[userID]
	if !ok {
		return nil
	}
	for _, token := range tokens {
		delete(s.tokens, token)
	}
	return nil
}

func (s *InMemoryTokenStore) VerifyToken(ctx context.Context, token string, now time.Time) (string, bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sToken, ok := s.tokens[token]
	if !ok {
		return "", false, nil
	}
	return sToken.userID, sToken.expiresAt.After(now), nil
}
