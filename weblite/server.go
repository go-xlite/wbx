package weblite

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-xlite/wbx/comm/routes"
	"github.com/gorilla/mux"
)

// WebLite represents a lightweight web server instance
type WebLite struct {
	Provider *WebLiteProvider
	Name     string
	mux      *mux.Router
	Routes   *routes.Routes

	// Port listeners configuration
	PortListeners []*PortListener

	// Server management
	servers []*http.Server
	running bool
	mu      sync.RWMutex
}

// NewWebLite creates a new WebLite instance with default configuration
func NewWebLite(name string) *WebLite {
	wl := &WebLite{
		Name:          name,
		mux:           mux.NewRouter(),
		servers:       make([]*http.Server, 0),
		PortListeners: make([]*PortListener, 0),
	}
	wl.Routes = routes.NewRoutes(wl.mux)
	return wl
}

// Configuration methods

// AddPortListener adds a new port listener configuration
func (wl *WebLite) AddPortListener(config map[string]string) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	listener := NewPortListener(config)
	wl.PortListeners = append(wl.PortListeners, listener)

	return wl
}

// IsRunning returns whether the server is currently running
func (wl *WebLite) IsRunning() bool {
	wl.mu.RLock()
	defer wl.mu.RUnlock()
	return wl.running
}

// Server lifecycle methods

// Start starts the server in blocking mode
func (wl *WebLite) Start() error {
	wl.mu.Lock()
	if wl.running {
		wl.mu.Unlock()
		return fmt.Errorf("server %s is already running", wl.Name)
	}

	// Check if PortListeners are configured
	if len(wl.PortListeners) == 0 {
		wl.mu.Unlock()
		return fmt.Errorf("no port listeners configured for server %s", wl.Name)
	}

	wl.running = true
	wl.mu.Unlock()

	defer func() {
		wl.mu.Lock()
		wl.running = false
		wl.mu.Unlock()
	}()

	return wl.startWithPortListeners()
}

// startWithPortListeners starts servers based on PortListener configurations
func (wl *WebLite) startWithPortListeners() error {
	type bindResult struct {
		addr string
		err  error
	}

	// Calculate total number of bind addresses across all listeners
	wl.mu.RLock()
	totalBinds := 0
	for _, listener := range wl.PortListeners {
		totalBinds += len(listener.Ports) * len(listener.Addresses)
	}
	wl.mu.RUnlock()

	resultChan := make(chan bindResult, totalBinds)
	var wg sync.WaitGroup

	// Start servers for each port listener
	wl.mu.RLock()
	listeners := make([]*PortListener, len(wl.PortListeners))
	copy(listeners, wl.PortListeners)
	wl.mu.RUnlock()

	for _, listener := range listeners {
		for _, port := range listener.Ports {
			for _, addr := range listener.Addresses {
				wg.Add(1)
				go func(pl *PortListener, p string, a string) {
					defer wg.Done()
					err := wl.startListenerServer(pl, a, p)
					if err != nil && err != http.ErrServerClosed {
						bindAddr := net.JoinHostPort(a, p)
						resultChan <- bindResult{addr: bindAddr, err: err}
					} else if err == nil {
						bindAddr := net.JoinHostPort(a, p)
						resultChan <- bindResult{addr: bindAddr, err: nil}
					}
				}(listener, port, addr)
			}
		}
	}

	// Wait for all servers to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	var successAddrs []string
	var errors []bindResult
	for result := range resultChan {
		if result.err == nil {
			successAddrs = append(successAddrs, result.addr)
		} else {
			errors = append(errors, result)
		}
	}

	// Check for ignorable errors (IPv4/IPv6 dual-stack)
	for _, errResult := range errors {
		canIgnore := false

		if strings.Contains(errResult.err.Error(), "address already in use") ||
			strings.Contains(errResult.err.Error(), "bind: address already in use") {
			isIPv4Wildcard := strings.HasPrefix(errResult.addr, "0.0.0.0")
			hasIPv6Success := false
			for _, successAddr := range successAddrs {
				if successAddr == "::" || strings.HasPrefix(successAddr, "::") {
					hasIPv6Success = true
					break
				}
			}

			if isIPv4Wildcard && hasIPv6Success {
				canIgnore = true
				fmt.Printf("WebLite [%s] IPv4 bind on %s failed (address in use), but IPv6 is bound - assuming dual-stack mode\n", wl.Name, errResult.addr)
			}
		}

		if !canIgnore {
			return errResult.err
		}
	}

	return nil
}

// startListenerServer starts a server for a specific PortListener configuration
func (wl *WebLite) startListenerServer(listener *PortListener, bindAddr, port string) error {
	addr := net.JoinHostPort(bindAddr, port)

	// Wrap handler with domain validation if needed
	handler := http.Handler(wl.mux)

	// Apply domain validation through DomainValidator
	if listener.DomainValidator != nil && listener.DomainValidator.IsEnabled() {
		handler = listener.DomainValidator.Middleware(handler)
	}

	isHTTPS := listener.IsHTTPS()
	hasSSL := listener.HasSSLConfig()

	// Wrap with HTTPS redirect if needed
	if isHTTPS && hasSSL && listener.HTTPSRedirect {
		handler = wrapWithHTTPSRedirect(handler)
	}

	// Wrap with HTTP/3 Alt-Svc if enabled
	if wl.isHTTP3Enabled() && isHTTPS && hasSSL {
		handler = wrapWithHTTP3AltSvc(handler, port)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	wl.mu.Lock()
	wl.servers = append(wl.servers, server)
	wl.mu.Unlock()

	// Build log message
	logMsg := fmt.Sprintf("WebLite [%s] starting %s server on %s", wl.Name, strings.ToUpper(listener.Protocol), addr)
	if listener.OptimizeCloudflare {
		logMsg += " (CloudFlare optimized)"
	}
	if wl.isHTTP3Enabled() && isHTTPS && hasSSL {
		logMsg += " (with HTTP/3)"
	}
	fmt.Println(logMsg)

	// Handle CloudFlare optimization
	if listener.OptimizeCloudflare {
		ln, err := wl.CreateCloudFlareListener("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to create CloudFlare listener: %w", err)
		}
		defer ln.Close()

		if isHTTPS && hasSSL {
			tlsConfig, err := wl.createTLSConfigFromListener(listener)
			if err != nil {
				return err
			}
			server.TLSConfig = tlsConfig
			tlsListener := tls.NewListener(ln, server.TLSConfig)
			return server.Serve(tlsListener)
		}

		return server.Serve(ln)
	}

	// Standard listener (no CloudFlare optimizations)
	if isHTTPS && hasSSL {
		tlsConfig, err := wl.createTLSConfigFromListener(listener)
		if err != nil {
			return err
		}

		// Handle mixed protocol if HTTPS redirect is enabled
		if listener.HTTPSRedirect {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("failed to create listener: %w", err)
			}
			defer ln.Close()

			mixedLn := &mixedProtocolListener{
				Listener:  ln,
				tlsConfig: tlsConfig,
				httpsPort: port,
			}

			tlsLn := tls.NewListener(mixedLn, tlsConfig)

			// Start HTTP/3 if enabled
			if wl.isHTTP3Enabled() {
				go func() {
					if err := wl.startHTTP3Server(addr, tlsConfig, handler); err != nil {
						fmt.Printf("HTTP/3 server error: %v\n", err)
					}
				}()
			}

			return server.Serve(tlsLn)
		}

		// Start HTTP/3 if enabled
		if wl.isHTTP3Enabled() {
			errChan := make(chan error, 2)

			go func() {
				if err := wl.startHTTP3Server(addr, tlsConfig, handler); err != nil {
					errChan <- fmt.Errorf("HTTP/3 server error: %w", err)
				}
			}()

			go func() {
				var err error
				if listener.SSLCertData != "" && listener.SSLKeyData != "" {
					server.TLSConfig = tlsConfig
					err = server.ListenAndServeTLS("", "")
				} else {
					err = server.ListenAndServeTLS(listener.SSLCertPath, listener.SSLKeyPath)
				}
				if err != nil && err != http.ErrServerClosed {
					errChan <- fmt.Errorf("HTTP/1.1/2.0 server error: %w", err)
				}
			}()

			return <-errChan
		}

		// Regular HTTPS without HTTP/3
		if listener.SSLCertData != "" && listener.SSLKeyData != "" {
			server.TLSConfig = tlsConfig
			return server.ListenAndServeTLS("", "")
		}
		return server.ListenAndServeTLS(listener.SSLCertPath, listener.SSLKeyPath)
	}

	// Regular HTTP server
	// Check if this HTTP listener should redirect to HTTPS
	if !isHTTPS && listener.HTTPSRedirectPort != "" {
		// This is an HTTP listener with HTTPS redirect configured
		// Replace handler with redirect handler
		redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract host without port
			host := r.Host
			if h, _, err := net.SplitHostPort(r.Host); err == nil {
				host = h
			}

			// Build HTTPS URL
			var httpsURL string
			if listener.HTTPSRedirectPort == "443" {
				httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
			} else {
				httpsURL = fmt.Sprintf("https://%s:%s%s", host, listener.HTTPSRedirectPort, r.RequestURI)
			}

			// Permanent redirect
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		})

		server.Handler = redirectHandler
		fmt.Printf("WebLite [%s] HTTP redirect to HTTPS port %s\n", wl.Name, listener.HTTPSRedirectPort)
	}

	return server.ListenAndServe()
}

// createTLSConfigFromListener creates a TLS config from a PortListener
func (wl *WebLite) createTLSConfigFromListener(listener *PortListener) (*tls.Config, error) {
	if listener.SSLCertData != "" && listener.SSLKeyData != "" {
		// Use raw data
		cert, err := tls.X509KeyPair([]byte(listener.SSLCertData), []byte(listener.SSLKeyData))
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate from data: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	if listener.SSLCertPath != "" && listener.SSLKeyPath != "" {
		// Use files
		cert, err := tls.LoadX509KeyPair(listener.SSLCertPath, listener.SSLKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate from files: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	return nil, fmt.Errorf("no SSL configuration provided")
}

// Stop gracefully stops all server instances
func (wl *WebLite) Stop() error {
	wl.mu.Lock()
	if !wl.running {
		wl.mu.Unlock()
		return fmt.Errorf("server %s is not running", wl.Name)
	}
	wl.mu.Unlock()

	fmt.Printf("WebLite [%s] stopping...\n", wl.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var errors []error
	wl.mu.Lock()
	servers := wl.servers
	wl.mu.Unlock()

	for _, server := range servers {
		if err := server.Shutdown(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	wl.mu.Lock()
	wl.servers = make([]*http.Server, 0)
	wl.running = false
	wl.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping server: %v", errors)
	}

	fmt.Printf("WebLite [%s] stopped\n", wl.Name)
	return nil
}

// Close immediately closes all server connections
func (wl *WebLite) Close() error {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if !wl.running {
		return nil
	}

	fmt.Printf("WebLite [%s] closing...\n", wl.Name)

	var errors []error
	for _, server := range wl.servers {
		if err := server.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	wl.servers = make([]*http.Server, 0)
	wl.running = false

	if len(errors) > 0 {
		return fmt.Errorf("errors closing server: %v", errors)
	}

	return nil
}

// GetAddr returns the addresses the server is bound to
func (wl *WebLite) GetAddr() []string {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	var addrs []string
	for _, listener := range wl.PortListeners {
		for _, port := range listener.Ports {
			for _, addr := range listener.Addresses {
				addrs = append(addrs, net.JoinHostPort(addr, port))
			}
		}
	}
	return addrs
}

// GetRoutes returns the Routes instance
func (wl *WebLite) GetRoutes() *routes.Routes {
	return wl.Routes
}

// GetMux returns the mux.Router instance
func (wl *WebLite) GetMux() *mux.Router {
	return wl.mux
}

// wrapWithHTTPSRedirect wraps a handler to redirect HTTP requests to HTTPS
// This handles the case where someone sends an HTTP request to an HTTPS port
func wrapWithHTTPSRedirect(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If TLS is nil, this is an HTTP request on an HTTPS port
		if r.TLS == nil {
			// Extract host without port
			host := r.Host
			if h, _, err := net.SplitHostPort(r.Host); err == nil {
				host = h
			}

			// Get the port from the request (the HTTPS port they're hitting)
			_, port, _ := net.SplitHostPort(r.Host)

			// Build HTTPS URL
			var httpsURL string
			if port == "" || port == "443" {
				httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
			} else {
				httpsURL = fmt.Sprintf("https://%s:%s%s", host, port, r.RequestURI)
			}

			// Permanent redirect
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
			return
		}

		// Normal HTTPS request, pass through
		handler.ServeHTTP(w, r)
	})
}
