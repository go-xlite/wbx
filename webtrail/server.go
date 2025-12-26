package webtrail

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// WebTrail represents a backend server component that handles requests
// after they've been proxied from the main server. Routes are registered
// without the proxy prefix since the main server strips it before forwarding.
//
// Example: Main server proxies /api/* to WebTrail -> WebTrail sees /users, /orders, etc.
type WebTrail struct {
	Mux      *mux.Router
	Routes   *wrRoutes
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
}

// NewWebtrail creates a new WebTrail instance with proper routing capabilities
func NewWebtrail() *WebTrail {
	wt := &WebTrail{
		Mux: mux.NewRouter(),
	}
	wt.Routes = &wrRoutes{wt: wt}
	wt.NotFound = http.NotFound
	return wt
}

// NewWebtrailWithBase creates a WebTrail with a base path (for documentation/clarity)
// Note: The base path is NOT used in actual routing, only for helper methods
func NewWebtrailWithBase(basePath string) *WebTrail {
	wt := NewWebtrail()
	wt.PathBase = basePath
	return wt
}

type wrRoutes struct {
	wt *WebTrail
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

// Routes registration methods

// Handle registers an http.Handler for the exact path match
func (r *wrRoutes) Handle(path string, handler http.Handler) {
	r.wt.Mux.Handle(path, handler)
}

// HandleFunc registers a handler function for exact path match
func (r *wrRoutes) HandleFunc(path string, handler http.HandlerFunc) {
	r.wt.Mux.HandleFunc(path, handler)
}

// HandlePath is an alias for HandleFunc for consistency with existing code
func (r *wrRoutes) HandlePath(path string, handler http.HandlerFunc) {
	r.HandleFunc(path, handler)
}

// HandlePathPrefix registers a handler for all paths under the given prefix
// The prefix is automatically stripped before passing to the handler
// Example: HandlePathPrefix("/static/", handler) serves "/static/file.css" as "/file.css"
func (r *wrRoutes) HandlePathPrefix(prefix string, handler http.Handler) {
	// Ensure prefix ends with / for proper matching
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	r.wt.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, handler))
}

// HandlePathPrefixFunc registers a handler function for all paths under the prefix
// The prefix is automatically stripped before passing to the handler
func (r *wrRoutes) HandlePathPrefixFunc(prefix string, handler http.HandlerFunc) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	r.wt.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, handler))
}

// HandlePathPrefixH is a convenience method that accepts a handler function and converts it to http.Handler
// This eliminates the need to wrap with http.HandlerFunc manually
// Example: HandlePathPrefixH("/api/", myHandlerFunc) instead of HandlePathPrefix("/api/", http.HandlerFunc(myHandlerFunc))
func (r *wrRoutes) HandlePathPrefixH(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	r.wt.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.HandlerFunc(handler)))
}

// HandleH is a convenience method that accepts a handler function and converts it to http.Handler
// This eliminates the need to wrap with http.HandlerFunc manually
// Example: HandleH("/api/endpoint", myHandlerFunc) instead of Handle("/api/endpoint", http.HandlerFunc(myHandlerFunc))
func (r *wrRoutes) HandleH(path string, handler func(http.ResponseWriter, *http.Request)) {
	r.wt.Mux.Handle(path, http.HandlerFunc(handler))
}

// HandleMethod registers a handler for a specific HTTP method and path
func (r *wrRoutes) HandleMethod(method, path string, handler http.HandlerFunc) {
	r.wt.Mux.HandleFunc(path, handler).Methods(method)
}

// Common HTTP method helpers

// ANY registers a handler that responds to any HTTP method
func (r *wrRoutes) ANY(path string, handler http.HandlerFunc) {
	r.wt.Mux.HandleFunc(path, handler)
}

func (r *wrRoutes) GET(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodGet, path, handler)
}

func (r *wrRoutes) POST(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodPost, path, handler)
}

func (r *wrRoutes) PUT(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodPut, path, handler)
}

func (r *wrRoutes) PATCH(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodPatch, path, handler)
}

func (r *wrRoutes) DELETE(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodDelete, path, handler)
}

func (r *wrRoutes) OPTIONS(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodOptions, path, handler)
}

func (r *wrRoutes) HEAD(path string, handler http.HandlerFunc) {
	r.HandleMethod(http.MethodHead, path, handler)
}

// GetRoutes returns all registered routes for introspection/debugging
func (r *wrRoutes) GetRoutes() []map[string]string {
	routes := []map[string]string{}
	_ = r.wt.Mux.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()

		methodStr := "ANY"
		if len(methods) > 0 {
			methodStr = strings.Join(methods, ",")
		}

		routes = append(routes, map[string]string{
			"path":    pathTemplate,
			"methods": methodStr,
		})
		return nil
	})
	return routes
}

// SetNotFoundHandler sets a custom 404 handler
func (wt *WebTrail) SetNotFoundHandler(handler http.HandlerFunc) {
	wt.NotFound = handler
	wt.Mux.NotFoundHandler = handler
}
