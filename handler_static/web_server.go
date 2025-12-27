package weblite

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	comm "github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// WebHandler is optimized for serving HTML applications with linked assets
// Features: Template rendering, asset serving, security headers
type WebHandler struct {
	*handler_role.HandlerRole
	StaticDir       string
	TemplateDir     string
	Templates       *template.Template
	IndexFile       string
	NotFoundFile    string
	SecurityHeaders bool
	CacheMaxAge     time.Duration
}

// NewWebHandler creates a WebHandler wrapper around an existing handler instance
func NewWebHandler(wl handler_role.IHandler) *WebHandler {
	return &WebHandler{
		HandlerRole:     &handler_role.HandlerRole{Handler: wl, PathPrefix: "/"},
		IndexFile:       "index.html",
		NotFoundFile:    "404.html",
		SecurityHeaders: true,
		CacheMaxAge:     1 * time.Hour,
	}
}

// SetStaticDir sets the directory for static files
func (ws *WebHandler) SetStaticDir(dir string) *WebHandler {
	ws.StaticDir = dir
	return ws
}

// SetTemplateDir sets the directory for templates
func (ws *WebHandler) SetTemplateDir(dir string) error {
	ws.TemplateDir = dir

	// Load templates
	pattern := filepath.Join(dir, "*.html")
	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	ws.Templates = tmpl
	return nil
}

// ServeStatic serves static files from a filesystem provider with proper MIME types
func (ws *WebHandler) ServeStatic(urlPath string, fsProvider comm.IFsAdapter) {
	fullPath := ws.PathPrefix + urlPath

	ws.Handler.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Path
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
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}

		// Apply caching
		ws.applyCacheHeaders(w, r)

		// Set MIME type based on extension
		ext := filepath.Ext(filePath)
		mimeType := ws.HandlerRole.GetMimeType(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}

// ServeHTML serves an HTML file from a filesystem provider
func (ws *WebHandler) ServeHTML(path, htmlPath string, fsProvider comm.IFsAdapter) error {
	data, err := fsProvider.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	fullPath := ws.PathPrefix + path
	ws.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	})

	return nil
}

// RenderTemplate renders a template with the given data
func (ws *WebHandler) RenderTemplate(path, templateName string, dataFunc func(r *http.Request) any) {
	fullPath := ws.PathPrefix + path
	ws.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}

		var data any
		if dataFunc != nil {
			data = dataFunc(r)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")

		if err := ws.Templates.ExecuteTemplate(w, templateName, data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	})
}

// HandlePage handles a page request with custom logic
func (ws *WebHandler) HandlePage(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	fullPath := ws.PathPrefix + path
	ws.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}
		handler(w, r)
	})
}

// applySecurityHeaders applies common security headers
func (ws *WebHandler) applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// applyCacheHeaders applies caching headers based on content type
func (ws *WebHandler) applyCacheHeaders(w http.ResponseWriter, r *http.Request) {
	ext := filepath.Ext(r.URL.Path)

	// HTML should not be cached
	if ext == ".html" || ext == ".htm" || ext == "" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}
	if comm.IsStaticExtension(ext) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ws.CacheMaxAge.Seconds())))
		return
	}
	// Default: no cache
	w.Header().Set("Cache-Control", "no-cache")
}
