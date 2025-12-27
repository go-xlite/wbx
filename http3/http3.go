//go:build !http3
// +build !http3

package http3

import (
	"fmt"
	"net/http"
	"strings"
)

// HTTP version constants
const (
	HTTPVersion1 = 1
	HTTPVersion2 = 2
	HTTPVersion3 = 3
)

// IsHTTP3Request checks if the current request is using HTTP/3
// Returns false when HTTP/3 is not compiled in
func IsHTTP3Request(req *http.Request) bool {
	return false
}

// GetHTTPVersionString returns a human-readable string for the HTTP version
func GetHTTPVersionString(version int) string {
	switch version {
	case HTTPVersion1:
		return "HTTP/1.1"
	case HTTPVersion2:
		return "HTTP/2.0"
	case HTTPVersion3:
		return "HTTP/3 (not compiled)"
	default:
		return "Unknown"
	}
}

// LogHTTP3Status logs that HTTP/3 is not enabled
func LogHTTP3Status() {
	fmt.Println("HTTP/3 support is not compiled in. To enable, build with -tags http3")
}

// AddHTTP3AltSvcHeader is a no-op when HTTP/3 is not compiled
func AddHTTP3AltSvcHeader(w http.ResponseWriter, port string) {
	// No-op: HTTP/3 not compiled
}

// ForceHTTP3Redirect is a no-op when HTTP/3 is not compiled
func ForceHTTP3Redirect(w http.ResponseWriter, r *http.Request, currentVersion int, preferredVersion int) bool {
	return false
}

// GetClientInfo returns information about the client
func GetClientInfo(r *http.Request) map[string]interface{} {
	userAgent := r.UserAgent()

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
		"http3Compiled":   false,
	}
}

// SuppressQuicBufferWarning is a no-op when HTTP/3 is not compiled
func SuppressQuicBufferWarning(err error) bool {
	return false
}

// GetPlatformOptimizationTips returns platform-specific tips (stub version)
func GetPlatformOptimizationTips() string {
	return "HTTP/3 is not compiled. Build with -tags http3 to enable."
}
