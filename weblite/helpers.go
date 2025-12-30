package weblite

import "strings"

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
