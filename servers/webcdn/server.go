package webcdn

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-xlite/wbx/comm"
	"github.com/go-xlite/wbx/comm/mime"
)

type AssetRequest struct {
	Path string
	W    http.ResponseWriter
	R    *http.Request
}

type WebCdn struct {
	*comm.ServerCore
	PathBase      string // Optional base path for convenience (e.g., "/cdn" for documentation)
	NotFound      http.HandlerFunc
	CacheMaxAge   time.Duration
	EnableBrowser bool // Allow browser caching
	EnableETags   bool
}

// NewWebCdn creates a new WebCdn instance with proper routing capabilities
func NewWebCdn() *WebCdn {
	wt := &WebCdn{
		ServerCore:    comm.NewServerCore(),
		PathBase:      "",
		CacheMaxAge:   24 * time.Hour,
		EnableBrowser: true,
		EnableETags:   true,
	}
	wt.NotFound = http.NotFound
	return wt
}

// SetCaching configures caching behavior
func (wt *WebCdn) SetCaching(maxAge time.Duration, enableBrowser bool) *WebCdn {
	wt.CacheMaxAge = maxAge
	wt.EnableBrowser = enableBrowser
	return wt
}

// OnRequest handles an incoming HTTP request using the registered routes
func (wt *WebCdn) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}

// HandleResponse sends data with proper CDN headers
func (wt *WebCdn) HandleResponse(assetReq *AssetRequest, data []byte, mimeType string) {
	wt.applyCacheHeaders(assetReq.W)
	assetReq.W.Header().Set("Content-Type", mimeType)
	assetReq.W.Write(data)
}

// ServeFile serves files from a filesystem provider
func (wt *WebCdn) ServeFile(urlPath string, fsProvider comm.IFsAdapter) {
	wt.GetRoutes().HandlePathPrefixFn(urlPath, func(w http.ResponseWriter, r *http.Request) {
		relativePath := r.URL.Path
		if relativePath == "" || relativePath == "/" {
			wt.NotFound(w, r)
			return
		}

		// Read file from filesystem provider
		data, err := fsProvider.ReadFile(relativePath)
		if err != nil {
			wt.NotFound(w, r)
			return
		}

		// Apply caching and MIME type
		wt.applyCacheHeaders(w)
		ext := filepath.Ext(relativePath)
		w.Header().Set("Content-Type", mime.GetMimeType(ext))
		w.Write(data)
	})
}

// ServeBytes serves raw bytes with specified MIME type
func (wt *WebCdn) ServeBytes(urlPath string, data []byte, mimeType string) {
	wt.GetRoutes().HandlePathFn(urlPath, func(w http.ResponseWriter, r *http.Request) {
		wt.applyCacheHeaders(w)
		w.Header().Set("Content-Type", mimeType)
		w.Write(data)
	})
}

// HandlePrefix registers a custom handler for a path prefix
func (wt *WebCdn) HandlePrefix(path string, handlerFunc func(assetReq *AssetRequest)) {
	wt.GetRoutes().HandlePathPrefixFn(path, func(w http.ResponseWriter, r *http.Request) {
		assetReq := &AssetRequest{
			Path: r.URL.Path,
			R:    r,
			W:    w,
		}
		handlerFunc(assetReq)
	})
}

// applyCacheHeaders applies appropriate caching headers
func (wt *WebCdn) applyCacheHeaders(w http.ResponseWriter) {
	if wt.EnableBrowser {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(wt.CacheMaxAge.Seconds())))
		w.Header().Set("Expires", time.Now().Add(wt.CacheMaxAge).Format(http.TimeFormat))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// MakePath creates a full path by prepending the PathBase (if set)
func (wt *WebCdn) MakePath(suffix string) string {
	if wt.PathBase == "" {
		return suffix
	}
	return wt.PathBase + suffix
}
