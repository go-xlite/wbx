package handlerroot

import (
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// RootHandler is optimized for serving root-level files
// Features: favicon.ico, robots.txt, sitemap.xml, manifest.json, error pages, etc.
type RootHandler struct {
	*handler_role.HandlerRole
	CacheMaxAge     time.Duration
	NotFoundPage    []byte
	ServerErrorPage []byte
}

// NewRootHandler creates a RootHandler wrapper around an existing handler instance
func NewRootHandler(handler handler_role.IHandler) *RootHandler {
	handlerRole := &handler_role.HandlerRole{Handler: handler}
	handlerRole.SetPathPrefix("/")
	return &RootHandler{
		HandlerRole: handlerRole,
		CacheMaxAge: 24 * time.Hour,
	}
}
