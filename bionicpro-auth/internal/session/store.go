package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type Session struct {
	ID           string
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiry       time.Time // access token expiry
	ExpiresAt    time.Time // session expiry
}

type PendingAuth struct {
	State        string
	CodeVerifier string
	ExpiresAt    time.Time
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	pending  map[string]*PendingAuth
	ttl      time.Duration
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{
		sessions: make(map[string]*Session),
		pending:  make(map[string]*PendingAuth),
		ttl:      ttl,
	}
	go s.cleanupLoop()
	return s
}

func (s *Store) Create(accessToken, refreshToken, idToken string, accessExpiry time.Time) (*Session, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	sess := &Session{
		ID:           id,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Expiry:       accessExpiry,
		ExpiresAt:    now.Add(s.ttl),
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess, nil
}

func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	cp := *sess
	return &cp, true
}

func (s *Store) UpdateTokens(id, accessToken, refreshToken string, accessExpiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return fmt.Errorf("session not found")
	}
	sess.AccessToken = accessToken
	if refreshToken != "" {
		sess.RefreshToken = refreshToken
	}
	sess.Expiry = accessExpiry
	return nil
}

// Rotate moves tokens to a new session id and deletes the old one.
func (s *Store) Rotate(oldID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.sessions[oldID]
	if !ok || time.Now().After(old.ExpiresAt) {
		return nil, fmt.Errorf("session not found")
	}

	newID, err := newID()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		ID:           newID,
		AccessToken:  old.AccessToken,
		RefreshToken: old.RefreshToken,
		IDToken:      old.IDToken,
		Expiry:       old.Expiry,
		ExpiresAt:    old.ExpiresAt,
	}
	s.sessions[newID] = sess
	delete(s.sessions, oldID)
	cp := *sess
	return &cp, nil
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *Store) SavePending(state, codeVerifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[state] = &PendingAuth{
		State:        state,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	return nil
}

func (s *Store) TakePending(state string) (*PendingAuth, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[state]
	if !ok {
		return nil, false
	}
	delete(s.pending, state)
	if time.Now().After(p.ExpiresAt) {
		return nil, false
	}
	return p, true
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	for state, p := range s.pending {
		if now.After(p.ExpiresAt) {
			delete(s.pending, state)
		}
	}
}

func newID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
