package middleware

import (
	"context"
	"net/http"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "request_id"

	// UserKey is the context key for user information
	UserKey ContextKey = "user"

	// AuthKey is the context key for authentication context
	AuthKey ContextKey = "auth"
)

// GetFromContext retrieves a value from the request context
func GetFromContext(r *http.Request, key ContextKey) any {
	return r.Context().Value(key)
}

// SetInContext creates a new request with the value set in context
func SetInContext(r *http.Request, key ContextKey, value any) *http.Request {
	ctx := context.WithValue(r.Context(), key, value)
	return r.WithContext(ctx)
}

// WithContext is a middleware that adds a value to the request context
func WithContext(key ContextKey, value any) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = SetInContext(r, key, value)
			next.ServeHTTP(w, r)
		})
	}
}

// WithContextFunc is a middleware that adds a dynamically computed value to context
func WithContextFunc(key ContextKey, fn func(*http.Request) any) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			value := fn(r)
			r = SetInContext(r, key, value)
			next.ServeHTTP(w, r)
		})
	}
}
