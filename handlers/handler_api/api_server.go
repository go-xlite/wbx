package weblite

import (
	"embed"
	"net/http"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/servers/webtrail"
	hl1 "github.com/go-xlite/wbx/utils"
	"github.com/go-xlite/wbx/weblite"
)

//go:embed app-dist/*
var content embed.FS

// ApiHandler is optimized for serving API requests, typically returning JSON data
// Features: JSON serialization, CORS support, request validation, error handling
type ApiHandler struct {
	*handler_role.HandlerRole
	Timeout time.Duration
	trail   *webtrail.WebTrail
}

// NewApiHandler creates a new API handler with sensible defaults
func NewApiHandler(server *webtrail.WebTrail) *ApiHandler {
	sr := handler_role.NewHandler()
	sr.CORS.EnableCORS = true
	sr.CORS.CORSOrigins = []string{"*"}

	return &ApiHandler{
		HandlerRole: sr,
		Timeout:     30 * time.Second,
		trail:       server,
	}
}

func (as *ApiHandler) Run() {
	// No-op for now; could be used to initialize resources if needed
	server := weblite.Provider.Servers.GetByIndex(0)
	if server == nil {
		panic("No WebLite server available to register WebSocket handler")
	}

	server.GetRoutes().ForwardPathPrefixFn(as.PathPrefix.Suffix("p"), func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})

	server.GetRoutes().ForwardPathPrefixFn(as.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		as.trail.OnRequest(w, r)
	})

}
