package handler_auth

import (
	"embed"
	"net/http"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/services/webauth"
	hl1 "github.com/go-xlite/wbx/utils"
	"github.com/go-xlite/wbx/weblite"
)

//go:embed app-dist/*
var content embed.FS

// AuthHandler is optimized for serving API requests, typically returning JSON data
// Features: JSON serialization, CORS support, request validation, error handling
type AuthHandler struct {
	*handler_role.HandlerRole
	Timeout time.Duration
	auth    *webauth.WebAuth
}

// NewAuthHandler creates a new Auth handler with sensible defaults
func NewAuthHandler(auth *webauth.WebAuth) *AuthHandler {
	sr := handler_role.NewHandler()
	sr.CORS.EnableCORS = true
	sr.CORS.CORSOrigins = []string{"*"}

	return &AuthHandler{
		HandlerRole: sr,
		Timeout:     30 * time.Second,
		auth:        auth,
	}
}

func (as *AuthHandler) Run() {
	// No-op for now; could be used to initialize resources if needed
	server := weblite.Provider.Servers.GetByIndex(0)
	if server == nil {
		panic("No WebLite server available to register WebSocket handler")
	}

	server.GetRoutes().ForwardPathPrefixFn("/m/xlite/auth/p", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})

	server.GetRoutes().ForwardPathPrefixFn("/g/xt23/auth", func(w http.ResponseWriter, r *http.Request) {
		as.auth.OnRequest(w, r)
	})

}
