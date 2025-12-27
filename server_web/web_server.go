package weblite

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	comm "github.com/go-xlite/wbx/comm"
	serverrole "github.com/go-xlite/wbx/server_role"
	weblite "github.com/go-xlite/wbx/weblite"
)

// WebServer is optimized for serving HTML applications with linked assets
// Features: Template rendering, SPA support, asset serving, security headers
type WebServer struct {
	*serverrole.ServerRole
	StaticDir       string
	TemplateDir     string
	Templates       *template.Template
	IndexFile       string
	NotFoundFile    string
	EnableSPA       bool // Single Page Application mode
	SecurityHeaders bool
	CacheMaxAge     time.Duration
}

// NewWebServerFromWebLite creates a WebServer wrapper around an existing WebLite instance
func NewWebServer(wl *weblite.WebLite) *WebServer {
	return &WebServer{
		ServerRole:      &serverrole.ServerRole{Server: wl, PathPrefix: "/"},
		IndexFile:       "index.html",
		NotFoundFile:    "404.html",
		EnableSPA:       false,
		SecurityHeaders: true,
		CacheMaxAge:     1 * time.Hour,
	}
}

// SetStaticDir sets the directory for static files
func (ws *WebServer) SetStaticDir(dir string) *WebServer {
	ws.StaticDir = dir
	return ws
}

// SetTemplateDir sets the directory for templates
func (ws *WebServer) SetTemplateDir(dir string) error {
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

// EnableSPAMode enables Single Page Application mode
// All routes will serve the index file, letting the client-side router handle routing
func (ws *WebServer) EnableSPAMode() *WebServer {
	ws.EnableSPA = true
	return ws
}

// ServeStatic serves static files from a filesystem provider with proper MIME types
func (ws *WebServer) ServeStatic(urlPath string, fsProvider comm.IFsProvider) {
	fullPath := ws.PathPrefix + urlPath

	ws.Server.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
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
		mimeType := ws.ServerRole.GetMimeType(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}

// ServeSPA serves a Single Page Application from a filesystem provider
func (ws *WebServer) ServeSPA(indexPath string, fsProvider comm.IFsProvider) error {
	indexData, err := fsProvider.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	ws.Server.GetRoutes().HandlePathFn(ws.PathPrefix, func(w http.ResponseWriter, r *http.Request) {
		// For SPA mode, serve index for all non-asset paths
		if ws.EnableSPA && !strings.Contains(r.URL.Path, ".") {
			if ws.SecurityHeaders {
				ws.applySecurityHeaders(w)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(indexData)
			return
		}

		// Try to serve the actual file
		path := strings.TrimPrefix(r.URL.Path, ws.PathPrefix)
		if path == "" || path == "/" {
			path = indexPath
		}

		data, err := fsProvider.ReadFile(path)
		if err != nil {
			// If SPA mode and not found, serve index
			if ws.EnableSPA {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(indexData)
			} else {
				http.NotFound(w, r)
			}
			return
		}

		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}
		ws.applyCacheHeaders(w, r)
		ext := filepath.Ext(path)
		w.Header().Set("Content-Type", ws.ServerRole.GetMimeType(ext))
		w.Write(data)
	})

	return nil
}

// ServeHTML serves an HTML file from a filesystem provider
func (ws *WebServer) ServeHTML(path, htmlPath string, fsProvider comm.IFsProvider) error {
	data, err := fsProvider.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	fullPath := ws.PathPrefix + path
	ws.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
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
func (ws *WebServer) RenderTemplate(path, templateName string, dataFunc func(r *http.Request) any) {
	fullPath := ws.PathPrefix + path
	ws.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
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
func (ws *WebServer) HandlePage(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	fullPath := ws.PathPrefix + path
	ws.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}
		handler(w, r)
	})
}

// Redirect creates a redirect from one path to another
func (ws *WebServer) Redirect(fromPath, toPath string, permanent bool) {
	fullPath := ws.PathPrefix + fromPath
	statusCode := http.StatusFound
	if permanent {
		statusCode = http.StatusMovedPermanently
	}

	ws.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, toPath, statusCode)
	})
}

// applySecurityHeaders applies common security headers
func (ws *WebServer) applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// applyCacheHeaders applies caching headers based on content type
func (ws *WebServer) applyCacheHeaders(w http.ResponseWriter, r *http.Request) {
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

// Start starts the web server
func (ws *WebServer) Start() error {
	return nil
}

// Stop stops the web server
func (ws *WebServer) Stop() error {
	return nil
}

// Helper functions for HTML responses

// WriteHTMLTemplate writes an HTML template response
func WriteHTMLTemplate(w http.ResponseWriter, tmpl *template.Template, templateName string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	return tmpl.ExecuteTemplate(w, templateName, data)
}

// WriteHTMLString writes an HTML string response
func WriteHTMLString(w http.ResponseWriter, html string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, err := w.Write([]byte(html))
	return err
}

// WriteRedirect writes a redirect response
func WriteRedirect(w http.ResponseWriter, r *http.Request, url string, permanent bool) {
	statusCode := http.StatusFound
	if permanent {
		statusCode = http.StatusMovedPermanently
	}
	http.Redirect(w, r, url, statusCode)
}
