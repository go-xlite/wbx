package webtrail

import (
	"net/http"

	"github.com/go-xlite/wbx/routes"
	"github.com/gorilla/mux"
)

// WebTrail represents a backend server component that handles requests
// after they've been proxied from the main server. Routes are registered
// without the proxy prefix since the main server strips it before forwarding.
//
// Example: Main server proxies /api/* to WebTrail -> WebTrail sees /users, /orders, etc.
type WebTrail struct {
	Mux    *mux.Router
	Routes *routes.Routes
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
}

// NewWebtrail creates a new WebTrail instance with proper routing capabilities
func NewWebtrail() *WebTrail {
	wt := &WebTrail{
		Mux:      mux.NewRouter(),
		PathBase: "",
	}
	wt.Routes = routes.NewRoutes(wt.Mux, 1)
	wt.NotFound = http.NotFound
	return wt
}

// OnRequest handles an incoming HTTP request using the registered routes
// This is the main entry point when the main server forwards a request
func (wt *WebTrail) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}

// MakePath creates a full path by prepending the PathBase (if set)
// Useful for documentation or when you want to know the full proxied path
func (wt *WebTrail) MakePath(suffix string) string {
	if wt.PathBase == "" {
		return suffix
	}
	return wt.PathBase + suffix
}

// GetRoutes returns the Routes instance
func (wt *WebTrail) GetRoutes() *routes.Routes {
	return wt.Routes
}

// GetMux returns the mux.Router instance
func (wt *WebTrail) GetMux() *mux.Router {
	return wt.Mux
}

// SetNotFoundHandler sets a custom 404 handler
func (wt *WebTrail) SetNotFoundHandler(handler http.HandlerFunc) {
	wt.NotFound = handler
	wt.Mux.NotFoundHandler = handler
}
