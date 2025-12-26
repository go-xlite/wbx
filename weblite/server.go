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

	"github.com/gorilla/mux"
)

// WebLite represents a lightweight web server instance
type WebLite struct {
	Provider *WebLiteProvider
	Name     string
	Mux      *mux.Router
	Routes   *wlRoutes
	Port     string
	BindAddr []string
	SslCert  string
	SslKey   string
	Stats    *wlStats

	// Raw SSL/TLS data (alternative to file paths)
	sslCertData []byte
	sslKeyData  []byte

	// Server management
	servers  []*http.Server
	running  bool
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewWebLite creates a new WebLite instance with default configuration
func NewWebLite(name string) *WebLite {
	wl := &WebLite{
		Name:     name,
		Mux:      mux.NewRouter(),
		Port:     "8080",
		BindAddr: []string{"0.0.0.0", "::"}, // Default to dual-stack (IPv4 + IPv6)
		servers:  make([]*http.Server, 0),
		stopChan: make(chan struct{}),
	}
	wl.Routes = &wlRoutes{
		wl: wl,
	}
	wl.Stats = &wlStats{
		wl: wl,
	}
	wl.Stats.init()
	return wl
}

// Configuration methods

// SetPort sets the port for the server
func (wl *WebLite) SetPort(port string) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.Port = port
	return wl
}

// SetBindAddr sets the bind addresses for the server
func (wl *WebLite) SetBindAddr(addrs ...string) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.BindAddr = addrs
	return wl
}

// SetBindAddrsWithPorts configures bind addresses that may include ports.
// Addresses can be specified with or without ports:
//   - "0.0.0.0" will use defaultPort
//   - "0.0.0.0:8080" will use port 8080
//   - "[::]:9000" will use port 9000
//
// If both IPv4 (0.0.0.0) and IPv6 (::) wildcards are specified with the same port,
// only the IPv6 address will be kept (as it typically binds to both IPv4 and IPv6).
func (wl *WebLite) SetBindAddrsWithPorts(defaultPort string, addrs ...string) *WebLite {
	// Apply default port to addresses without ports
	processed := applyDefaultPortToAddrs(addrs, defaultPort)

	// Filter redundant addresses (e.g., 0.0.0.0 when :: is present on same port)
	filtered := filterRedundantAddrs(processed)

	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.Port = "" // Empty port means addresses include their own ports
	wl.BindAddr = filtered
	return wl
}

// SetSSL configures SSL/TLS for the server using file paths
func (wl *WebLite) SetSSL(certFile, keyFile string) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.SslCert = certFile
	wl.SslKey = keyFile
	wl.sslCertData = nil
	wl.sslKeyData = nil
	return wl
}

// SetSSLFromData configures SSL/TLS for the server using raw certificate and key data
func (wl *WebLite) SetSSLFromData(certData, keyData []byte) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.sslCertData = certData
	wl.sslKeyData = keyData
	wl.SslCert = ""
	wl.SslKey = ""
	return wl
}

// SetSSLFromText configures SSL/TLS for the server using raw certificate and key text
func (wl *WebLite) SetSSLFromText(certText, keyText string) *WebLite {
	return wl.SetSSLFromData([]byte(certText), []byte(keyText))
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
	wl.running = true
	wl.mu.Unlock()

	// Reset stats on start
	wl.Stats.init()

	defer func() {
		wl.mu.Lock()
		wl.running = false
		wl.mu.Unlock()
	}()

	// Start servers for all bind addresses
	errChan := make(chan error, len(wl.BindAddr))
	var wg sync.WaitGroup

	for _, addr := range wl.BindAddr {
		wg.Add(1)
		go func(bindAddr string) {
			defer wg.Done()
			if err := wl.startServer(bindAddr); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}(addr)
	}

	// Wait for all servers to complete
	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return err
	}

	return nil
}

// StartBackground starts the server in background mode (non-blocking)
func (wl *WebLite) StartBackground() error {
	wl.mu.Lock()
	if wl.running {
		wl.mu.Unlock()
		return fmt.Errorf("server %s is already running", wl.Name)
	}
	wl.running = true
	wl.mu.Unlock()

	// Reset stats on start
	wl.Stats.init()

	// Track bind results
	resultChan := make(chan error, len(wl.BindAddr))

	// Start servers for all bind addresses
	for _, addr := range wl.BindAddr {
		go func(bindAddr string) {
			if err := wl.startServer(bindAddr); err != nil && err != http.ErrServerClosed {
				fmt.Printf("WebLite [%s] error on %s: %v\n", wl.Name, bindAddr, err)
				resultChan <- err
			} else {
				resultChan <- nil
			}
		}(addr)
	}

	// Give servers time to start and check if at least one succeeded
	time.Sleep(100 * time.Millisecond)

	// Check if at least one bind succeeded
	successCount := 0
	errorCount := 0
	for i := 0; i < len(wl.BindAddr); i++ {
		select {
		case err := <-resultChan:
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		case <-time.After(50 * time.Millisecond):
			// Timeout waiting for result, assume success
			successCount++
		}
	}

	// If all binds failed, return error
	if successCount == 0 && errorCount > 0 {
		wl.mu.Lock()
		wl.running = false
		wl.mu.Unlock()
		return fmt.Errorf("server %s failed to bind to any address", wl.Name)
	}

	return nil
}

// startServer starts a single server instance for a specific bind address
func (wl *WebLite) startServer(bindAddr string) error {
	// Check if bindAddr already includes a port
	// If it does, use it directly; otherwise, join with wl.Port
	var addr string
	if wl.Port == "" {
		// Port is empty, assume bindAddr includes the port
		addr = bindAddr
	} else {
		// Use JoinHostPort to properly format the address with the port
		addr = net.JoinHostPort(bindAddr, wl.Port)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: wl.Mux,
	}

	wl.mu.Lock()
	wl.servers = append(wl.servers, server)

	// Check if SSL is configured via raw data
	useTLSFromData := len(wl.sslCertData) > 0 && len(wl.sslKeyData) > 0
	useTLSFromFiles := wl.SslCert != "" && wl.SslKey != ""
	certData := wl.sslCertData
	keyData := wl.sslKeyData
	certFile := wl.SslCert
	keyFile := wl.SslKey
	wl.mu.Unlock()

	fmt.Printf("WebLite [%s] starting on %s\n", wl.Name, addr)

	// Configure TLS if raw certificate data is provided
	if useTLSFromData {
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return fmt.Errorf("failed to parse certificate and key: %w", err)
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		// Start TLS server with empty cert/key paths since we're using TLSConfig
		return server.ListenAndServeTLS("", "")
	}

	// Use file paths if provided
	if useTLSFromFiles {
		return server.ListenAndServeTLS(certFile, keyFile)
	}

	// Start regular HTTP server
	return server.ListenAndServe()
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

	close(wl.stopChan)
	wl.stopChan = make(chan struct{})

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
	close(wl.stopChan)
	wl.stopChan = make(chan struct{})

	if len(errors) > 0 {
		return fmt.Errorf("errors closing server: %v", errors)
	}

	return nil
}

// GetAddr returns the addresses the server is bound to
func (wl *WebLite) GetAddr() []string {
	wl.mu.RLock()
	defer wl.mu.RUnlock()

	addrs := make([]string, len(wl.BindAddr))
	for i, addr := range wl.BindAddr {
		if wl.Port == "" {
			// Port is empty, addresses already include ports
			addrs[i] = addr
		} else {
			// Join host and port
			addrs[i] = net.JoinHostPort(addr, wl.Port)
		}
	}
	return addrs
}

// Stats convenience methods

// GetStats returns a snapshot of current statistics
func (wl *WebLite) GetStats() StatsSnapshot {
	return wl.Stats.Get()
}

// ResetStats clears all statistics
func (wl *WebLite) ResetStats() {
	wl.Stats.Reset()
}

// EnableDetailedStats enables detailed path and status code tracking
// This has a small performance impact due to map updates under lock
func (wl *WebLite) EnableDetailedStats() {
	wl.Stats.SetDetailedTracking(true)
}

// DisableDetailedStats disables detailed tracking for maximum performance
// Only basic counters (total requests, active, bytes) will be tracked
func (wl *WebLite) DisableDetailedStats() {
	wl.Stats.SetDetailedTracking(false)
}

// Routes wrapper type

type wlRoutes struct {
	wl *WebLite
}

// Handle registers a handler for the given pattern
func (wlr *wlRoutes) Handle(pattern string, handler http.Handler) {
	wlr.wl.Mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern
func (wlr *wlRoutes) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.HandleFunc(pattern, handler)
}

// HandlePathPrefix registers a handler for all paths under the given prefix
// The prefix is automatically stripped from the request path before passing to the handler
// Example: HandlePathPrefix("/static/", handler) will serve "/static/file.css" as "/file.css" to the handler
func (wlr *wlRoutes) HandlePathPrefix(prefix string, handler http.Handler) {
	wlr.wl.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, handler))
}

// HandlePathPrefixFunc registers a handler function for all paths under the given prefix
// The prefix is automatically stripped from the request path before passing to the handler
// Example: HandlePathPrefixFunc("/static/", handler) will serve "/static/file.css" as "/file.css" to the handler
func (wlr *wlRoutes) HandlePathPrefixFunc(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.HandlerFunc(handler)))
}

// HandlePathPrefixH is a convenience method that accepts a handler function and converts it to http.Handler
// This eliminates the need to wrap with http.HandlerFunc manually
// Example: HandlePathPrefixH("/api/", myHandlerFunc) instead of HandlePathPrefix("/api/", http.HandlerFunc(myHandlerFunc))
func (wlr *wlRoutes) HandlePathPrefixH(prefix string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.HandlerFunc(handler)))
}

// HandleH is a convenience method that accepts a handler function and converts it to http.Handler
// This eliminates the need to wrap with http.HandlerFunc manually
// Example: HandleH("/api/endpoint", myHandlerFunc) instead of Handle("/api/endpoint", http.HandlerFunc(myHandlerFunc))
func (wlr *wlRoutes) HandleH(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.Handle(pattern, http.HandlerFunc(handler))
}

// GetRoutes returns all registered routes with their methods
func (wlr *wlRoutes) GetRoutes() []map[string]string {
	routes := []map[string]string{}

	err := wlr.wl.Mux.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			// If GetPathTemplate fails, try GetPathRegexp
			pathRegexp, _ := route.GetPathRegexp()
			if pathRegexp != "" {
				pathTemplate = pathRegexp
			}
		}

		// Skip empty paths
		if pathTemplate == "" {
			return nil
		}

		methods, _ := route.GetMethods()

		methodStr := "GET, POST, PUT, DELETE, PATCH, OPTIONS" // Default if no methods specified
		if len(methods) > 0 {
			methodStr = strings.Join(methods, ", ")
		}

		routes = append(routes, map[string]string{
			"path":    pathTemplate,
			"methods": methodStr,
		})
		return nil
	})

	// If Walk returns an error or no routes found, return empty slice
	if err != nil || len(routes) == 0 {
		return []map[string]string{}
	}

	return routes
}

// HandleWithStats registers a handler with automatic stats tracking
func (wlr *wlRoutes) HandleWithStats(pattern string, handler http.Handler) {
	wlr.wl.Mux.Handle(pattern, wlr.statsMiddleware(pattern, handler))
}

// HandleFuncWithStats registers a handler function with automatic stats tracking
func (wlr *wlRoutes) HandleFuncWithStats(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.HandleFunc(pattern, handler).Handler(wlr.statsMiddleware(pattern, http.HandlerFunc(handler)))
}

// HandleWithStatsFast registers a handler with fast stats tracking (no path/code details)
// Use this for high-throughput endpoints where detailed stats aren't needed
func (wlr *wlRoutes) HandleWithStatsFast(pattern string, handler http.Handler) {
	wlr.wl.Mux.Handle(pattern, wlr.statsFastMiddleware(handler))
}

// HandleFuncWithStatsFast registers a handler function with fast stats tracking
func (wlr *wlRoutes) HandleFuncWithStatsFast(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wlr.wl.Mux.HandleFunc(pattern, handler).Handler(wlr.statsFastMiddleware(http.HandlerFunc(handler)))
}

// statsMiddleware wraps a handler to track request statistics
func (wlr *wlRoutes) statsMiddleware(pattern string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record request start
		wlr.wl.Stats.RecordRequest(pattern)

		// Wrap response writer to capture status code and bytes
		srw := &statsResponseWriter{
			ResponseWriter: w,
			statusCode:     200, // default status code
		}

		// Serve the request
		handler.ServeHTTP(srw, r)

		// Record response completion
		wlr.wl.Stats.RecordResponse(srw.statusCode, srw.bytesWritten)
	})
}

// statsFastMiddleware wraps a handler for ultra-fast stats (no detailed tracking)
// Only tracks total requests, active requests, and bytes - maximum performance
func (wlr *wlRoutes) statsFastMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fast atomic increment (no locks)
		wlr.wl.Stats.RecordRequestFast()

		// Minimal response wrapper for byte counting only
		srw := &statsFastResponseWriter{
			ResponseWriter: w,
		}

		// Serve the request
		handler.ServeHTTP(srw, r)

		// Fast atomic operations (no locks)
		wlr.wl.Stats.RecordResponseFast(srw.bytesWritten)
	})
}

// statsResponseWriter wraps http.ResponseWriter to capture response details
type statsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten uint64
}

func (w *statsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += uint64(n)
	return n, err
}

// statsFastResponseWriter is a lightweight wrapper that only counts bytes
type statsFastResponseWriter struct {
	http.ResponseWriter
	bytesWritten uint64
}

func (w *statsFastResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += uint64(n)
	return n, err
}

// Helper functions for SetBindAddrsWithPorts

// applyDefaultPortToAddrs adds the default port to addresses that don't have a port specified.
// It handles both IPv4 (e.g., "0.0.0.0") and IPv6 (e.g., "[::]") address formats.
// If an address already has a port (e.g., "0.0.0.0:8080" or "[::]:8080"), it's left unchanged.
func applyDefaultPortToAddrs(addrs []string, defaultPort string) []string {
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		// Check if address already has a port
		hasPort := false

		// For IPv6 addresses in brackets like [::] or [::1]
		if strings.HasPrefix(addr, "[") {
			// Check if there's a port after the closing bracket
			if closingBracket := strings.Index(addr, "]"); closingBracket != -1 {
				if closingBracket < len(addr)-1 && addr[closingBracket+1] == ':' {
					hasPort = true
				}
			}
		} else if strings.Contains(addr, ":") {
			// For IPv4 or IPv6 addresses with colons
			// Simple heuristic: if there's only one colon, it's IPv4:port
			// If there are multiple colons, it's IPv6 without port
			colonCount := strings.Count(addr, ":")
			if colonCount == 1 {
				// IPv4 with port like "0.0.0.0:8080"
				hasPort = true
			}
			// If colonCount > 1, it's IPv6 without brackets and without port
		}

		if hasPort {
			result[i] = addr
		} else {
			// Add default port
			if strings.HasPrefix(addr, "[") {
				// IPv6 with brackets but no port: append :port after the closing bracket
				result[i] = addr + ":" + defaultPort
			} else if strings.Contains(addr, ":") {
				// IPv6 without brackets (multiple colons): wrap in brackets and add port
				result[i] = "[" + addr + "]:" + defaultPort
			} else {
				// IPv4 without port: append :port
				result[i] = addr + ":" + defaultPort
			}
		}
	}
	return result
}

// filterRedundantAddrs removes redundant bind addresses.
// If both 0.0.0.0 and :: are present (with the same port), keeps only :: since it typically
// binds to both IPv4 and IPv6 on most systems (unless IPV6_V6ONLY is set).
func filterRedundantAddrs(addrs []string) []string {
	// Group addresses by port to check for redundancy within each port
	portGroups := make(map[string][]string) // port -> []addresses

	for _, addr := range addrs {
		// Extract port from address
		var port string

		// Handle IPv6 with brackets like [::]:2000
		if strings.HasPrefix(addr, "[") {
			if closingBracket := strings.Index(addr, "]"); closingBracket != -1 {
				if closingBracket < len(addr)-1 && addr[closingBracket+1] == ':' {
					port = addr[closingBracket+2:]
				}
			}
		} else if strings.Contains(addr, ":") {
			// IPv4 with port like 0.0.0.0:2000
			parts := strings.Split(addr, ":")
			if len(parts) == 2 {
				port = parts[1]
			}
		}

		// If no port was extracted, use "default" as the port key
		if port == "" {
			port = "default"
		}

		portGroups[port] = append(portGroups[port], addr)
	}

	// Filter redundant addresses within each port group
	result := make([]string, 0, len(addrs))
	for _, group := range portGroups {
		hasIPv4Any := false
		hasIPv6Any := false
		var ipv4AnyAddr string

		// Check what we have in this port group
		for _, addr := range group {
			if strings.HasPrefix(addr, "0.0.0.0") || addr == "0.0.0.0" {
				hasIPv4Any = true
				ipv4AnyAddr = addr
			} else if strings.HasPrefix(addr, "[::]") || addr == "::" {
				hasIPv6Any = true
			}
		}

		// If both IPv4 and IPv6 wildcards exist for this port, filter out IPv4
		if hasIPv4Any && hasIPv6Any {
			for _, addr := range group {
				if addr != ipv4AnyAddr {
					result = append(result, addr)
				}
			}
		} else {
			result = append(result, group...)
		}
	}

	return result
}
