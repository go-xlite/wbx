package routes

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Routes provides standardized routing methods for mux.Router
type Routes struct {
	Mux  *mux.Router
	mode int // If 1, strips prefix before passing to handler (webtrail mode). If 0, passes full path (weblite mode)
}

// NewRoutes creates a new Routes instance
func NewRoutes(mux *mux.Router, mode int) *Routes {
	return &Routes{
		Mux:  mux,
		mode: mode, // Default to weblite mode (no stripping)
	}
}

// SetStripPrefix sets whether to strip prefix from paths before passing to handlers
// Use true for webtrail mode, false for weblite mode (default)

// HandlePathH registers an http.Handler for the exact path match
func (r *Routes) HandlePathH(pattern string, handler http.Handler) {
	r.Mux.Handle(pattern, handler)
}

// HandlePathFn registers a handler function for exact path match
func (r *Routes) HandlePathFn(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.Mux.HandleFunc(pattern, handler)
}

// HandlePathPrefixH registers a handler for all paths under the given prefix
// The prefix is automatically stripped from the request path before passing to the handler
// Example: HandlePathPrefixH("/static/", handler) will serve "/static/file.css" as "/file.css" to the handler
func (r *Routes) HandlePathPrefixH(prefix string, handler http.Handler) {
	r.handlePathPrefixWithMethod(prefix, handler)
}

// HandlePathPrefixFn registers a handler function for all paths under the prefix
// The prefix is automatically stripped before passing to the handler
func (r *Routes) HandlePathPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler)
}

// HandlePathFnc is a convenience method that accepts a raw handler function and converts it to http.Handler
// This eliminates the need to wrap with http.HandlerFunc manually
// Example: HandlePathFnc("/api/endpoint", myFunc) instead of HandlePathH("/api/endpoint", http.HandlerFunc(myFunc))
func (r *Routes) HandlePathFnc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.Mux.Handle(pattern, http.HandlerFunc(handler))
}

// HandlePathPrefixFnc is a convenience method that accepts a raw handler function for path prefixes
// The prefix is automatically stripped before passing to the handler
// Example: HandlePathPrefixFnc("/api/", myFunc) instead of HandlePathPrefixH("/api/", http.HandlerFunc(myFunc))
func (r *Routes) HandlePathPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler))
}

func (r *Routes) ForwardPathFn(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	// Wrap the handler to strip the base path and preserve original path in header
	wrappedHandler := func(w http.ResponseWriter, req *http.Request) {
		// Save original path in header if not already set
		if req.Header.Get("X-Original-Path") == "" {
			req.Header.Set("X-Original-Path", req.URL.Path)
		}

		// Strip the pattern from the path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, pattern)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}

		handler(w, req)
	}

	r.Mux.HandleFunc(pattern, wrappedHandler)
}

func (r *Routes) ForwardPathPrefixFn(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	// Normalize prefix
	prefix = r.normalizePrefix(prefix)

	// Wrap the handler to strip the prefix and preserve original path in header
	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Save original path in header if not already set
		if req.Header.Get("X-Original-Path") == "" {
			req.Header.Set("X-Original-Path", req.URL.Path)
		}

		// Strip the prefix from the path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, strings.TrimSuffix(prefix, "/"))
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}

		handler(w, req)
	})

	r.Mux.PathPrefix(prefix).Handler(wrappedHandler)
}

// Internal helpers for standardizing prefix handling

func (r *Routes) normalizePrefix(prefix string) string {
	if !strings.HasSuffix(prefix, "/") {
		return prefix + "/"
	}
	return prefix
}

func (r *Routes) handlePathPrefixWithMethod(prefix string, handler http.Handler, methods ...string) {
	prefix = r.normalizePrefix(prefix)
	var route *mux.Route
	if r.mode == 1 {
		// Webtrail mode: strip prefix before passing to handler
		route = r.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, handler))
	} else {
		// Weblite mode: pass full path to handler
		route = r.Mux.PathPrefix(prefix).Handler(handler)
	}
	if len(methods) > 0 {
		route.Methods(methods...)
	}
}

// GetRoutes returns all registered routes with their methods
func (r *Routes) GetRoutes() []map[string]string {
	routes := []map[string]string{}

	err := r.Mux.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			// If GetPathTemplate fails, try GetPathRegexp
			pathRegexp, _ := route.GetPathRegexp()
			if pathRegexp != "" {
				pathTemplate = pathRegexp
			}
		}

		// Skip empty paths
		if pathTemplate == "" {
			return nil
		}

		methods, _ := route.GetMethods()

		methodStr := "ANY"
		if len(methods) > 0 {
			methodStr = strings.Join(methods, ", ")
		}

		routes = append(routes, map[string]string{
			"path":    pathTemplate,
			"methods": methodStr,
		})
		return nil
	})

	// If Walk returns an error or no routes found, return empty slice
	if err != nil || len(routes) == 0 {
		return []map[string]string{}
	}

	return routes
}

// Common HTTP method helpers

// GETPathFn registers a GET handler for exact path match
func (r *Routes) GETPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodGet)
}

// GETPrefixFn registers a GET handler for path prefix with http.HandlerFunc
func (r *Routes) GETPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodGet)
}

// GETPrefixFnc registers a GET handler for path prefix with raw function
func (r *Routes) GETPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodGet)
}

// POSTPathFn registers a POST handler for exact path match
func (r *Routes) POSTPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodPost)
}

// POSTPrefixFn registers a POST handler for path prefix with http.HandlerFunc
func (r *Routes) POSTPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodPost)
}

// POSTPrefixFnc registers a POST handler for path prefix with raw function
func (r *Routes) POSTPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodPost)
}

// PUTPathFn registers a PUT handler for exact path match
func (r *Routes) PUTPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodPut)
}

// PUTPrefixFn registers a PUT handler for path prefix with http.HandlerFunc
func (r *Routes) PUTPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodPut)
}

// PUTPrefixFnc registers a PUT handler for path prefix with raw function
func (r *Routes) PUTPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodPut)
}

// PATCHPathFn registers a PATCH handler for exact path match
func (r *Routes) PATCHPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodPatch)
}

// PATCHPrefixFn registers a PATCH handler for path prefix with http.HandlerFunc
func (r *Routes) PATCHPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodPatch)
}

// PATCHPrefixFnc registers a PATCH handler for path prefix with raw function
func (r *Routes) PATCHPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodPatch)
}

// DELETEPathFn registers a DELETE handler for exact path match
func (r *Routes) DELETEPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodDelete)
}

// DELETEPrefixFn registers a DELETE handler for path prefix with http.HandlerFunc
func (r *Routes) DELETEPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodDelete)
}

// DELETEPrefixFnc registers a DELETE handler for path prefix with raw function
func (r *Routes) DELETEPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodDelete)
}

// OPTIONSPathFn registers an OPTIONS handler for exact path match
func (r *Routes) OPTIONSPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodOptions)
}

// OPTIONSPrefixFn registers an OPTIONS handler for path prefix with http.HandlerFunc
func (r *Routes) OPTIONSPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodOptions)
}

// OPTIONSPrefixFnc registers an OPTIONS handler for path prefix with raw function
func (r *Routes) OPTIONSPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodOptions)
}

// HEADPathFn registers a HEAD handler for exact path match
func (r *Routes) HEADPathFn(path string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(path, handler).Methods(http.MethodHead)
}

// HEADPrefixFn registers a HEAD handler for path prefix with http.HandlerFunc
func (r *Routes) HEADPrefixFn(prefix string, handler http.HandlerFunc) {
	r.handlePathPrefixWithMethod(prefix, handler, http.MethodHead)
}

// HEADPrefixFnc registers a HEAD handler for path prefix with raw function
func (r *Routes) HEADPrefixFnc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	r.handlePathPrefixWithMethod(prefix, http.HandlerFunc(handler), http.MethodHead)
}
