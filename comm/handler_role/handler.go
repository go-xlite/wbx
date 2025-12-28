package handler_role

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-xlite/wbx/comm"
	"github.com/go-xlite/wbx/comm/routes"
	"github.com/gorilla/mux"
)

// IServer is the interface that WebLite and WebTrail implement
type IHandler interface {
	GetRoutes() *routes.Routes
	GetMux() *mux.Router
}

type PathPrefix struct {
	sv     *HandlerRole
	Prefix string
}

func (pp *PathPrefix) Set(prefix string) {
	// Normalize prefix to have leading slash, unless empty or just "/"
	if prefix == "" || prefix == "/" {
		pp.Prefix = ""
	} else {
		if !strings.HasPrefix(prefix, "/") {
			pp.Prefix = "/" + prefix
		} else {
			pp.Prefix = prefix
		}
	}
}

func (pp *PathPrefix) Get() string {
	return pp.Prefix
}

// IsSet returns true if a non-empty prefix is configured
func (pp *PathPrefix) IsSet() bool {
	return pp.Prefix != ""
}

// StripPrefix removes the path prefix from a request path
// Returns the relative path with prefix removed, or error if path doesn't start with prefix
func (pp *PathPrefix) StripPrefix(requestPath string) (string, error) {
	if !pp.IsSet() {
		return requestPath, nil
	}

	pathPrefix := pp.Get()
	if !strings.HasPrefix(requestPath, pathPrefix) {
		return "", fmt.Errorf("path does not start with PathPrefix")
	}

	return requestPath[len(pathPrefix):], nil
}

// PatchHTML injects the path prefix into HTML content by replacing absolute URLs
// Replaces src="/ and href="/ with src="/prefix/ and href="/prefix/
// Also adds a base tag for additional support
func (pp *PathPrefix) PatchHTML(htmlContent string) string {
	if !pp.IsSet() {
		return htmlContent
	}

	prefix := pp.Get()

	// Replace common absolute paths with prefixed versions
	// Replace src="/... and href="/... with prefixed versions
	htmlContent = strings.ReplaceAll(htmlContent, `src="/`, `src="`+prefix+`/`)
	htmlContent = strings.ReplaceAll(htmlContent, `href="/`, `href="`+prefix+`/`)
	htmlContent = strings.ReplaceAll(htmlContent, `SRC="/`, `SRC="`+prefix+`/`)
	htmlContent = strings.ReplaceAll(htmlContent, `HREF="/`, `HREF="`+prefix+`/`)

	// Also add base tag for additional support
	baseTag := "<base href=\"" + prefix + "/\">"
	if strings.Contains(strings.ToLower(htmlContent), "<head>") {
		htmlContent = strings.Replace(htmlContent, "<head>", "<head>\n    "+baseTag, 1)
		htmlContent = strings.Replace(htmlContent, "<HEAD>", "<HEAD>\n    "+baseTag, 1)
	}

	return htmlContent
}

type HandlerRole struct {
	Handler     IHandler
	CustomMimes map[string]string
	PathPrefix  *PathPrefix
	CORS        CORS
	OnStart     func() error
	OnStop      func() error
	OnRequest   func(w http.ResponseWriter, r *http.Request) bool
}

func (sr *HandlerRole) Start() error {
	if sr.OnStart != nil {
		return sr.OnStart()
	}
	return nil
}

func (sr *HandlerRole) Stop() error {
	if sr.OnStop != nil {
		return sr.OnStop()
	}
	return nil
}

type CORS struct {
	sv          *HandlerRole
	EnableCORS  bool
	CORSOrigins []string
}

// SetCORS configures CORS settings
func (c *CORS) SetCORS(enabled bool, origins ...string) {
	c.EnableCORS = enabled
	if len(origins) > 0 {
		c.CORSOrigins = origins
	}
}

// ApplyCORS applies CORS headers to the response
func (c *CORS) ApplyCORS(w http.ResponseWriter, r *http.Request) {
	if !c.EnableCORS {
		return
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	// Check if origin is allowed
	allowed := false
	for _, allowedOrigin := range c.CORSOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			allowed = true
			break
		}
	}

	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")
	}
}

func NewHandler() *HandlerRole {
	sr := &HandlerRole{
		CustomMimes: make(map[string]string),
		PathPrefix:  &PathPrefix{sv: nil, Prefix: ""},
	}
	sr.CORS.sv = sr
	sr.CORS.EnableCORS = false
	sr.CORS.CORSOrigins = []string{"*"}
	return sr
}

// AddCustomMime adds a custom MIME type mapping
func (sr *HandlerRole) AddCustomMime(extension, mimeType string) *HandlerRole {
	if sr.CustomMimes == nil {
		sr.CustomMimes = make(map[string]string)
	}
	sr.CustomMimes[extension] = mimeType
	return sr
}

// GetMimeType returns the MIME type for a file extension
// Checks custom MIME types first, then falls back to standard types
func (sr *HandlerRole) GetMimeType(ext string) string {
	ext = strings.ToLower(ext)

	// Check custom MIME types first
	if sr.CustomMimes != nil {
		if mime, ok := sr.CustomMimes[ext]; ok {
			return mime
		}
	}

	// Return standard MIME type
	return comm.GetMimeType(ext)
}

func (hr *HandlerRole) SetPathPrefix(prefix string) {
	hr.PathPrefix.Set(prefix)
}

// Redirect performs an HTTP redirect to the specified path
func (hr *HandlerRole) Redirect(w http.ResponseWriter, r *http.Request, toPath string) {
	http.Redirect(w, r, toPath, http.StatusFound)
}

// RedirectPermanent performs a permanent HTTP redirect to the specified path
func (hr *HandlerRole) RedirectPermanent(w http.ResponseWriter, r *http.Request, toPath string) {
	http.Redirect(w, r, toPath, http.StatusMovedPermanently)
}
