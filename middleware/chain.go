package middleware

import "net/http"

// Middleware wraps an http.Handler with additional functionality
type Middleware func(http.Handler) http.Handler

// Chain allows composing multiple middleware
type Chain struct {
	middlewares []Middleware
}

// New creates a new middleware chain
func New(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Use adds a middleware to the chain
func (c *Chain) Use(mw Middleware) *Chain {
	c.middlewares = append(c.middlewares, mw)
	return c
}

// Then wraps the final handler with all middleware in the chain
// Middleware are applied in the order they were added
func (c *Chain) Then(h http.Handler) http.Handler {
	// Apply middleware in reverse order so they execute in the correct order
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i](h)
	}
	return h
}

// ThenFunc is a convenience method for wrapping an http.HandlerFunc
func (c *Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
	return c.Then(fn)
}

// Append creates a new chain with additional middleware
func (c *Chain) Append(middlewares ...Middleware) *Chain {
	newChain := &Chain{
		middlewares: make([]Middleware, len(c.middlewares)+len(middlewares)),
	}
	copy(newChain.middlewares, c.middlewares)
	copy(newChain.middlewares[len(c.middlewares):], middlewares)
	return newChain
}
