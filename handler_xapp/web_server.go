package weblite

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	comm "github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// XAppHandler is optimized for serving HTML applications with linked assets
// Features: Template rendering, asset serving, security headers
type XAppHandler struct {
	*handler_role.HandlerRole
	SecurityHeaders   bool
	CacheMaxAge       time.Duration
	VirtualDirSegment string // Virtual directory segment (default: "p")
	SessionResolver   comm.SessionResolver
	LoginPage         string
	AuthSkippedPaths  []string
	DefaultRoute      string // Default route for root path (e.g., "index" to serve from /index directory when accessing /)
}

// NewXAppHandler creates a XAppHandler wrapper around an existing handler instance
func NewXAppHandler(wl handler_role.IHandler) *XAppHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.Handler = wl

	return &XAppHandler{
		HandlerRole:       handlerRole,
		SecurityHeaders:   true,
		CacheMaxAge:       1 * time.Hour,
		VirtualDirSegment: "p",
		LoginPage:         "/login",
		AuthSkippedPaths:  []string{"/login", "/logout", "/"},
		DefaultRoute:      "index", // Default to serving from "index" directory for root path
	}
}

// extractStoragePath converts a request path to a storage path
// It handles the /p/ virtual directory pattern where:
// - URL: example.com/index/p/app.js -> Storage: index/app.js
// - URL: example.com/index/p/ -> Storage: index/index.html
// - URL: example.com/index/p/nested1/nested2/app.js -> Storage: index/nested1/nested2/app.js
// Returns an error if the path contains invalid or malicious patterns
func (ws *XAppHandler) extractStoragePath(requestPath, urlPath string) (string, error) {
	// Remove the handler's PathPrefix if set (e.g., "/api" or "api")
	relativePath, err := ws.PathPrefix.StripPrefix(requestPath)
	if err != nil {
		return "", err
	}

	// Determine the app directory (storage directory)
	var appDir string

	if urlPath == "/" {
		// Serving from root - extract app directory from first path segment
		// Examples:
		// /home/index.html -> appDir="home", relativePath="/index.html"
		// /home/p/app.js -> appDir="home", relativePath="/p/app.js"
		// /p/app.js -> appDir="index" (default), relativePath="/p/app.js"
		// / -> appDir="index" (default), relativePath="/"

		if relativePath == "" || relativePath == "/" {
			// Root path, use default
			appDir = ws.DefaultRoute
			relativePath = "/"
		} else {
			// Extract first segment
			pathParts := strings.SplitN(strings.TrimPrefix(relativePath, "/"), "/", 2)
			firstSegment := pathParts[0]

			// Check if first segment is the virtual directory (e.g., "p")
			if firstSegment == ws.VirtualDirSegment {
				// Path like /p/app.js - use default app directory
				appDir = ws.DefaultRoute
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
	// e.g., /p/app.js or /p/nested1/nested2/app.js or /p/
	virtualSegment := "/" + ws.VirtualDirSegment
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
	// e.g., appDir="index" + relativePath="/app.js" -> "index/app.js"
	// e.g., appDir="home" + relativePath="/index.html" -> "home/index.html"
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

// ServeStatic serves static files from a filesystem provider with proper MIME types
// It handles the /p/ virtual directory pattern where:
// - URL: example.com/index/p/app.js -> Storage: dist/index/app.js
// - URL: example.com/index/p/ -> Storage: dist/index/index.html
func (ws *XAppHandler) ServeStatic(urlPath string, fsProvider comm.IFsAdapter) {
	pathPrefix := ws.PathPrefix.Get()
	fullPath := pathPrefix + urlPath

	// If serving from urlPath "/" with a PathPrefix, setup redirect from root "/" to prefix
	// Example: PathPrefix="/xt23", urlPath="/" -> redirect "/" to "/xt23/"
	if urlPath == "/" && ws.PathPrefix.IsSet() {
		ws.Handler.GetRoutes().HandlePathFn("/", func(w http.ResponseWriter, r *http.Request) {
			redirectTarget := pathPrefix
			if !strings.HasSuffix(redirectTarget, "/") {
				redirectTarget += "/"
			}
			http.Redirect(w, r, redirectTarget, http.StatusMovedPermanently)
		})
	}

	ws.Handler.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate storage path from request path
		storagePath, err := ws.extractStoragePath(r.URL.Path, urlPath)
		if err != nil {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Resolve session if configured and serving HTML file
		ext := filepath.Ext(storagePath)
		isHTML := ext == ".html" || ext == ".htm" || ext == ""

		if isHTML && ws.SessionResolver != nil {
			// Check if path is in auth-skipped paths
			skipAuth := false
			for _, skippedPath := range ws.AuthSkippedPaths {
				if r.URL.Path == skippedPath || r.URL.Path == ws.PathPrefix.Get()+skippedPath {
					skipAuth = true
					break
				}
			}

			if !skipAuth {
				ctx := ws.SessionResolver.ResolveSession(r)
				r = r.WithContext(ctx)

				// Check if user is authenticated
				if _, ok := comm.GetUserID(r); !ok {
					// User not authenticated, redirect to login
					http.Redirect(w, r, ws.PathPrefix.Get()+ws.LoginPage, http.StatusFound)
					return
				}
			}
		}

		// Call OnRequest interceptor if defined
		if ws.OnRequest != nil {
			if !ws.OnRequest(w, r) {
				// Request was handled by interceptor, stop processing
				return
			}
		}

		// Read file from filesystem provider
		data, err := fsProvider.ReadFile(storagePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Inject base tag and prefix paths for HTML files when PathPrefix is set
		if isHTML && ws.PathPrefix.IsSet() {
			htmlContent := ws.PathPrefix.PatchHTML(string(data))
			data = []byte(htmlContent)
		}

		// Apply security headers
		if ws.SecurityHeaders {
			ws.applySecurityHeaders(w)
		}

		// Apply caching
		ws.applyCacheHeaders(w, r)

		// Set MIME type based on extension (ext already extracted above)
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

// applySecurityHeaders applies common security headers
func (ws *XAppHandler) applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// applyCacheHeaders applies caching headers based on content type
func (ws *XAppHandler) applyCacheHeaders(w http.ResponseWriter, r *http.Request) {
	ext := filepath.Ext(r.URL.Path)

	// HTML should not be cached
	if ext == ".html" || ext == ".htm" || ext == "" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}
	if comm.Mime.IsStaticExtension(ext) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ws.CacheMaxAge.Seconds())))
		return
	}
	// Default: no cache
	w.Header().Set("Cache-Control", "no-cache")
}
