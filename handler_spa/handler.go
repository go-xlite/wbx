package handlerspa

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	comm "github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// SPAHandler is optimized for serving Single Page Applications
// Features: Client-side routing support, asset serving, security headers
type SPAHandler struct {
	*handler_role.HandlerRole
	IndexFile       string
	SecurityHeaders bool
	CacheMaxAge     time.Duration
}

// NewSPAHandler creates a SPAHandler wrapper around an existing handler instance
func NewSPAHandler(handler handler_role.IHandler) *SPAHandler {
	handlerRole := &handler_role.HandlerRole{Handler: handler}
	handlerRole.SetPathPrefix("/")
	return &SPAHandler{
		HandlerRole:     handlerRole,
		IndexFile:       "index.html",
		SecurityHeaders: true,
		CacheMaxAge:     1 * time.Hour,
	}
}

// SetIndexFile sets the index file for the SPA
func (sh *SPAHandler) SetIndexFile(indexFile string) *SPAHandler {
	sh.IndexFile = indexFile
	return sh
}

// SetSecurityHeaders enables/disables security headers
func (sh *SPAHandler) SetSecurityHeaders(enabled bool) *SPAHandler {
	sh.SecurityHeaders = enabled
	return sh
}

// SetCacheMaxAge sets the cache duration for static assets
func (sh *SPAHandler) SetCacheMaxAge(duration time.Duration) *SPAHandler {
	sh.CacheMaxAge = duration
	return sh
}

// ServeSPA serves a Single Page Application from a filesystem provider
// All non-asset routes will serve the index file, letting the client-side router handle routing
func (sh *SPAHandler) ServeSPA(fsProvider comm.IFsAdapter) error {
	return sh.ServeSPAWithIndex(sh.IndexFile, fsProvider)
}

// ServeSPAWithIndex serves a Single Page Application with a custom index path
func (sh *SPAHandler) ServeSPAWithIndex(indexPath string, fsProvider comm.IFsAdapter) error {
	indexData, err := fsProvider.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	sh.Handler.GetRoutes().HandlePathPrefixFn(sh.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		// For SPA mode, serve index for all non-asset paths
		if !strings.Contains(r.URL.Path, ".") {
			if sh.SecurityHeaders {
				sh.applySecurityHeaders(w)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(indexData)
			return
		}

		// Try to serve the actual file (assets like .js, .css, .png, etc.)
		path := strings.TrimPrefix(r.URL.Path, sh.PathPrefix.Get())
		if path == "" || path == "/" {
			path = indexPath
		}

		data, err := fsProvider.ReadFile(path)
		if err != nil {
			// If asset not found, serve index for SPA routing
			if sh.SecurityHeaders {
				sh.applySecurityHeaders(w)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(indexData)
			return
		}

		if sh.SecurityHeaders {
			sh.applySecurityHeaders(w)
		}
		sh.applyCacheHeaders(w, path)
		ext := filepath.Ext(path)
		mimeType := sh.HandlerRole.GetMimeType(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)
		w.Write(data)
	})

	return nil
}

// ServeStatic serves static files from a filesystem provider with proper MIME types
// This is useful for serving assets alongside the SPA
func (sh *SPAHandler) ServeStatic(urlPath string, fsProvider comm.IFsAdapter) {
	fullPath := sh.PathPrefix.Get() + urlPath

	sh.Handler.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		filePath := strings.TrimPrefix(r.URL.Path, fullPath)
		if filePath == "" || filePath == "/" {
			filePath = "index.html"
		}

		// Read file from filesystem provider
		data, err := fsProvider.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Apply security headers
		if sh.SecurityHeaders {
			sh.applySecurityHeaders(w)
		}

		// Apply caching
		sh.applyCacheHeaders(w, filePath)

		// Set MIME type based on extension
		ext := filepath.Ext(filePath)
		mimeType := sh.HandlerRole.GetMimeType(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}

// applySecurityHeaders applies common security headers
func (sh *SPAHandler) applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// applyCacheHeaders applies caching headers based on content type
func (sh *SPAHandler) applyCacheHeaders(w http.ResponseWriter, filePath string) {
	ext := filepath.Ext(filePath)

	// HTML should not be cached
	if ext == ".html" || ext == ".htm" || ext == "" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}

	// Static assets can be cached
	if comm.IsStaticExtension(ext) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(sh.CacheMaxAge.Seconds())))
		return
	}

	// Default: no cache
	w.Header().Set("Cache-Control", "no-cache")
}
