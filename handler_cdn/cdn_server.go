package weblite

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// CdnServer is optimized for serving CDN/static content
// Features: Automatic MIME types, caching headers, compression, embedded FS support
type CdnHandler struct {
	*handler_role.HandlerRole
	CacheMaxAge   time.Duration
	EnableBrowser bool // Allow browser caching
	EnableETags   bool
}

// NewCdnHandlerFromWebLite wraps an existing WebLite with CdnHandler functionality
func NewCdnHandler(wl handler_role.IHandler) *CdnHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.SetPathPrefix("/cdn")
	return &CdnHandler{
		HandlerRole:   handlerRole,
		CacheMaxAge:   24 * time.Hour,
		EnableBrowser: true,
		EnableETags:   true,
	}
}

// SetCaching configures caching behavior
func (cs *CdnHandler) SetCaching(maxAge time.Duration, enableBrowser bool) *CdnHandler {
	cs.CacheMaxAge = maxAge
	cs.EnableBrowser = enableBrowser
	return cs
}

// ServeFile serves files from a filesystem provider
func (cs *CdnHandler) ServeFile(urlPath string, fsProvider comm.IFsAdapter) {
	fullPath := cs.PathPrefix.Get() + urlPath

	cs.Handler.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		relativePath := r.URL.Path
		if relativePath == "" || relativePath == "/" {
			http.NotFound(w, r)
			return
		}

		// Read file from filesystem provider (embedPath is handled inside the provider)
		data, err := fsProvider.ReadFile(relativePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Apply caching and MIME type
		cs.applyCacheHeaders(w)
		ext := filepath.Ext(relativePath)
		w.Header().Set("Content-Type", cs.HandlerRole.GetMimeType(ext))
		w.Write(data)
	})
}

// ServeBytes serves raw bytes with specified MIME type
func (cs *CdnHandler) ServeBytes(urlPath string, data []byte, mimeType string) {
	fullPath := cs.PathPrefix.Get() + urlPath
	cs.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		cs.applyCacheHeaders(w)
		w.Header().Set("Content-Type", mimeType)
		w.Write(data)
	})
}

// applyCacheHeaders applies appropriate caching headers
func (cs *CdnHandler) applyCacheHeaders(w http.ResponseWriter) {

	if cs.EnableBrowser {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cs.CacheMaxAge.Seconds())))
		w.Header().Set("Expires", time.Now().Add(cs.CacheMaxAge).Format(http.TimeFormat))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}
