package weblite

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-xlite/wbx/comm"
	serverrole "github.com/go-xlite/wbx/server_role"
	weblite "github.com/go-xlite/wbx/weblite"
)

// CdnServer is optimized for serving CDN/static content
// Features: Automatic MIME types, caching headers, compression, embedded FS support
type CdnServer struct {
	*serverrole.ServerRole
	CacheMaxAge   time.Duration
	EnableBrowser bool // Allow browser caching
	EnableETags   bool
}

// NewCdnServerFromWebLite wraps an existing WebLite with CdnServer functionality
func NewCdnServer(wl *weblite.WebLite) *CdnServer {
	return &CdnServer{
		ServerRole: &serverrole.ServerRole{
			Server:      wl,
			CustomMimes: make(map[string]string),
			PathPrefix:  "/cdn",
		},
		CacheMaxAge:   24 * time.Hour,
		EnableBrowser: true,
		EnableETags:   true,
	}
}

// SetCaching configures caching behavior
func (cs *CdnServer) SetCaching(maxAge time.Duration, enableBrowser bool) *CdnServer {
	cs.CacheMaxAge = maxAge
	cs.EnableBrowser = enableBrowser
	return cs
}

// ServeFile serves files from a filesystem provider
func (cs *CdnServer) ServeFile(urlPath string, fsProvider comm.IFsProvider) {
	fullPath := cs.PathPrefix + urlPath

	cs.Server.GetRoutes().HandlePathPrefixFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
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
		cs.applyCacheHeaders(w, r)
		ext := filepath.Ext(relativePath)
		w.Header().Set("Content-Type", cs.ServerRole.GetMimeType(ext))
		w.Write(data)
	})
}

// ServeBytes serves raw bytes with specified MIME type
func (cs *CdnServer) ServeBytes(urlPath string, data []byte, mimeType string) {
	fullPath := cs.PathPrefix + urlPath
	cs.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		cs.applyCacheHeaders(w, r)
		w.Header().Set("Content-Type", mimeType)
		w.Write(data)
	})
}

// applyCacheHeaders applies appropriate caching headers
func (cs *CdnServer) applyCacheHeaders(w http.ResponseWriter, r *http.Request) {

	if cs.EnableBrowser {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cs.CacheMaxAge.Seconds())))
		w.Header().Set("Expires", time.Now().Add(cs.CacheMaxAge).Format(http.TimeFormat))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// Start starts the CDN server
func (cs *CdnServer) Start() error {
	return nil
}

// Stop stops the CDN server
func (cs *CdnServer) Stop() error {
	return nil
}

// WriteBinary writes binary data with appropriate content type
func WriteBinary(w http.ResponseWriter, data []byte, filename string) error {
	ext := filepath.Ext(filename)
	w.Header().Set("Content-Type", comm.GetMimeType(ext))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	_, err := w.Write(data)
	return err
}

// WriteDownload forces download with appropriate headers
func WriteDownload(w http.ResponseWriter, data []byte, filename string) error {
	ext := filepath.Ext(filename)
	w.Header().Set("Content-Type", comm.GetMimeType(ext))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	_, err := w.Write(data)
	return err
}

// StreamFile streams a file to the response
func StreamFile(w http.ResponseWriter, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	ext := filepath.Ext(filePath)
	w.Header().Set("Content-Type", comm.GetMimeType(ext))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	_, err = io.Copy(w, file)
	return err
}
