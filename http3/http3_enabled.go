//go:build http3
// +build http3

package http3

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/quic-go/quic-go/http3"
)

// HTTP version constants
const (
	HTTPVersion1 = 1
	HTTPVersion2 = 2
	HTTPVersion3 = 3
)

// IsHTTP3Request checks if the current request is using HTTP/3
func IsHTTP3Request(req *http.Request) bool {
	return req != nil && req.Context().Value(http3.ServerContextKey) != nil
}

// GetHTTPVersionString returns a human-readable string for the HTTP version
func GetHTTPVersionString(version int) string {
	switch version {
	case HTTPVersion1:
		return "HTTP/1.1"
	case HTTPVersion2:
		return "HTTP/2.0"
	case HTTPVersion3:
		return "HTTP/3"
	default:
		return "Unknown"
	}
}

// LogHTTP3Status logs information about HTTP/3 status including supported platforms
func LogHTTP3Status() {
	fmt.Printf("HTTP/3 support enabled: %s\n", GetHTTPVersionString(HTTPVersion3))

	// Log platform-specific information
	tips := GetPlatformOptimizationTips()
	if tips != "" {
		fmt.Println(tips)
	}
}

// GetPlatformOptimizationTips returns platform-specific optimization tips
func GetPlatformOptimizationTips() string {
	switch strings.ToLower(runtime.GOOS) {
	case "linux":
		return "For optimal HTTP/3 performance on Linux, consider increasing UDP buffer sizes:\n" +
			"Add to /etc/sysctl.conf: net.core.rmem_max=2500000 and net.core.wmem_max=2500000"
	case "darwin":
		return "For optimal HTTP/3 performance on macOS, consider increasing UDP buffer sizes:\n" +
			"Run: sudo sysctl -w net.inet.udp.recvspace=2500000 net.inet.udp.maxdgram=2500000"
	case "windows":
		return "HTTP/3 performance on Windows should work with default settings"
	default:
		return ""
	}
}

// AddHTTP3AltSvcHeader adds the Alt-Svc header to advertise HTTP/3 support
func AddHTTP3AltSvcHeader(w http.ResponseWriter, port string) {
	if port == "" {
		port = "443"
	}

	// First check if the header already exists to avoid duplication
	if len(w.Header().Values("Alt-Svc")) > 0 {
		return
	}

	// Use h3 and h3-29 for broader client support
	w.Header().Add("Alt-Svc", fmt.Sprintf(`h3=":%s"; ma=86400, h3-29=":%s"; ma=86400`, port, port))
}

// ForceHTTP3Redirect checks if the client is attempting HTTP/3 and handles redirects
func ForceHTTP3Redirect(w http.ResponseWriter, r *http.Request, currentVersion int, preferredVersion int) bool {
	// If client is already using the preferred version, no need to redirect
	if currentVersion == preferredVersion {
		return false
	}

	// Only force HTTP/3 headers if the client isn't already using HTTP/3
	if preferredVersion == HTTPVersion3 && currentVersion != HTTPVersion3 {
		// Add HTTP/3 Alt-Svc header
		AddHTTP3AltSvcHeader(w, "")

		// Add a header indicating preferred protocol
		w.Header().Set("X-Preferred-Protocol", "http3")
	}

	return false
}

// GetClientInfo returns information about the client's HTTP/3 support
func GetClientInfo(r *http.Request) map[string]interface{} {
	userAgent := r.UserAgent()

	// Check if the client is likely to support HTTP/3 based on user agent
	isChrome := strings.Contains(strings.ToLower(userAgent), "chrome")
	isFirefox := strings.Contains(strings.ToLower(userAgent), "firefox")
	isEdge := strings.Contains(strings.ToLower(userAgent), "edg")
	isSafari := strings.Contains(strings.ToLower(userAgent), "safari") && !isChrome

	return map[string]interface{}{
		"userAgent":       userAgent,
		"isChrome":        isChrome,
		"isFirefox":       isFirefox,
		"isEdge":          isEdge,
		"isSafari":        isSafari,
		"maySupportHTTP3": isChrome || isFirefox || isEdge || isSafari,
		"isCurl":          strings.Contains(strings.ToLower(userAgent), "curl"),
		"http3Compiled":   true,
	}
}

// SuppressQuicBufferWarning filters out the UDP buffer size warning from quic-go logs.
func SuppressQuicBufferWarning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "failed to sufficiently increase send buffer size")
}
