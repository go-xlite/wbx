package websvc

import (
	"net/http"

	"github.com/go-xlite/wbx/comm"
)

type WebSvc struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
}

// NewWebSvc creates a new WebSvc instance with proper routing capabilities
func NewWebSvc() *WebSvc {
	wt := &WebSvc{
		ServerCore: comm.NewServerCore(),
		PathBase:   "",
	}
	wt.NotFound = http.NotFound
	return wt
}

func (wt *WebSvc) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}
