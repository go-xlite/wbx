//go:build http3
// +build http3

package weblite

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

// http3AltSvcMiddleware adds the Alt-Svc header to advertise HTTP/3 availability
type http3AltSvcMiddleware struct {
	handler http.Handler
	port    string
}

func (m *http3AltSvcMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add Alt-Svc header before serving the request
	if m.port != "" {
		// Check if header already exists to avoid duplication
		if len(w.Header().Values("Alt-Svc")) == 0 {
			w.Header().Add("Alt-Svc", fmt.Sprintf(`h3=":%s"; ma=86400`, m.port))
		}
	}
	m.handler.ServeHTTP(w, r)
}

// wrapWithHTTP3AltSvc wraps a handler to automatically add Alt-Svc headers
func wrapWithHTTP3AltSvc(handler http.Handler, port string) http.Handler {
	return &http3AltSvcMiddleware{
		handler: handler,
		port:    port,
	}
}

// startHTTP3Server starts an HTTP/3 server for the given address
// This runs in addition to the HTTP/1.1/2.0 server
func (wl *WebLite) startHTTP3Server(addr string, tlsConfig *tls.Config, handler http.Handler) error {
	http3Server := &http3.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
	}

	fmt.Printf("WebLite [%s] starting HTTP/3 on %s\n", wl.Name, addr)

	return http3Server.ListenAndServe()
}

// isHTTP3Enabled returns true when HTTP/3 is compiled in
func (wl *WebLite) isHTTP3Enabled() bool {
	return true
}
