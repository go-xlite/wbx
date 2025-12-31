package dummy_session_svc

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// SessionData represents the data stored in a session
type SessionData struct {
	UserID    string
	Username  string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      map[string]interface{}
}

// MySessionService is a non-persistent in-memory session service
type MySessionService struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
	ttl      time.Duration
}

func NewDummySessionService() *MySessionService {
	svc := &MySessionService{
		sessions: make(map[string]*SessionData),
		ttl:      24 * time.Hour, // Default 24 hour expiration
	}

	// Start cleanup goroutine
	go svc.cleanupExpired()

	return svc
}

// SetTTL sets the time-to-live for sessions
func (s *MySessionService) SetTTL(ttl time.Duration) *MySessionService {
	s.ttl = ttl
	return s
}

func (s *MySessionService) Validate(token string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return nil, errors.New("session not found")
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	return session, nil
}

func (s *MySessionService) Issue(data any) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	sessionData := &SessionData{
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
		Data:      make(map[string]interface{}),
	}

	// If data is provided, extract common fields
	if dataMap, ok := data.(map[string]interface{}); ok {
		if userID, ok := dataMap["user_id"].(string); ok {
			sessionData.UserID = userID
		}
		if username, ok := dataMap["username"].(string); ok {
			sessionData.Username = username
		}
		if email, ok := dataMap["email"].(string); ok {
			sessionData.Email = email
		}
		sessionData.Data = dataMap
	}

	s.mu.Lock()
	s.sessions[token] = sessionData
	s.mu.Unlock()

	return token, nil
}

func (s *MySessionService) Refresh(token string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[token]
	if !exists {
		return "", errors.New("session not found")
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		return "", errors.New("session expired")
	}

	// Generate new token
	newToken, err := generateToken()
	if err != nil {
		return "", err
	}

	// Create new session with same data but extended expiration
	newSession := &SessionData{
		UserID:    session.UserID,
		Username:  session.Username,
		Email:     session.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.ttl),
		Data:      session.Data,
	}

	// Store new session and remove old one
	s.sessions[newToken] = newSession
	delete(s.sessions, token)

	return newToken, nil
}

func (s *MySessionService) Revoke(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[token]; !exists {
		return errors.New("session not found")
	}

	delete(s.sessions, token)
	return nil
}

// GetSessionCount returns the number of active sessions (useful for monitoring)
func (s *MySessionService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// cleanupExpired removes expired sessions periodically
func (s *MySessionService) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}

// generateToken creates a random session token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
