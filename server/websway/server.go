package websway

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-xlite/wbx/comm"
	hl1 "github.com/go-xlite/wbx/helpers"
)

// WebSway represents a backend server component that handles requests
// after they've been proxied from the main server. Routes are registered
// without the proxy prefix since the main server strips it before forwarding.
//
// Example: Main server proxies /api/* to WebSway -> WebSway sees /users, /orders, etc.
type WebSway struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase          string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound          http.HandlerFunc
	FsProvider        comm.IFsAdapter
	SecurityHeaders   bool
	CacheMaxAge       time.Duration
	VirtualDirSegment string // Virtual directory segment (default: "p")
	DefaultRoute      string // Default route for root path
}

// NewWebSway creates a new WebSway instance with proper routing capabilities
func NewWebSway() *WebSway {
	wt := &WebSway{
		ServerCore:        comm.NewServerCore(),
		PathBase:          "",
		SecurityHeaders:   true,
		CacheMaxAge:       1 * time.Hour,
		VirtualDirSegment: "p",
		DefaultRoute:      "index",
	}
	wt.NotFound = http.NotFound
	return wt
}

// OnRequest handles an incoming HTTP request using the registered routes
// This is the main entry point when the main server forwards a request
func (wt *WebSway) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}

// MakePath creates a full path by prepending the PathBase (if set)
// Useful for documentation or when you want to know the full proxied path
func (wt *WebSway) MakePath(suffix string) string {
	if wt.PathBase == "" {
		return suffix
	}
	return wt.PathBase + suffix
}

// SetNotFoundHandler sets a custom 404 handler
func (wt *WebSway) SetNotFoundHandler(handler http.HandlerFunc) {
	wt.NotFound = handler
	wt.Mux.NotFoundHandler = handler
}

// ExtractStoragePath converts a request path to a storage path
// It handles the /p/ virtual directory pattern where:
// - URL: example.com/index/p/app.js -> Storage: index/app.js
// - URL: example.com/index/p/ -> Storage: index/index.html
// - URL: example.com/index/p/nested1/nested2/app.js -> Storage: index/nested1/nested2/app.js
// Returns an error if the path contains invalid or malicious patterns
func (wt *WebSway) ExtractStoragePath(requestPath, urlPath, pathPrefix string) (string, error) {
	// Remove the handler's PathPrefix if set (e.g., "/api" or "api")
	relativePath := requestPath
	if pathPrefix != "" {
		if !strings.HasPrefix(requestPath, pathPrefix) {
			return "", fmt.Errorf("path does not start with PathPrefix")
		}
		relativePath = requestPath[len(pathPrefix):]
	}

	// Determine the app directory (storage directory)
	var appDir string

	if urlPath == "/" {
		// Serving from root - extract app directory from first path segment
		if relativePath == "" || relativePath == "/" {
			// Root path, use default
			appDir = wt.DefaultRoute
			relativePath = "/"
		} else {
			// Extract first segment
			pathParts := strings.SplitN(strings.TrimPrefix(relativePath, "/"), "/", 2)
			firstSegment := pathParts[0]

			// Check if first segment is the virtual directory (e.g., "p")
			if firstSegment == wt.VirtualDirSegment {
				// Path like /p/app.js - use default app directory
				appDir = wt.DefaultRoute
			} else {
				// First segment is the app directory (e.g., "home", "index", "about")
				appDir = firstSegment
				if len(pathParts) > 1 {
					relativePath = "/" + pathParts[1]
				} else {
					relativePath = "/"
				}
			}
		}
	} else {
		// Extract app directory from urlPath (e.g., "/index" -> "index")
		appDir = strings.TrimPrefix(urlPath, "/")

		// Remove the urlPath from relativePath
		if !strings.HasPrefix(relativePath, urlPath) {
			return "", fmt.Errorf("path does not start with urlPath")
		}
		relativePath = strings.TrimPrefix(relativePath, urlPath)
	}

	if relativePath == "" {
		relativePath = "/"
	}
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	relativePath = filepath.Clean(relativePath)

	// Check if the path contains virtual directory segment (e.g., /p/)
	virtualSegment := "/" + wt.VirtualDirSegment
	virtualSegmentLen := len(virtualSegment)
	if len(relativePath) >= virtualSegmentLen && relativePath[:virtualSegmentLen] == virtualSegment {
		if len(relativePath) == virtualSegmentLen || (len(relativePath) > virtualSegmentLen && relativePath[virtualSegmentLen] == '/') {
			// Strip the virtual segment part to get the actual storage path
			relativePath = relativePath[virtualSegmentLen:] // Remove virtual segment
			if relativePath == "" || relativePath == "/" {
				relativePath = "/index.html"
			}
		}
	}

	// If no file specified, serve index.html
	if relativePath == "" || relativePath == "/" {
		relativePath = "/index.html"
	}

	// Build storage path: {appDir}{relativePath}
	cleanRelativePath := strings.TrimPrefix(relativePath, "/")

	storagePath := filepath.Join(appDir, cleanRelativePath)
	storagePath = filepath.Clean(storagePath)

	// Validate path to prevent directory traversal attacks
	if filepath.IsAbs(storagePath) {
		return "", fmt.Errorf("invalid path: absolute path not allowed")
	}

	// Check for path traversal attempts
	if len(storagePath) >= 2 && storagePath[0:2] == ".." {
		return "", fmt.Errorf("invalid path: directory traversal not allowed")
	}

	// Ensure path doesn't contain "../" sequences after cleaning
	if filepath.Dir(storagePath) != filepath.Clean(filepath.Dir(storagePath)) {
		return "", fmt.Errorf("invalid path: malformed path")
	}

	// Handle root path and directories as index.html
	if storagePath == "." || storagePath == "/" || storagePath == "" {
		storagePath = filepath.Join(urlPath, "index.html")
	}

	return storagePath, nil
}

// ApplySecurityHeaders applies common security headers
func (wt *WebSway) ApplySecurityHeaders(w http.ResponseWriter) {
	if !wt.SecurityHeaders {
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// ApplyCacheHeaders applies caching headers based on content type
func (wt *WebSway) ApplyCacheHeaders(w http.ResponseWriter, requestPath string) {
	ext := filepath.Ext(requestPath)

	// HTML should not be cached
	if ext == ".html" || ext == ".htm" || ext == "" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}
	if comm.Mime.IsStaticExtension(ext) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(wt.CacheMaxAge.Seconds())))
		return
	}
	// Default: no cache
	w.Header().Set("Cache-Control", "no-cache")
}

// ServeFile serves a single file from the filesystem with proper headers and MIME type
func (wt *WebSway) ServeFile(w http.ResponseWriter, r *http.Request) {
	// Read file from filesystem provider

	storagePath, err := wt.ExtractStoragePath(r.URL.Path, "/", wt.PathBase)
	if err != nil {
		wt.NotFound(w, r)
		return
	}

	data, err := wt.FsProvider.ReadFile(storagePath)
	if err != nil {
		wt.NotFound(w, r)
		return
	}

	// Apply security headers
	wt.ApplySecurityHeaders(w)

	// Apply caching
	wt.ApplyCacheHeaders(w, r.URL.Path)

	// Set MIME type based on extension
	ext := filepath.Ext(storagePath)
	mimeType := comm.Mime.GetType(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (wt *WebSway) ServeWebManifest(storagePath, prefix string, w http.ResponseWriter, r *http.Request) {
	data, err := wt.FsProvider.ReadFile(storagePath)
	if err != nil {
		wt.NotFound(w, r)
		return
	}
	// execute textTemplate and replace {{ .Prefix }} with actual prefix
	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "{{.Prefix}}", prefix)
	data = []byte(dataStr)
	hl1.Helpers.WriteWebManifestBytes(w, data)
}
func (wt *WebSway) ServeServiceWorker(path, scope string, w http.ResponseWriter, r *http.Request) bool {
	data, err := wt.FsProvider.ReadFile(path)
	if err != nil {
		wt.NotFound(w, r)
		return false
	}

	// Apply security headers
	wt.ApplySecurityHeaders(w)
	// Service Workers must have specific headers
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Service-Worker-Allowed", scope)

	// Apply caching
	wt.ApplyCacheHeaders(w, path)

	// Write response
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return true
}
