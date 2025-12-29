package webtrail

import (
	"net/http"

	"github.com/go-xlite/wbx/comm"
)

type WebTrail struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
}

// NewWebtrail creates a new WebTrail instance with proper routing capabilities
func NewWebtrail() *WebTrail {
	wt := &WebTrail{
		ServerCore: comm.NewServerCore(),
		PathBase:   "",
	}
	wt.NotFound = http.NotFound
	return wt
}

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

func (wt *WebTrail) ServeData(w http.ResponseWriter, r *http.Request) {

}
