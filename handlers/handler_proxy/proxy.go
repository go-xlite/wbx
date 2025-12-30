package handlerproxy

import (
	"net/http"
	"strings"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/servers/webproxy"
)

// ProxyHandler provides reverse proxy functionality
// This is a thin wrapper that delegates to the webproxy server
type ProxyHandler struct {
	*handler_role.HandlerRole
	webproxy *webproxy.Webproxy
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(wp *webproxy.Webproxy) *ProxyHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.Handler = wp

	return &ProxyHandler{
		HandlerRole: handlerRole,
		webproxy:    wp,
	}
}

// HandleProxy creates an HTTP handler for the proxy
func (ph *ProxyHandler) HandleProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path from URL
		path := strings.TrimPrefix(r.URL.Path, ph.PathPrefix.Get())
		path = strings.TrimPrefix(path, "/")

		// Set the cleaned path
		r.URL.Path = "/" + path

		// Delegate to webproxy
		ph.webproxy.OnRequest(w, r)
	}
}
