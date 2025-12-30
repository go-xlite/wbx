package helpers

// getSegment extracts a single segment from a URL path by index (0-based)
// Example: getSegment("/a/b/c", 0) returns "a", getSegment("/a/b/c", 1) returns "b"
// Returns empty string if index is out of bounds
func (s *XHelpers) GetSegment(url string, index int) string {
	// Remove leading slash if present
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}

	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	if len(url) == 0 {
		return ""
	}

	currentIndex := 0
	start := 0

	for i := 0; i <= len(url); i++ {
		if i == len(url) || url[i] == '/' {
			if currentIndex == index {
				return url[start:i]
			}
			currentIndex++
			start = i + 1
		}
	}

	return ""
}

// getSegmentRange extracts a range of segments from a URL path (from index inclusive, to index exclusive)
// Example: getSegmentRange("/a/b/c/d", 1, 3) returns "b/c"
// Returns empty string if range is invalid or out of bounds
func (s *XHelpers) GetSegmentRange(url string, from, to int) string {
	// Remove leading slash if present
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}

	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	if len(url) == 0 || from < 0 || to <= from {
		return ""
	}

	currentIndex := 0
	start := -1
	end := -1
	segmentStart := 0

	for i := 0; i <= len(url); i++ {
		if i == len(url) || url[i] == '/' {
			// We've reached the end of segment at currentIndex
			if currentIndex >= from && currentIndex < to {
				// This segment should be included
				if start == -1 {
					start = segmentStart
				}
				if i == len(url) || currentIndex == to-1 {
					end = i
				}
			}

			if currentIndex >= to {
				break
			}

			currentIndex++
			segmentStart = i + 1
		}
	}

	// Handle case where we need to extend to end
	if start >= 0 && end == -1 {
		end = len(url)
	}

	if start >= 0 && end > start {
		return url[start:end]
	}

	return ""
}

func (s *XHelpers) LengthSegments(url string) int {
	// Remove leading slash if present
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}

	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	// Empty URL after trimming means 0 segments
	if len(url) == 0 {
		return 0
	}

	// Start with 1 segment, then count additional '/' separators
	count := 1
	for i := 0; i < len(url); i++ {
		if url[i] == '/' {
			count++
		}
	}
	return count
}

func (s *XHelpers) RemoveTrailingSlash(url string) string {
	if len(url) > 0 && url[len(url)-1] == '/' {
		return url[:len(url)-1]
	}
	return url
}
func (s *XHelpers) RemoveLeadingSlash(url string) string {
	if len(url) > 0 && url[0] == '/' {
		return url[1:]
	}
	return url
}

func (s *XHelpers) RemoveBothSlashes(url string) string {
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	return url
}

// ShiftLeft removes the first N segments from the URL path
// Example: ShiftLeft("/a/b/c/d", 2) returns "/c/d"
// Optimized for performance with single-pass traversal
func (s *XHelpers) ShiftLeft(url string, segments int) string {
	// Remove leading slash if present
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}

	// Remove trailing slash if present
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	// Handle edge cases
	if len(url) == 0 {
		return "/"
	}

	if segments <= 0 {
		return "/" + url
	}

	// Single-pass: skip 'segments' number of segments
	currentIndex := 0
	for i := 0; i < len(url); i++ {
		if url[i] == '/' {
			currentIndex++
			if currentIndex == segments {
				// Return everything after this slash with leading slash
				return "/" + url[i+1:]
			}
		}
	}

	// If we haven't found enough segments, return root
	return "/"
}
