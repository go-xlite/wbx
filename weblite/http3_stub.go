//go:build !http3
// +build !http3

package weblite

import (
	"crypto/tls"
	"net/http"
)

// wrapWithHTTP3AltSvc is a no-op when HTTP/3 is not compiled
func wrapWithHTTP3AltSvc(handler http.Handler, port string) http.Handler {
	return handler
}

// startHTTP3Server is a no-op when HTTP/3 is not compiled
func (wl *WebLite) startHTTP3Server(addr string, tlsConfig *tls.Config, handler http.Handler) error {
	// No-op: HTTP/3 not compiled
	return nil
}

// isHTTP3Enabled returns false when HTTP/3 is not compiled in
func (wl *WebLite) isHTTP3Enabled() bool {
	return false
}

// getHTTP3Port returns empty string when HTTP/3 is not compiled
func getHTTP3Port(addr string) string {
	return ""
}
