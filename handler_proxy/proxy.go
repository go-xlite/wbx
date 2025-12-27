package handlerproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// ProxyHandler provides reverse proxy functionality
type ProxyHandler struct {
	*handler_role.HandlerRole
	targets         []*url.URL
	currentTarget   int
	mu              sync.RWMutex
	Timeout         time.Duration
	PreserveHost    bool
	StripPrefix     string
	AddPrefix       string
	CustomHeaders   map[string]string
	RemoveHeaders   []string
	OnRequest       func(r *http.Request)
	OnResponse      func(r *http.Response) error
	OnError         func(w http.ResponseWriter, r *http.Request, err error)
	FollowRedirects bool
	LoadBalanceMode string // "round-robin", "random", "first"
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(handler handler_role.IHandler, targetURL string) (*ProxyHandler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	handlerRole := &handler_role.HandlerRole{Handler: handler}
	handlerRole.SetPathPrefix("/")
	ph := &ProxyHandler{
		HandlerRole:     handlerRole,
		targets:         []*url.URL{target},
		Timeout:         30 * time.Second,
		PreserveHost:    false,
		CustomHeaders:   make(map[string]string),
		RemoveHeaders:   []string{},
		FollowRedirects: true,
		LoadBalanceMode: "round-robin",
	}

	return ph, nil
}

// AddTarget adds an additional target for load balancing
func (ph *ProxyHandler) AddTarget(targetURL string) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	ph.mu.Lock()
	ph.targets = append(ph.targets, target)
	ph.mu.Unlock()

	return nil
}

// SetTimeout sets the proxy timeout
func (ph *ProxyHandler) SetTimeout(timeout time.Duration) *ProxyHandler {
	ph.Timeout = timeout
	return ph
}

// SetPreserveHost sets whether to preserve the original Host header
func (ph *ProxyHandler) SetPreserveHost(preserve bool) *ProxyHandler {
	ph.PreserveHost = preserve
	return ph
}

// SetStripPrefix sets a prefix to strip from the request path
func (ph *ProxyHandler) SetStripPrefix(prefix string) *ProxyHandler {
	ph.StripPrefix = prefix
	return ph
}

// SetAddPrefix sets a prefix to add to the request path
func (ph *ProxyHandler) SetAddPrefix(prefix string) *ProxyHandler {
	ph.AddPrefix = prefix
	return ph
}

// AddHeader adds a custom header to all proxied requests
func (ph *ProxyHandler) AddHeader(key, value string) *ProxyHandler {
	ph.mu.Lock()
	ph.CustomHeaders[key] = value
	ph.mu.Unlock()
	return ph
}

// RemoveHeader removes a header from all proxied requests
func (ph *ProxyHandler) RemoveHeader(key string) *ProxyHandler {
	ph.mu.Lock()
	ph.RemoveHeaders = append(ph.RemoveHeaders, key)
	ph.mu.Unlock()
	return ph
}

// SetLoadBalanceMode sets the load balancing mode
func (ph *ProxyHandler) SetLoadBalanceMode(mode string) *ProxyHandler {
	ph.LoadBalanceMode = mode
	return ph
}

// getNextTarget returns the next target based on load balancing mode
func (ph *ProxyHandler) getNextTarget() *url.URL {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if len(ph.targets) == 0 {
		return nil
	}

	if len(ph.targets) == 1 {
		return ph.targets[0]
	}

	switch ph.LoadBalanceMode {
	case "round-robin":
		target := ph.targets[ph.currentTarget]
		ph.currentTarget = (ph.currentTarget + 1) % len(ph.targets)
		return target
	case "first":
		return ph.targets[0]
	default:
		return ph.targets[0]
	}
}

// HandleProxy creates an HTTP handler for the proxy
func (ph *ProxyHandler) HandleProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := ph.getNextTarget()
		if target == nil {
			http.Error(w, "No proxy targets configured", http.StatusInternalServerError)
			return
		}

		// Create a reverse proxy for this request
		proxy := ph.createReverseProxy(target)
		proxy.ServeHTTP(w, r)
	}
}

// createReverseProxy creates a reverse proxy for the given target
func (ph *ProxyHandler) createReverseProxy(target *url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		// Preserve original URL for reference
		originalHost := req.Host

		// Set target scheme and host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// Handle path modifications
		if ph.StripPrefix != "" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, ph.StripPrefix)
		}
		if ph.AddPrefix != "" {
			req.URL.Path = ph.AddPrefix + req.URL.Path
		}

		// If no path modifications, use target path as base
		if ph.StripPrefix == "" && ph.AddPrefix == "" && target.Path != "" {
			req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		}

		// Preserve or replace host header
		if ph.PreserveHost {
			req.Host = originalHost
		} else {
			req.Host = target.Host
		}

		// Apply custom headers
		ph.mu.RLock()
		for key, value := range ph.CustomHeaders {
			req.Header.Set(key, value)
		}

		// Remove specified headers
		for _, key := range ph.RemoveHeaders {
			req.Header.Del(key)
		}
		ph.mu.RUnlock()

		// Set standard proxy headers
		if clientIP, _, ok := splitHostPort(req.RemoteAddr); ok {
			if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
				clientIP = prior + ", " + clientIP
			}
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		req.Header.Set("X-Forwarded-Proto", getScheme(req))
		req.Header.Set("X-Forwarded-Host", originalHost)
		req.Header.Set("X-Real-IP", req.RemoteAddr)

		// Call custom request modifier if set
		if ph.OnRequest != nil {
			ph.OnRequest(req)
		}
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableCompression:  false,
		},
	}

	// Set custom response modifier if provided
	if ph.OnResponse != nil {
		proxy.ModifyResponse = ph.OnResponse
	}

	// Set custom error handler if provided
	if ph.OnError != nil {
		proxy.ErrorHandler = ph.OnError
	} else {
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		}
	}

	return proxy
}

// ProxyPass sets up a simple proxy pass for a given path
func (ph *ProxyHandler) ProxyPass(path string) {
	fullPath := ph.PathPrefix.Get() + path
	ph.Handler.GetRoutes().HandlePathPrefixFn(fullPath, ph.HandleProxy())
}

// Helper functions

// singleJoiningSlash joins two URL paths with a single slash
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// splitHostPort splits host:port string
func splitHostPort(hostport string) (host, port string, ok bool) {
	host, port, ok = strings.Cut(hostport, ":")
	if !ok {
		return hostport, "", false
	}
	return host, port, true
}

// getScheme returns the request scheme (http or https)
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// ProxyConfig provides a fluent configuration interface
type ProxyConfig struct {
	TargetURL         string
	Timeout           time.Duration
	PreserveHost      bool
	StripPrefix       string
	AddPrefix         string
	CustomHeaders     map[string]string
	RemoveHeaders     []string
	LoadBalanceMode   string
	AdditionalTargets []string
}

// NewProxyFromConfig creates a proxy handler from a configuration
func NewProxyFromConfig(handler handler_role.IHandler, config ProxyConfig) (*ProxyHandler, error) {
	ph, err := NewProxyHandler(handler, config.TargetURL)
	if err != nil {
		return nil, err
	}

	if config.Timeout > 0 {
		ph.SetTimeout(config.Timeout)
	}
	ph.SetPreserveHost(config.PreserveHost)
	ph.SetStripPrefix(config.StripPrefix)
	ph.SetAddPrefix(config.AddPrefix)
	ph.SetLoadBalanceMode(config.LoadBalanceMode)

	for key, value := range config.CustomHeaders {
		ph.AddHeader(key, value)
	}

	for _, header := range config.RemoveHeaders {
		ph.RemoveHeader(header)
	}

	for _, target := range config.AdditionalTargets {
		if err := ph.AddTarget(target); err != nil {
			return nil, err
		}
	}

	return ph, nil
}
