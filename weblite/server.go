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

type DomainValidator struct {
	AllowedDomains []string // If empty, accepts all domains. Supports wildcards: *.example.com, abc-*.example.com
	mu             sync.RWMutex
}

// NewDomainValidator creates a new domain validator
func NewDomainValidator() *DomainValidator {
	return &DomainValidator{
		AllowedDomains: []string{},
	}
}

// SetAllowedDomains sets the list of allowed domains with wildcard support
func (dv *DomainValidator) SetAllowedDomains(domains ...string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.AllowedDomains = domains
}

// AddAllowedDomain adds a single domain to the allowed list
func (dv *DomainValidator) AddAllowedDomain(domain string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.AllowedDomains = append(dv.AllowedDomains, domain)
}

// IsAllowed checks if a domain is allowed based on AllowedDomains patterns
func (dv *DomainValidator) IsAllowed(domain string) bool {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// If no domains specified, allow all
	if len(dv.AllowedDomains) == 0 {
		return true
	}

	// Strip port from domain if present
	if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
		domain = domain[:colonIdx]
	}

	// Check against allowed patterns
	for _, pattern := range dv.AllowedDomains {
		if matchWildcardDomain(pattern, domain) {
			return true
		}
	}

	return false
}

// Middleware creates a middleware function for domain validation
func (dv *DomainValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !dv.IsAllowed(r.Host) {
			http.Error(w, "Domain not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IsEnabled returns true if domain validation is enabled (has allowed domains configured)
func (dv *DomainValidator) IsEnabled() bool {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	return len(dv.AllowedDomains) > 0
}

type SSL struct {
	CertFilePath string
	KeyFilePath  string
	// Raw SSL/TLS data (alternative to file paths)
	CertData []byte
	KeyData  []byte
	// Multi-domain certificate support
	Certificates map[string]*tls.Certificate // domain -> certificate
	mu           sync.RWMutex
}

// SetFromFiles configures SSL/TLS using file paths (single certificate for all domains)
func (s *SSL) SetFromFiles(certFile, keyFile string) {
	s.CertFilePath = certFile
	s.KeyFilePath = keyFile
	s.CertData = nil
	s.KeyData = nil
}

// AddCertificateForDomain adds a certificate for a specific domain
func (s *SSL) AddCertificateForDomain(domain, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate for domain %s: %w", domain, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Certificates == nil {
		s.Certificates = make(map[string]*tls.Certificate)
	}
	s.Certificates[domain] = &cert
	return nil
}

// AddCertificateForDomainFromData adds a certificate for a specific domain from raw data
func (s *SSL) AddCertificateForDomainFromData(domain string, certData, keyData []byte) error {
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return fmt.Errorf("failed to parse certificate for domain %s: %w", domain, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Certificates == nil {
		s.Certificates = make(map[string]*tls.Certificate)
	}
	s.Certificates[domain] = &cert
	return nil
}

// SetFromData configures SSL/TLS using raw certificate and key data
func (s *SSL) SetFromData(certData, keyData []byte) {
	s.CertData = certData
	s.KeyData = keyData
	s.CertFilePath = ""
	s.KeyFilePath = ""
}

// SetFromText configures SSL/TLS using raw certificate and key text
func (s *SSL) SetFromText(certText, keyText string) {
	s.SetFromData([]byte(certText), []byte(keyText))
}

// IsConfigured returns true if SSL is configured (either from files or data)
func (s *SSL) IsConfigured() bool {
	return s.HasData() || s.HasFiles()
}

// HasData returns true if SSL is configured from raw data
func (s *SSL) HasData() bool {
	return len(s.CertData) > 0 && len(s.KeyData) > 0
}

// HasFiles returns true if SSL is configured from file paths
func (s *SSL) HasFiles() bool {
	return s.CertFilePath != "" && s.KeyFilePath != ""
}

// GetTLSConfig creates and returns a TLS configuration with SNI support
func (s *SSL) GetTLSConfig() (*tls.Config, error) {
	s.mu.RLock()
	hasDomainCerts := len(s.Certificates) > 0
	s.mu.RUnlock()

	// If we have domain-specific certificates, use SNI
	if hasDomainCerts {
		config := &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				s.mu.RLock()
				defer s.mu.RUnlock()

				// Try exact match first
				if cert, ok := s.Certificates[hello.ServerName]; ok {
					return cert, nil
				}

				// Try wildcard match
				for domain, cert := range s.Certificates {
					if matchWildcardDomain(domain, hello.ServerName) {
						return cert, nil
					}
				}

				// Fall back to default certificate if available
				if s.HasData() {
					cert, err := tls.X509KeyPair(s.CertData, s.KeyData)
					if err == nil {
						return &cert, nil
					}
				}

				if s.HasFiles() {
					cert, err := tls.LoadX509KeyPair(s.CertFilePath, s.KeyFilePath)
					if err == nil {
						return &cert, nil
					}
				}

				return nil, fmt.Errorf("no certificate found for %s", hello.ServerName)
			},
		}
		return config, nil
	}

	// Single certificate configuration
	if s.HasData() {
		cert, err := tls.X509KeyPair(s.CertData, s.KeyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate and key: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}, nil
	}

	if s.HasFiles() {
		cert, err := tls.LoadX509KeyPair(s.CertFilePath, s.KeyFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate and key: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}, nil
	}

	return nil, fmt.Errorf("SSL not configured")
}

// WebLite represents a lightweight web server instance
type WebLite struct {
	Provider            *WebLiteProvider
	Name                string
	Mux                 *mux.Router
	Routes              *routes.Routes
	Port                string
	BindAddr            []string
	SSL                 *SSL
	CloudFlareOptimized bool
	DomainValidator     *DomainValidator
	HTTPSRedirect       bool // Automatically redirect HTTP to HTTPS when SSL is enabled

	// Server management
	servers []*http.Server
	running bool
	mu      sync.RWMutex
}

// NewWebLite creates a new WebLite instance with default configuration
func NewWebLite(name string) *WebLite {
	wl := &WebLite{
		Name:                name,
		Mux:                 mux.NewRouter(),
		Port:                "8080",
		BindAddr:            []string{"::"}, // Default to IPv6 (typically binds to both IPv4 and IPv6)
		servers:             make([]*http.Server, 0),
		CloudFlareOptimized: false,
		SSL:                 &SSL{},
		DomainValidator:     NewDomainValidator(),
		HTTPSRedirect:       true, // Enable HTTPS redirect by default
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

// SetHTTPSRedirect enables or disables automatic HTTP to HTTPS redirection
func (wl *WebLite) SetHTTPSRedirect(enable bool) *WebLite {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	wl.HTTPSRedirect = enable
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
	wl.running = true
	wl.mu.Unlock()

	defer func() {
		wl.mu.Lock()
		wl.running = false
		wl.mu.Unlock()
	}()

	// Start servers for all bind addresses
	type bindResult struct {
		addr string
		err  error
	}
	resultChan := make(chan bindResult, len(wl.BindAddr))
	var wg sync.WaitGroup

	for _, addr := range wl.BindAddr {
		wg.Add(1)
		go func(bindAddr string) {
			defer wg.Done()
			err := wl.startServer(bindAddr)
			if err != nil && err != http.ErrServerClosed {
				resultChan <- bindResult{addr: bindAddr, err: err}
			} else if err == nil {
				resultChan <- bindResult{addr: bindAddr, err: nil}
			}
		}(addr)
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

	// If we have errors, check if they can be safely ignored
	for _, errResult := range errors {
		// Check if this is an IPv4 bind failure after IPv6 success
		// IPv6 :: often binds to both IPv4 and IPv6, so IPv4 0.0.0.0 bind may fail
		canIgnore := false

		// Check if error is "address already in use"
		if strings.Contains(errResult.err.Error(), "address already in use") ||
			strings.Contains(errResult.err.Error(), "bind: address already in use") {
			// Check if the failed address is IPv4 wildcard
			isIPv4Wildcard := strings.HasPrefix(errResult.addr, "0.0.0.0")

			// Check if we have a successful IPv6 wildcard bind
			hasIPv6Success := false
			for _, successAddr := range successAddrs {
				if successAddr == "::" || strings.HasPrefix(successAddr, "::") {
					hasIPv6Success = true
					break
				}
			}

			// If IPv4 bind failed but IPv6 succeeded, assume dual-stack and ignore
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

	// Wrap handler with domain validation if needed
	handler := http.Handler(wl.Mux)
	if wl.DomainValidator.IsEnabled() {
		handler = wl.DomainValidator.Middleware(handler)
	}

	wl.mu.Lock()
	sslConfigured := wl.SSL.IsConfigured()
	cloudFlareOptimized := wl.CloudFlareOptimized
	httpsRedirect := wl.HTTPSRedirect
	wl.mu.Unlock()

	// Wrap handler with HTTPS redirect middleware if SSL is configured and redirect is enabled
	// This handles HTTP requests sent directly to the HTTPS port
	if sslConfigured && httpsRedirect {
		handler = wrapWithHTTPSRedirect(handler)
	}

	// Wrap handler with HTTP/3 Alt-Svc advertisement if HTTP/3 is enabled and SSL is configured
	if wl.isHTTP3Enabled() && sslConfigured {
		if port := getHTTP3Port(addr); port != "" {
			handler = wrapWithHTTP3AltSvc(handler, port)
		}
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	wl.mu.Lock()
	wl.servers = append(wl.servers, server)
	wl.mu.Unlock()

	fmt.Printf("WebLite [%s] starting on %s", wl.Name, addr)
	if cloudFlareOptimized {
		fmt.Printf(" (CloudFlare optimized)")
	}
	if wl.isHTTP3Enabled() && sslConfigured {
		fmt.Printf(" (with HTTP/3)")
	}
	fmt.Println()

	// If CloudFlare optimizations are enabled, create custom listener
	if cloudFlareOptimized {
		listener, err := wl.CreateCloudFlareListener("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to create CloudFlare listener: %w", err)
		}
		defer listener.Close()

		// Configure TLS if SSL is configured
		if sslConfigured {
			tlsConfig, err := wl.SSL.GetTLSConfig()
			if err != nil {
				return err
			}

			server.TLSConfig = tlsConfig
			tlsListener := tls.NewListener(listener, server.TLSConfig)
			return server.Serve(tlsListener)
		}

		// Start regular HTTP server with CloudFlare listener
		return server.Serve(listener)
	}

	// Standard listener (no CloudFlare optimizations)
	// Configure TLS if SSL is configured
	if sslConfigured {
		// If HTTPS redirect is enabled, start HTTP redirect server
		if httpsRedirect {
			// Calculate HTTP port (HTTPS port - 363, or default to 80)
			httpPort := wl.calculateHTTPPort()
			httpAddr := net.JoinHostPort(bindAddr, httpPort)

			// Start HTTP redirect server in background
			go wl.startHTTPRedirectServer(httpAddr, wl.Port)
		}

		var tlsConfig *tls.Config
		var err error

		if wl.SSL.HasData() {
			// Use raw data - must set TLSConfig
			tlsConfig, err = wl.SSL.GetTLSConfig()
			if err != nil {
				return err
			}
			server.TLSConfig = tlsConfig
		} else if wl.SSL.HasFiles() {
			// Get TLS config for HTTP/3
			tlsConfig, err = wl.SSL.GetTLSConfig()
			if err != nil {
				return err
			}
		}

		// If HTTPS redirect is enabled, use mixed protocol listener
		// This handles HTTP requests sent directly to the HTTPS port
		if httpsRedirect && tlsConfig != nil {
			// Create TCP listener
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("failed to create listener: %w", err)
			}
			defer ln.Close()

			// Wrap with mixed protocol listener
			mixedLn := &mixedProtocolListener{
				Listener:  ln,
				tlsConfig: tlsConfig,
				httpsPort: wl.Port,
			}

			// Wrap with TLS
			tlsLn := tls.NewListener(mixedLn, tlsConfig)

			// Start HTTP/3 if enabled
			if wl.isHTTP3Enabled() {
				go func() {
					if err := wl.startHTTP3Server(addr, tlsConfig, handler); err != nil {
						fmt.Printf("HTTP/3 server error: %v\n", err)
					}
				}()
			}

			// Serve with mixed listener
			return server.Serve(tlsLn)
		}

		// Start HTTP/3 server if enabled (runs in parallel with HTTP/1.1/2.0)
		if wl.isHTTP3Enabled() && tlsConfig != nil {
			// Start HTTP/3 server in a goroutine
			errChan := make(chan error, 2)

			go func() {
				if err := wl.startHTTP3Server(addr, tlsConfig, handler); err != nil {
					errChan <- fmt.Errorf("HTTP/3 server error: %w", err)
				}
			}()

			// Start HTTP/1.1/2.0 server
			go func() {
				var err error
				if wl.SSL.HasData() {
					err = server.ListenAndServeTLS("", "")
				} else if wl.SSL.HasFiles() {
					err = server.ListenAndServeTLS(wl.SSL.CertFilePath, wl.SSL.KeyFilePath)
				}
				if err != nil && err != http.ErrServerClosed {
					errChan <- fmt.Errorf("HTTP/1.1/2.0 server error: %w", err)
				}
			}()

			// Wait for first error (blocking)
			return <-errChan
		}

		// HTTP/3 not enabled, just start regular HTTPS
		if wl.SSL.HasData() {
			return server.ListenAndServeTLS("", "")
		}
		if wl.SSL.HasFiles() {
			return server.ListenAndServeTLS(wl.SSL.CertFilePath, wl.SSL.KeyFilePath)
		}
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

// matchWildcardDomain checks if a domain matches a wildcard pattern
// Supports patterns like:
// - *.example.com (matches any.example.com but not example.com)
// - abc-*.example.com (matches abc-xyz.example.com)
// - example.com (exact match)
func matchWildcardDomain(pattern, domain string) bool {
	// Exact match
	if pattern == domain {
		return true
	}

	// No wildcard, no match
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Convert wildcard pattern to segments
	patternParts := strings.Split(pattern, ".")
	domainParts := strings.Split(domain, ".")

	// Must have same number of segments
	if len(patternParts) != len(domainParts) {
		return false
	}

	// Match each segment
	for i := 0; i < len(patternParts); i++ {
		patternSegment := patternParts[i]
		domainSegment := domainParts[i]

		if !matchWildcardSegment(patternSegment, domainSegment) {
			return false
		}
	}

	return true
}

// matchWildcardSegment matches a single segment with wildcard support
// Supports patterns like: *, abc-*, *-xyz, abc-*-xyz
func matchWildcardSegment(pattern, segment string) bool {
	// Exact match or pure wildcard
	if pattern == segment || pattern == "*" {
		return true
	}

	// No wildcard, no match
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Split by wildcard and match parts
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]

		// Check if segment starts with prefix and ends with suffix
		if len(segment) < len(prefix)+len(suffix) {
			return false
		}

		if prefix != "" && !strings.HasPrefix(segment, prefix) {
			return false
		}

		if suffix != "" && !strings.HasSuffix(segment, suffix) {
			return false
		}

		return true
	}

	// For more complex patterns with multiple wildcards, use simple approach
	// Convert pattern to a regex-like match
	pos := 0
	for i, part := range parts {
		if i > 0 {
			// Skip any characters for the wildcard
			if part == "" {
				continue
			}
			idx := strings.Index(segment[pos:], part)
			if idx == -1 {
				return false
			}
			pos += idx + len(part)
		} else {
			// First part must match at the beginning
			if !strings.HasPrefix(segment, part) {
				return false
			}
			pos = len(part)
		}
	}

	return true
}

// parseHostPort splits a network address of the form "host:port" or "[host]:port"
// into host and port. This is similar to net.SplitHostPort but doesn't return an error
// for addresses without ports.
func parseHostPort(addr string) (host, port string, err error) {
	// Try standard parsing first
	h, p, err := net.SplitHostPort(addr)
	if err == nil {
		return h, p, nil
	}

	// If it failed, it might be because there's no port
	// Return the address as host with empty port
	return addr, "", nil
}

// calculateHTTPPort returns the HTTP port to use for redirects
// If HTTPS port is standard (443), use 80
// Otherwise, calculate as HTTPS_PORT - 363 (8080 -> 7717, 8443 -> 8080)
func (wl *WebLite) calculateHTTPPort() string {
	wl.mu.RLock()
	httpsPort := wl.Port
	wl.mu.RUnlock()

	if httpsPort == "443" || httpsPort == "" {
		return "80"
	}

	// For common dev ports, use sensible defaults
	switch httpsPort {
	case "8443":
		return "8080"
	case "9443":
		return "9080"
	case "8080":
		return "7717" // 8080 - 363
	default:
		// Try to parse as number and subtract 363
		var portNum int
		if _, err := fmt.Sscanf(httpsPort, "%d", &portNum); err == nil && portNum > 363 {
			return fmt.Sprintf("%d", portNum-363)
		}
		return "80" // Fallback
	}
}

// startHTTPRedirectServer starts an HTTP server that redirects all traffic to HTTPS
func (wl *WebLite) startHTTPRedirectServer(httpAddr, httpsPort string) {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract host without port
		host := r.Host
		if h, _, err := net.SplitHostPort(r.Host); err == nil {
			host = h
		}

		// Build HTTPS URL
		var httpsURL string
		if httpsPort == "443" {
			httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
		} else {
			httpsURL = fmt.Sprintf("https://%s:%s%s", host, httpsPort, r.RequestURI)
		}

		// Permanent redirect
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: redirectHandler,
	}

	wl.mu.Lock()
	wl.servers = append(wl.servers, httpServer)
	wl.mu.Unlock()

	fmt.Printf("WebLite [%s] HTTP->HTTPS redirect server starting on %s\n", wl.Name, httpAddr)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("WebLite [%s] HTTP redirect server error: %v\n", wl.Name, err)
	}
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
