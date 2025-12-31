package weblite

import "context"

type sessionContextKey struct{}

// SetSessionContext stores session data in request context
func SetSessionContext(ctx context.Context, sessionData interface{}) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sessionData)
}

// GetSessionContext retrieves session data from request context
func GetSessionContext(ctx context.Context) (interface{}, bool) {
	data := ctx.Value(sessionContextKey{})
	return data, data != nil
}
