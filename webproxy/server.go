package webproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-xlite/wbx/comm"
)

// ProxyStats tracks statistics for the proxy server
type ProxyStats struct {
	TotalRequests      int64     `json:"totalRequests"`
	SuccessfulRequests int64     `json:"successfulRequests"`
	FailedRequests     int64     `json:"failedRequests"`
	BytesProxied       int64     `json:"bytesProxied"`
	LastRequestTime    time.Time `json:"lastRequestTime"`
}

// Webproxy represents a reverse proxy server
type Webproxy struct {
	*comm.ServerCore
	PathBase string
	NotFound http.HandlerFunc

	// Proxy specific fields
	targets       []*url.URL
	currentTarget int
	mu            sync.RWMutex
	stats         ProxyStats
	statsMu       sync.RWMutex

	// Configuration
	Timeout         time.Duration
	PreserveHost    bool
	StripPrefix     string
	AddPrefix       string
	CustomHeaders   map[string]string
	RemoveHeaders   []string
	RequestModifier func(r *http.Request)
	ResponseHandler func(r *http.Response) error
	ErrorHandler    func(w http.ResponseWriter, r *http.Request, err error)
	FollowRedirects bool
	LoadBalanceMode string // "round-robin", "random", "first"
}

// NewWebproxy creates a new Webproxy instance
func NewWebproxy(targetURL string) (*Webproxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	wp := &Webproxy{
		ServerCore:      comm.NewServerCore(),
		PathBase:        "/",
		targets:         []*url.URL{target},
		Timeout:         30 * time.Second,
		PreserveHost:    false,
		CustomHeaders:   make(map[string]string),
		RemoveHeaders:   []string{},
		FollowRedirects: true,
		LoadBalanceMode: "round-robin",
		stats:           ProxyStats{},
	}

	// Register default proxy route
	wp.Routes.HandlePathPrefixFn("/", wp.handleProxy)

	return wp, nil
}

// OnRequest handles incoming HTTP requests
func (wp *Webproxy) OnRequest(w http.ResponseWriter, r *http.Request) {
	wp.Mux.ServeHTTP(w, r)
}

// AddTarget adds an additional target for load balancing
func (wp *Webproxy) AddTarget(targetURL string) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	wp.mu.Lock()
	wp.targets = append(wp.targets, target)
	wp.mu.Unlock()

	return nil
}

// SetTimeout sets the proxy timeout
func (wp *Webproxy) SetTimeout(timeout time.Duration) *Webproxy {
	wp.Timeout = timeout
	return wp
}

// SetPreserveHost sets whether to preserve the original Host header
func (wp *Webproxy) SetPreserveHost(preserve bool) *Webproxy {
	wp.PreserveHost = preserve
	return wp
}

// SetStripPrefix sets a prefix to strip from the request path
func (wp *Webproxy) SetStripPrefix(prefix string) *Webproxy {
	wp.StripPrefix = prefix
	return wp
}

// SetAddPrefix sets a prefix to add to the request path
func (wp *Webproxy) SetAddPrefix(prefix string) *Webproxy {
	wp.AddPrefix = prefix
	return wp
}

// AddHeader adds a custom header to all proxied requests
func (wp *Webproxy) AddHeader(key, value string) *Webproxy {
	wp.mu.Lock()
	wp.CustomHeaders[key] = value
	wp.mu.Unlock()
	return wp
}

// RemoveHeader removes a header from all proxied requests
func (wp *Webproxy) RemoveHeader(key string) *Webproxy {
	wp.mu.Lock()
	wp.RemoveHeaders = append(wp.RemoveHeaders, key)
	wp.mu.Unlock()
	return wp
}

// SetLoadBalanceMode sets the load balancing mode
func (wp *Webproxy) SetLoadBalanceMode(mode string) *Webproxy {
	wp.LoadBalanceMode = mode
	return wp
}

// getNextTarget returns the next target based on load balancing mode
func (wp *Webproxy) getNextTarget() *url.URL {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.targets) == 0 {
		return nil
	}

	if len(wp.targets) == 1 {
		return wp.targets[0]
	}

	switch wp.LoadBalanceMode {
	case "round-robin":
		target := wp.targets[wp.currentTarget]
		wp.currentTarget = (wp.currentTarget + 1) % len(wp.targets)
		return target
	case "first":
		return wp.targets[0]
	default:
		return wp.targets[0]
	}
}

// handleProxy handles the actual proxying
func (wp *Webproxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	wp.statsMu.Lock()
	wp.stats.TotalRequests++
	wp.stats.LastRequestTime = time.Now()
	wp.statsMu.Unlock()

	target := wp.getNextTarget()
	if target == nil {
		http.Error(w, "No proxy targets configured", http.StatusInternalServerError)
		wp.statsMu.Lock()
		wp.stats.FailedRequests++
		wp.statsMu.Unlock()
		return
	}

	// Create a reverse proxy for this request
	proxy := wp.createReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

// createReverseProxy creates a reverse proxy for the given target
func (wp *Webproxy) createReverseProxy(target *url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		// Preserve original URL for reference
		originalHost := req.Host

		// Set target scheme and host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// Handle path modifications
		if wp.StripPrefix != "" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, wp.StripPrefix)
		}
		if wp.AddPrefix != "" {
			req.URL.Path = wp.AddPrefix + req.URL.Path
		}

		// If no path modifications, use target path as base
		if wp.StripPrefix == "" && wp.AddPrefix == "" && target.Path != "" {
			req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		}

		// Preserve or replace host header
		if wp.PreserveHost {
			req.Host = originalHost
		} else {
			req.Host = target.Host
		}

		// Apply custom headers
		wp.mu.RLock()
		for key, value := range wp.CustomHeaders {
			req.Header.Set(key, value)
		}

		// Remove specified headers
		for _, key := range wp.RemoveHeaders {
			req.Header.Del(key)
		}
		wp.mu.RUnlock()

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
		if wp.RequestModifier != nil {
			wp.RequestModifier(req)
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
	if wp.ResponseHandler != nil {
		proxy.ModifyResponse = wp.ResponseHandler
	}

	// Set custom error handler if provided
	if wp.ErrorHandler != nil {
		proxy.ErrorHandler = wp.ErrorHandler
	} else {
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			wp.statsMu.Lock()
			wp.stats.FailedRequests++
			wp.statsMu.Unlock()
			http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		}
	}

	return proxy
}

// GetStats returns current proxy statistics
func (wp *Webproxy) GetStats() ProxyStats {
	wp.statsMu.RLock()
	defer wp.statsMu.RUnlock()
	return wp.stats
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
