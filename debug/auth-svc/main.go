package authsvc

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-xlite/wbx/weblite"
)

// User represents a user in the system
type User struct {
	Username string `json:"username"`
	Password string `json:"-"` // Never expose password in JSON
	Role     string `json:"role"`
}

// AuthService implements IWebAuthProvider
type AuthService struct {
	users          map[string]*User
	mu             sync.RWMutex
	sessionManager *weblite.SessionManager
}

func NewWebAuthService() *AuthService {
	return &AuthService{
		users: make(map[string]*User),
	}
}

// SetSessionManager sets the session manager for cookie handling
func (s *AuthService) SetSessionManager(sm *weblite.SessionManager) *AuthService {
	s.sessionManager = sm
	return s
}

// AddUser adds a user to the auth service
func (s *AuthService) AddUser(username, password, role string) *AuthService {
	s.mu.Lock()
	s.users[username] = &User{
		Username: username,
		Password: password,
		Role:     role,
	}
	s.mu.Unlock()
	return s
}

// ValidateCredentials checks username/password
func (s *AuthService) ValidateCredentials(username, password string) (*User, bool) {
	s.mu.RLock()
	user, exists := s.users[username]
	s.mu.RUnlock()

	if !exists || user.Password != password {
		return nil, false
	}
	return user, true
}

// Login handles user login
func (s *AuthService) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	user, valid := s.ValidateCredentials(req.Username, req.Password)
	if !valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	// Issue session token
	if s.sessionManager != nil && s.sessionManager.Service != nil {
		sessionData := map[string]interface{}{
			"user_id":  user.Username,
			"username": user.Username,
			"role":     user.Role,
		}
		token, err := s.sessionManager.Service.Issue(sessionData)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
			return
		}
		// Set session cookie (24 hours)
		s.sessionManager.SetCookieWithExpiry(w, token, 86400)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Logout handles user logout
func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Get session cookie and revoke it
	if s.sessionManager != nil {
		if cookie, err := r.Cookie(s.sessionManager.CookieName); err == nil {
			if s.sessionManager.Service != nil {
				s.sessionManager.Service.Revoke(cookie.Value)
			}
		}
		s.sessionManager.ClearCookie(w)
	}

	writeJSON(w, http.StatusOK, map[string]string{"success": "logged out"})
}

// RefreshToken handles token refresh
func (s *AuthService) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if s.sessionManager == nil || s.sessionManager.Service == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session service not configured"})
		return
	}

	cookie, err := r.Cookie(s.sessionManager.CookieName)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no session"})
		return
	}

	newToken, err := s.sessionManager.Service.Refresh(cookie.Value)
	if err != nil {
		s.sessionManager.ClearCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired"})
		return
	}

	s.sessionManager.SetCookieWithExpiry(w, newToken, 86400)
	writeJSON(w, http.StatusOK, map[string]string{"success": "token refreshed"})
}

// RegisterUser handles user registration
func (s *AuthService) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Role == "" {
		req.Role = "user"
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	if len(req.Password) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 4 characters"})
		return
	}

	// Check if user exists
	s.mu.RLock()
	_, exists := s.users[req.Username]
	s.mu.RUnlock()

	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
		return
	}

	// Add user
	s.AddUser(req.Username, req.Password, req.Role)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"username": req.Username,
		"role":     req.Role,
	})
}

// GetCurrentUser returns the current authenticated user
func (s *AuthService) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	sessionData, ok := weblite.GetSessionContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"session":       sessionData,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
