package weblite

import (
	"net/http"
	"strings"
	"sync"
)

// PortListener represents a configured port listener with protocol and SSL settings
type PortListener struct {
	Protocol           string // "http" or "https"
	Ports              []string
	Addresses          []string
	OptimizeCloudflare bool
	SSLCertPath        string
	SSLKeyPath         string
	SSLCertData        string
	SSLKeyData         string
	HTTPSRedirectPort  string           // For HTTP listeners: redirect to this HTTPS port
	HTTPSRedirect      bool             // Automatically redirect HTTP to HTTPS when SSL is enabled (default: true)
	DomainValidator    *DomainValidator // Domain validator for validation
}

// NewPortListener creates a new PortListener from a configuration map
func NewPortListener(config map[string]string) *PortListener {
	pl := &PortListener{
		Protocol:           strings.ToLower(config["protocol"]),
		OptimizeCloudflare: config["optimizeCloudflare"] == "true", // Default false
		SSLCertPath:        config["ssl_cert_path"],
		SSLKeyPath:         config["ssl_key_path"],
		SSLCertData:        config["ssl_cert_data"],
		SSLKeyData:         config["ssl_key_data"],
		HTTPSRedirectPort:  config["https_redirect_port"],
		HTTPSRedirect:      config["https_redirect"] != "false", // Default true
	}

	// Parse ports
	if portsStr := config["ports"]; portsStr != "" {
		pl.Ports = strings.Split(portsStr, ",")
		for i := range pl.Ports {
			pl.Ports[i] = strings.TrimSpace(pl.Ports[i])
		}
	}

	// Parse addresses
	if addrsStr := config["addresses"]; addrsStr != "" {
		pl.Addresses = strings.Split(addrsStr, ",")
		for i := range pl.Addresses {
			pl.Addresses[i] = strings.TrimSpace(pl.Addresses[i])
			// Strip brackets from IPv6 addresses since net.JoinHostPort will add them
			pl.Addresses[i] = strings.Trim(pl.Addresses[i], "[]")
		}
	}

	// Default to IPv6 if no addresses specified
	if len(pl.Addresses) == 0 {
		pl.Addresses = []string{"::"}
	}

	// Initialize domain validator
	pl.DomainValidator = NewDomainValidator()

	// Parse allowed domains into validator
	if allowedStr := config["domains_allow"]; allowedStr != "" {
		domains := strings.Split(allowedStr, ",")
		for i := range domains {
			domains[i] = strings.TrimSpace(domains[i])
		}
		pl.DomainValidator.SetAllowedDomains(domains...)
	}

	// Parse disallowed domains into validator
	if disallowedStr := config["domains_block"]; disallowedStr != "" {
		domains := strings.Split(disallowedStr, ",")
		for i := range domains {
			domains[i] = strings.TrimSpace(domains[i])
		}
		pl.DomainValidator.SetDisallowedDomains(domains...)
	}

	return pl
}

// IsHTTPS returns true if this listener is configured for HTTPS
func (pl *PortListener) IsHTTPS() bool {
	return pl.Protocol == "https"
}

// HasSSLConfig returns true if SSL configuration is present
func (pl *PortListener) HasSSLConfig() bool {
	return (pl.SSLCertPath != "" && pl.SSLKeyPath != "") ||
		(pl.SSLCertData != "" && pl.SSLKeyData != "")
}

type DomainValidator struct {
	AllowedDomains    []string // If empty, accepts all domains. Supports wildcards: *.example.com, abc-*.example.com
	DisallowedDomains []string // Domains explicitly blocked (takes precedence over allowed)
	mu                sync.RWMutex
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

// SetDisallowedDomains sets the list of disallowed domains with wildcard support
func (dv *DomainValidator) SetDisallowedDomains(domains ...string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.DisallowedDomains = domains
}

// AddDisallowedDomain adds a single domain to the disallowed list
func (dv *DomainValidator) AddDisallowedDomain(domain string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.DisallowedDomains = append(dv.DisallowedDomains, domain)
}

// IsAllowed checks if a domain is allowed based on AllowedDomains patterns
func (dv *DomainValidator) IsAllowed(domain string) bool {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// Strip port from domain if present
	if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
		domain = domain[:colonIdx]
	}

	// Check disallowed domains first (takes precedence)
	if len(dv.DisallowedDomains) > 0 {
		for _, pattern := range dv.DisallowedDomains {
			if matchWildcardDomain(pattern, domain) {
				return false
			}
		}
	}

	// If no allowed domains specified, allow all (except disallowed)
	if len(dv.AllowedDomains) == 0 {
		return true
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

// IsEnabled returns true if domain validation is enabled (has allowed or disallowed domains configured)
func (dv *DomainValidator) IsEnabled() bool {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	return len(dv.AllowedDomains) > 0 || len(dv.DisallowedDomains) > 0
}
