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

	"github.com/go-xlite/wbx/routes"
	"github.com/gorilla/mux"
)

// WebLite represents a lightweight web server instance
type WebLite struct {
	Provider            *WebLiteProvider
	Name                string
	Mux                 *mux.Router
	Routes              *routes.Routes
	Port                string
	BindAddr            []string
	SslCert             string
	SslKey              string
	CloudFlareOptimized bool

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
		Name:                name,
		Mux:                 mux.NewRouter(),
		Port:                "8080",
		BindAddr:            []string{"0.0.0.0", "::"}, // Default to dual-stack (IPv4 + IPv6)
		servers:             make([]*http.Server, 0),
		stopChan:            make(chan struct{}),
		CloudFlareOptimized: false,
	}
	wl.Routes = routes.NewRoutes(wl.Mux)
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
	cloudFlareOptimized := wl.CloudFlareOptimized
	wl.mu.Unlock()

	fmt.Printf("WebLite [%s] starting on %s", wl.Name, addr)
	if cloudFlareOptimized {
		fmt.Printf(" (CloudFlare optimized)")
	}
	fmt.Println()

	// If CloudFlare optimizations are enabled, create custom listener
	if cloudFlareOptimized {
		listener, err := wl.CreateCloudFlareListener("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to create CloudFlare listener: %w", err)
		}
		defer listener.Close()

		// Configure TLS if raw certificate data is provided
		if useTLSFromData {
			cert, err := tls.X509KeyPair(certData, keyData)
			if err != nil {
				return fmt.Errorf("failed to parse certificate and key: %w", err)
			}

			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			tlsListener := tls.NewListener(listener, server.TLSConfig)
			return server.Serve(tlsListener)
		}

		// Use file paths if provided
		if useTLSFromFiles {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return fmt.Errorf("failed to load certificate and key: %w", err)
			}

			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			tlsListener := tls.NewListener(listener, server.TLSConfig)
			return server.Serve(tlsListener)
		}

		// Start regular HTTP server with CloudFlare listener
		return server.Serve(listener)
	}

	// Standard listener (no CloudFlare optimizations)
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

// GetRoutes returns the Routes instance
func (wl *WebLite) GetRoutes() *routes.Routes {
	return wl.Routes
}

// GetMux returns the mux.Router instance
func (wl *WebLite) GetMux() *mux.Router {
	return wl.Mux
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
