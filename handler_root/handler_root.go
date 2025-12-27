package handlerroot

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	comm "github.com/go-xlite/wbx/comm"
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

// SetCacheMaxAge sets the cache duration for root files
func (rh *RootHandler) SetCacheMaxAge(duration time.Duration) *RootHandler {
	rh.CacheMaxAge = duration
	return rh
}

// ServeFavicon serves favicon.ico from a filesystem provider
func (rh *RootHandler) ServeFavicon(fsProvider comm.IFsAdapter) error {
	return rh.ServeRootFile("/favicon.ico", "favicon.ico", fsProvider)
}

// ServeManifest serves manifest.json or site.webmanifest from a filesystem provider
func (rh *RootHandler) ServeManifest(manifestPath string, fsProvider comm.IFsAdapter) error {
	data, err := fsProvider.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	urlPath := "/" + filepath.Base(manifestPath)
	fullPath := rh.PathPrefix.Get() + strings.TrimPrefix(urlPath, "/")

	rh.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(manifestPath)

		// Set proper MIME type for manifest files
		switch ext {
		case ".webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		default:
			w.Header().Set("Content-Type", "application/manifest+json")
		}

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Apply caching
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(rh.CacheMaxAge.Seconds())))

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	return nil
}

// ServeRobotsTxt serves robots.txt from a filesystem provider
func (rh *RootHandler) ServeRobotsTxt(fsProvider comm.IFsAdapter) error {
	return rh.ServeRootFile("/robots.txt", "robots.txt", fsProvider)
}

// ServeSitemap serves sitemap.xml from a filesystem provider
func (rh *RootHandler) ServeSitemap(fsProvider comm.IFsAdapter) error {
	return rh.ServeRootFile("/sitemap.xml", "sitemap.xml", fsProvider)
}

// ServeRootFile serves a single file at the root level with proper caching
func (rh *RootHandler) ServeRootFile(urlPath, filePath string, fsProvider comm.IFsAdapter) error {
	data, err := fsProvider.ReadFile(filePath)
	if err != nil {
		return err
	}

	fullPath := rh.PathPrefix.Get() + strings.TrimPrefix(urlPath, "/")
	rh.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		// Set MIME type based on extension
		ext := filepath.Ext(filePath)
		mimeType := rh.HandlerRole.GetMimeType(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)

		// Apply caching
		rh.applyCacheHeaders(w, ext)

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	return nil
}

// ServeRootFiles serves multiple root files from a filesystem provider
func (rh *RootHandler) ServeRootFiles(files map[string]string, fsProvider comm.IFsAdapter) error {
	for urlPath, filePath := range files {
		if err := rh.ServeRootFile(urlPath, filePath, fsProvider); err != nil {
			return err
		}
	}
	return nil
}

// Serve404 sets up a custom 404 error page from a filesystem provider
func (rh *RootHandler) Serve404(filePath string, fsProvider comm.IFsAdapter) error {
	data, err := fsProvider.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read 404 page: %w", err)
	}
	rh.NotFoundPage = data
	return nil
}

// Serve500 sets up a custom 500 error page from a filesystem provider
func (rh *RootHandler) Serve500(filePath string, fsProvider comm.IFsAdapter) error {
	data, err := fsProvider.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read 500 page: %w", err)
	}
	rh.ServerErrorPage = data
	return nil
}

// Handle404 writes the 404 error page response
func (rh *RootHandler) Handle404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)

	if len(rh.NotFoundPage) > 0 {
		w.Write(rh.NotFoundPage)
	} else {
		w.Write([]byte("<html><body><h1>404 - Page Not Found</h1></body></html>"))
	}
}

// Handle500 writes the 500 error page response
func (rh *RootHandler) Handle500(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusInternalServerError)

	if len(rh.ServerErrorPage) > 0 {
		w.Write(rh.ServerErrorPage)
	} else {
		w.Write([]byte("<html><body><h1>500 - Internal Server Error</h1></body></html>"))
	}
}

// HandleError writes a custom error page response with any status code
func (rh *RootHandler) HandleError(w http.ResponseWriter, r *http.Request, statusCode int, errorPage []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(statusCode)

	if len(errorPage) > 0 {
		w.Write(errorPage)
	} else {
		w.Write([]byte(fmt.Sprintf("<html><body><h1>%d - Error</h1></body></html>", statusCode)))
	}
}

// applyCacheHeaders applies caching headers based on file type
func (rh *RootHandler) applyCacheHeaders(w http.ResponseWriter, ext string) {
	// Favicon and static assets can be cached
	if ext == ".ico" || ext == ".png" || ext == ".svg" || ext == ".webmanifest" || ext == ".json" {
		w.Header().Set("Cache-Control", "public, max-age="+string(rune(int(rh.CacheMaxAge.Seconds()))))
		return
	}

	// robots.txt and sitemap.xml should have shorter cache
	if ext == ".txt" || ext == ".xml" {
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
		return
	}

	// Default: no cache
	w.Header().Set("Cache-Control", "no-cache")
}
