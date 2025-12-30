// comm/session.go
package comm

import (
	"context"
	"net/http"
)

// Context keys for session data
type contextKey string

const (
	SessionIDKey contextKey = "session_id"
	UserIDKey    contextKey = "user_id"
	AccountIDKey contextKey = "account_id"
	UserDataKey  contextKey = "user_data"
)

// SessionResolver interface
type SessionResolver interface {
	ResolveSession(r *http.Request) context.Context
	GetUserID(r *http.Request) (int64, bool)
	GetAccountID(r *http.Request) (int64, bool)
	GetSessionID(r *http.Request) (string, bool)
}

// Helper functions to extract values from context
func GetSessionID(r *http.Request) (string, bool) {
	val, ok := r.Context().Value(SessionIDKey).(string)
	return val, ok
}

func GetUserID(r *http.Request) (int64, bool) {
	val, ok := r.Context().Value(UserIDKey).(int64)
	return val, ok
}

func GetAccountID(r *http.Request) (int64, bool) {
	val, ok := r.Context().Value(AccountIDKey).(int64)
	return val, ok
}

func GetUserData(r *http.Request) (any, bool) {
	val := r.Context().Value(UserDataKey)
	return val, val != nil
}

// Setters for building context
func WithSessionID(r *http.Request, sessionID string) context.Context {
	return context.WithValue(r.Context(), SessionIDKey, sessionID)
}

func WithUserID(r *http.Request, userID int64) context.Context {
	return context.WithValue(r.Context(), UserIDKey, userID)
}

func WithAccountID(r *http.Request, accountID int64) context.Context {
	return context.WithValue(r.Context(), AccountIDKey, accountID)
}

func WithUserData(r *http.Request, data any) context.Context {
	return context.WithValue(r.Context(), UserDataKey, data)
}
