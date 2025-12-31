package weblite

import (
	"net/http"
	"strings"
	"sync"
)

// SessionService interface for your external session validation/issuing service
type SessionService interface {
	// Validate checks if a session token is valid and returns session data
	Validate(token string) (interface{}, error)
	// Issue creates a new session and returns the token
	Issue(data interface{}) (string, error)
	// Refresh extends an existing session
	Refresh(token string) (string, error)
	// Revoke invalidates a session
	Revoke(token string) error
}

// SessionManager handles session cookie mechanics and path filtering
type SessionManager struct {
	Service      SessionService
	CookieName   string
	CookiePath   string
	CookieDomain string
	Secure       bool // HTTPS only
	HttpOnly     bool
	SameSite     http.SameSite
	SkipPaths    []string // Exact paths to skip
	SkipPrefixes []string // Path prefixes to skip
	mu           sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager(service SessionService) *SessionManager {
	return &SessionManager{
		Service:      service,
		CookieName:   "session",
		CookiePath:   "/",
		HttpOnly:     true,
		Secure:       true,
		SameSite:     http.SameSiteLaxMode,
		SkipPaths:    []string{},
		SkipPrefixes: []string{},
	}
}

// SetSkipPaths sets exact paths to skip session validation
func (sm *SessionManager) SetSkipPaths(paths ...string) *SessionManager {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.SkipPaths = paths
	return sm
}

// SetSkipPrefixes sets path prefixes to skip session validation
func (sm *SessionManager) SetSkipPrefixes(prefixes ...string) *SessionManager {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.SkipPrefixes = prefixes
	return sm
}

// AddSkipPath adds a single path to skip list
func (sm *SessionManager) AddSkipPath(path string) *SessionManager {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.SkipPaths = append(sm.SkipPaths, path)
	return sm
}

// AddSkipPrefix adds a single prefix to skip list
func (sm *SessionManager) AddSkipPrefix(prefix ...string) *SessionManager {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.SkipPrefixes = append(sm.SkipPrefixes, prefix...)
	return sm
}

// ShouldSkip checks if a path should skip session validation
func (sm *SessionManager) ShouldSkip(path string) bool {
	sm.mu.RLock()
	skipPaths := sm.SkipPaths
	skipPrefixes := sm.SkipPrefixes
	sm.mu.RUnlock()

	// Check exact paths (no lock held)
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	// Check prefixes (no lock held)
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// Middleware creates HTTP middleware for session handling
func (sm *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should skip session validation
		if sm.ShouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Try to get session cookie
		cookie, err := r.Cookie(sm.CookieName)
		if err != nil {
			// No session cookie - return unauthorized
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate session with your service
		sessionData, err := sm.Service.Validate(cookie.Value)
		if err != nil {
			// Invalid session - clear cookie and return unauthorized
			sm.ClearCookie(w)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Store session data in request context for handlers to use
		ctx := SetSessionContext(r.Context(), sessionData)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetCookie sets the session cookie
func (sm *SessionManager) SetCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     sm.CookieName,
		Value:    token,
		Path:     sm.CookiePath,
		Domain:   sm.CookieDomain,
		Secure:   sm.Secure,
		HttpOnly: sm.HttpOnly,
		SameSite: sm.SameSite,
		MaxAge:   0, // Session cookie (expires when browser closes)
	}
	http.SetCookie(w, cookie)
}

// SetCookieWithExpiry sets the session cookie with an expiration time
func (sm *SessionManager) SetCookieWithExpiry(w http.ResponseWriter, token string, maxAge int) {
	cookie := &http.Cookie{
		Name:     sm.CookieName,
		Value:    token,
		Path:     sm.CookiePath,
		Domain:   sm.CookieDomain,
		Secure:   sm.Secure,
		HttpOnly: sm.HttpOnly,
		SameSite: sm.SameSite,
		MaxAge:   maxAge,
	}
	http.SetCookie(w, cookie)
}

// ClearCookie removes the session cookie
func (sm *SessionManager) ClearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     sm.CookieName,
		Value:    "",
		Path:     sm.CookiePath,
		Domain:   sm.CookieDomain,
		MaxAge:   -1,
		Secure:   sm.Secure,
		HttpOnly: sm.HttpOnly,
	}
	http.SetCookie(w, cookie)
}
