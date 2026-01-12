package heimdall

import (
	"net"
	"net/http"
	"strings"

	"github.com/mssola/useragent"
)

// ExtractDeviceInfo extracts device information from an HTTP request.
func ExtractDeviceInfo(r *http.Request) DeviceInfo {
	ua := r.UserAgent()
	ip := extractIP(r)

	// Parse user agent
	parsed := useragent.New(ua)
	browser, browserVersion := parsed.Browser()
	if browserVersion != "" {
		browser = browser + " " + browserVersion
	}

	osInfo := parsed.OSInfo()
	os := osInfo.Name
	if osInfo.Version != "" {
		os = os + " " + osInfo.Version
	}

	// Determine device type
	deviceType := "desktop"
	if parsed.Mobile() {
		deviceType = "mobile"
	} else if parsed.Bot() {
		deviceType = "bot"
	} else if isTablet(ua) {
		deviceType = "tablet"
	}

	return DeviceInfo{
		IP:         ip,
		UserAgent:  ua,
		Browser:    browser,
		OS:         os,
		DeviceType: deviceType,
	}
}

// extractIP extracts the client IP from an HTTP request.
// It checks common proxy headers first, then falls back to RemoteAddr.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if isValidIP(ip) {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if isValidIP(ip) {
			return ip
		}
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		ip := strings.TrimSpace(cfip)
		if isValidIP(ip) {
			return ip
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return host
}

// isValidIP checks if the string is a valid IP address.
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// isTablet checks if the user agent indicates a tablet device.
func isTablet(ua string) bool {
	ua = strings.ToLower(ua)
	tabletKeywords := []string{"ipad", "tablet", "playbook", "silk"}
	for _, keyword := range tabletKeywords {
		if strings.Contains(ua, keyword) {
			return true
		}
	}
	return false
}

// IsPrivateIP returns true if the IP is in a private/reserved range.
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	// Check for loopback
	if parsed.IsLoopback() {
		return true
	}

	// Check for private ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 unique local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsed) {
			return true
		}
	}

	return false
}
