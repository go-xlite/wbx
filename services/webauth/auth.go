package webauth

import (
	"net/http"

	"github.com/go-xlite/wbx/comm"
)

type WebAuth struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
}

// NewWebAuth creates a new WebAuth instance with proper routing capabilities
func NewWebAuth() *WebAuth {
	wt := &WebAuth{
		ServerCore: comm.NewServerCore(),
		PathBase:   "",
	}
	wt.NotFound = http.NotFound
	return wt
}

func (wt *WebAuth) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}

// MakePath creates a full path by prepending the PathBase (if set)
// Useful for documentation or when you want to know the full proxied path
func (wt *WebAuth) MakePath(suffix string) string {
	if wt.PathBase == "" {
		return suffix
	}
	return wt.PathBase + suffix
}

func (wt *WebAuth) ServeData(w http.ResponseWriter, r *http.Request) {

}
