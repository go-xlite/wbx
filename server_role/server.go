package serverrole

import (
	"net/http"
	"strings"

	"github.com/go-xlite/wbx/comm"
	weblite "github.com/go-xlite/wbx/weblite"
)

type ServerRole struct {
	Server      *weblite.WebLite
	CustomMimes map[string]string
	PathPrefix  string
	CORS        CORS
}

type CORS struct {
	sv          *ServerRole
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

func NewServer() *ServerRole {
	sr := &ServerRole{
		CustomMimes: make(map[string]string),
		PathPrefix:  "",
	}
	sr.CORS.sv = sr
	sr.CORS.EnableCORS = false
	sr.CORS.CORSOrigins = []string{"*"}
	return sr
}

// AddCustomMime adds a custom MIME type mapping
func (sr *ServerRole) AddCustomMime(extension, mimeType string) *ServerRole {
	if sr.CustomMimes == nil {
		sr.CustomMimes = make(map[string]string)
	}
	sr.CustomMimes[extension] = mimeType
	return sr
}

// GetMimeType returns the MIME type for a file extension
// Checks custom MIME types first, then falls back to standard types
func (sr *ServerRole) GetMimeType(ext string) string {
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

// SetPathPrefix sets the base path prefix for CDN routes
func (cs *ServerRole) SetPathPrefix(prefix string) {
	cs.PathPrefix = prefix
}
