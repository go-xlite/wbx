package handlercdn

import (
	"net/http"

	"github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/services/webcdn"
)

// CdnHandler is a lightweight handler wrapper for WebCdn
type CdnHandler struct {
	*handler_role.HandlerRole
	webcdn *webcdn.WebCdn
}

// NewCdnHandler creates a new CdnHandler
func NewCdnHandler(cdn *webcdn.WebCdn) *CdnHandler {
	handlerRole := handler_role.NewHandler()
	return &CdnHandler{
		HandlerRole: handlerRole,
		webcdn:      cdn,
	}
}

// Run registers the CDN handler routes
func (ch *CdnHandler) Run() {
	ch.Handler.GetRoutes().ForwardPathPrefixFn(ch.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		ch.webcdn.OnRequest(w, r)
	})
}

// GetWebCdn returns the underlying WebCdn instance for direct configuration
func (ch *CdnHandler) GetWebCdn() *webcdn.WebCdn {
	return ch.webcdn
}
