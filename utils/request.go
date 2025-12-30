package helpers

import (
	"net/http"
	"strings"
)

type ClientInfo struct {
	UserAgent       string
	IsChrome        bool
	IsFirefox       bool
	IsEdge          bool
	IsSafari        bool
	MaySupportHTTP3 bool
	IsCurl          bool
	HTTP3Compiled   bool
	IsMobile        bool
	Os              string
}

func (ci *ClientInfo) IsBlink() bool {
	return ci.IsChrome || ci.IsEdge
}
func (ci *ClientInfo) IsWebKit() bool {
	return ci.IsSafari
}
func (ci *ClientInfo) IsGecko() bool {
	return ci.IsFirefox
}

// GetClientInfo returns information about the client's HTTP/3 support
func GetClientInfo(r *http.Request) *ClientInfo {
	userAgent := r.UserAgent()
	lowerUA := strings.ToLower(userAgent)

	// Check if the client is likely to support HTTP/3 based on user agent
	isChrome := strings.Contains(lowerUA, "chrome")
	isFirefox := strings.Contains(lowerUA, "firefox")
	isEdge := strings.Contains(lowerUA, "edg")
	isSafari := strings.Contains(lowerUA, "safari") && !isChrome

	// Detect mobile devices
	isMobile := detectMobile(lowerUA)

	// Detect OS
	clientOs := detectOS(userAgent)

	return &ClientInfo{
		UserAgent:       userAgent,
		IsChrome:        isChrome,
		IsFirefox:       isFirefox,
		IsEdge:          isEdge,
		IsSafari:        isSafari,
		MaySupportHTTP3: isChrome || isFirefox || isEdge || isSafari,
		IsCurl:          strings.Contains(lowerUA, "curl"),
		HTTP3Compiled:   true,
		IsMobile:        isMobile,
		Os:              clientOs,
	}
}

// detectMobile checks if the user agent indicates a mobile device
func detectMobile(lowerUA string) bool {
	mobileKeywords := []string{
		"mobile", "android", "iphone", "ipad", "ipod",
		"blackberry", "windows phone", "webos", "opera mini",
		"opera mobi", "iemobile", "kindle", "silk", "fennec",
	}

	for _, keyword := range mobileKeywords {
		if strings.Contains(lowerUA, keyword) {
			return true
		}
	}

	return false
}

// detectOS detects the operating system from the user agent
func detectOS(userAgent string) string {
	lowerUA := strings.ToLower(userAgent)

	// Check for mobile OS first
	if strings.Contains(lowerUA, "android") {
		return "Android"
	}
	if strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "ipad") || strings.Contains(lowerUA, "ipod") {
		return "iOS"
	}

	// Check for desktop OS
	if strings.Contains(lowerUA, "windows nt 10.0") {
		return "Windows 10/11"
	}
	if strings.Contains(lowerUA, "windows nt 6.3") {
		return "Windows 8.1"
	}
	if strings.Contains(lowerUA, "windows nt 6.2") {
		return "Windows 8"
	}
	if strings.Contains(lowerUA, "windows nt 6.1") {
		return "Windows 7"
	}
	if strings.Contains(lowerUA, "windows") {
		return "Windows"
	}

	if strings.Contains(lowerUA, "mac os x") || strings.Contains(lowerUA, "macos") {
		return "macOS"
	}

	if strings.Contains(lowerUA, "linux") {
		if strings.Contains(lowerUA, "ubuntu") {
			return "Ubuntu"
		}
		if strings.Contains(lowerUA, "fedora") {
			return "Fedora"
		}
		if strings.Contains(lowerUA, "debian") {
			return "Debian"
		}
		return "Linux"
	}

	if strings.Contains(lowerUA, "cros") {
		return "ChromeOS"
	}

	return "Unknown"
}
