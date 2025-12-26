package weblite

import (
	"sync"
	"sync/atomic"
	"time"
)

// Performance notes:
// - Basic counters (totalRequests, activeRequests, totalBytes) use lock-free atomic operations
// - Detailed tracking (path/code maps) requires mutex locks and can be disabled for max performance
// - Use RecordRequestFast/RecordResponseFast for highest throughput (no detailed tracking)
// - Use HandleWithStatsFast for automatic fast stats tracking on routes
// - Disable detailed tracking with DisableDetailedStats() for production high-load scenarios

// wlStats tracks server statistics
type wlStats struct {
	wl *WebLite

	// Atomic counters (lock-free for performance)
	totalRequests  uint64
	activeRequests int64
	totalBytes     uint64

	// Detailed tracking (uses locks, optional for high-performance scenarios)
	requestsByPath map[string]uint64
	requestsByCode map[int]uint64

	// Timing
	startTime       time.Time
	lastRequestNano int64 // atomic timestamp in nanoseconds

	// Configuration
	trackDetails bool // if false, skip path/code tracking for better performance

	mu sync.RWMutex
}

// StatsSnapshot represents a point-in-time view of server statistics
type StatsSnapshot struct {
	Name           string            `json:"name"`
	IsRunning      bool              `json:"is_running"`
	TotalRequests  uint64            `json:"total_requests"`
	ActiveRequests int64             `json:"active_requests"`
	TotalBytes     uint64            `json:"total_bytes"`
	RequestsByPath map[string]uint64 `json:"requests_by_path"`
	RequestsByCode map[int]uint64    `json:"requests_by_code"`
	Uptime         time.Duration     `json:"uptime"`
	UptimeString   string            `json:"uptime_string"`
	LastRequestAt  time.Time         `json:"last_request_at,omitempty"`
	StartTime      time.Time         `json:"start_time,omitempty"`
	RequestsPerSec float64           `json:"requests_per_sec"`
}

// Initialize stats tracking
func (s *wlStats) init() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requestsByPath = make(map[string]uint64)
	s.requestsByCode = make(map[int]uint64)
	s.startTime = time.Now()
	s.trackDetails = true // enable detailed tracking by default
	atomic.StoreUint64(&s.totalRequests, 0)
	atomic.StoreInt64(&s.activeRequests, 0)
	atomic.StoreUint64(&s.totalBytes, 0)
	atomic.StoreInt64(&s.lastRequestNano, 0)
}

// Reset clears all statistics
func (s *wlStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.StoreUint64(&s.totalRequests, 0)
	atomic.StoreInt64(&s.activeRequests, 0)
	atomic.StoreUint64(&s.totalBytes, 0)
	atomic.StoreInt64(&s.lastRequestNano, 0)
	s.requestsByPath = make(map[string]uint64)
	s.requestsByCode = make(map[int]uint64)
	s.startTime = time.Now()
}

// SetDetailedTracking enables or disables detailed path/code tracking
// Disable for high-performance scenarios where only basic stats are needed
func (s *wlStats) SetDetailedTracking(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trackDetails = enabled
}

// RecordRequest increments the request counter for a specific path
// Fast path: lock-free atomic operations for basic counters
// Slow path: only if detailed tracking is enabled, acquire lock for maps
func (s *wlStats) RecordRequest(path string) {
	// Fast atomic operations (no locks)
	atomic.AddUint64(&s.totalRequests, 1)
	atomic.AddInt64(&s.activeRequests, 1)
	atomic.StoreInt64(&s.lastRequestNano, time.Now().UnixNano())

	// Detailed tracking (only if enabled) - use RLock briefly
	if s.trackDetails { // unsafe read, but acceptable for perf - worst case we miss one update
		s.mu.Lock()
		s.requestsByPath[path]++
		s.mu.Unlock()
	}
}

// RecordRequestFast increments counters without path tracking (maximum performance)
// Use this for high-throughput scenarios where path details aren't needed
func (s *wlStats) RecordRequestFast() {
	atomic.AddUint64(&s.totalRequests, 1)
	atomic.AddInt64(&s.activeRequests, 1)
	atomic.StoreInt64(&s.lastRequestNano, time.Now().UnixNano())
}

// RecordResponse records the completion of a request with status code and bytes
// Fast path: lock-free atomic operations
func (s *wlStats) RecordResponse(statusCode int, bytes uint64) {
	// Fast atomic operations (no locks)
	atomic.AddInt64(&s.activeRequests, -1)
	atomic.AddUint64(&s.totalBytes, bytes)

	// Detailed tracking (only if enabled) - use unsafe read for perf
	if s.trackDetails { // unsafe read, but acceptable for perf
		s.mu.Lock()
		s.requestsByCode[statusCode]++
		s.mu.Unlock()
	}
}

// RecordResponseFast records response completion without status code tracking (maximum performance)
func (s *wlStats) RecordResponseFast(bytes uint64) {
	atomic.AddInt64(&s.activeRequests, -1)
	atomic.AddUint64(&s.totalBytes, bytes)
}

// Get returns a snapshot of the current statistics
func (s *wlStats) Get() StatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Copy maps to avoid race conditions
	pathsCopy := make(map[string]uint64, len(s.requestsByPath))
	for k, v := range s.requestsByPath {
		pathsCopy[k] = v
	}

	codesCopy := make(map[int]uint64, len(s.requestsByCode))
	for k, v := range s.requestsByCode {
		codesCopy[k] = v
	}

	uptime := time.Duration(0)
	reqPerSec := 0.0
	if !s.startTime.IsZero() && s.wl.IsRunning() {
		uptime = time.Since(s.startTime)
		if uptime.Seconds() > 0 {
			reqPerSec = float64(atomic.LoadUint64(&s.totalRequests)) / uptime.Seconds()
		}
	}

	// Convert atomic timestamp to time.Time
	lastReqNano := atomic.LoadInt64(&s.lastRequestNano)
	var lastRequestAt time.Time
	if lastReqNano > 0 {
		lastRequestAt = time.Unix(0, lastReqNano)
	}

	return StatsSnapshot{
		Name:           s.wl.Name,
		IsRunning:      s.wl.IsRunning(),
		TotalRequests:  atomic.LoadUint64(&s.totalRequests),
		ActiveRequests: atomic.LoadInt64(&s.activeRequests),
		TotalBytes:     atomic.LoadUint64(&s.totalBytes),
		RequestsByPath: pathsCopy,
		RequestsByCode: codesCopy,
		Uptime:         uptime,
		UptimeString:   uptime.String(),
		LastRequestAt:  lastRequestAt,
		StartTime:      s.startTime,
		RequestsPerSec: reqPerSec,
	}
}

// GetTotalRequests returns the total number of requests
func (s *wlStats) GetTotalRequests() uint64 {
	return atomic.LoadUint64(&s.totalRequests)
}

// GetActiveRequests returns the number of currently active requests
func (s *wlStats) GetActiveRequests() int64 {
	return atomic.LoadInt64(&s.activeRequests)
}

// GetTotalBytes returns the total bytes transferred
func (s *wlStats) GetTotalBytes() uint64 {
	return atomic.LoadUint64(&s.totalBytes)
}

// GetUptime returns the server uptime
func (s *wlStats) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.startTime.IsZero() || !s.wl.IsRunning() {
		return 0
	}
	return time.Since(s.startTime)
}

// GetLastRequestAt returns the time of the last request
func (s *wlStats) GetLastRequestAt() time.Time {
	nanos := atomic.LoadInt64(&s.lastRequestNano)
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// GetRequestsByPath returns a copy of requests grouped by path
func (s *wlStats) GetRequestsByPath() map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]uint64, len(s.requestsByPath))
	for k, v := range s.requestsByPath {
		result[k] = v
	}
	return result
}

// GetRequestsByCode returns a copy of requests grouped by status code
func (s *wlStats) GetRequestsByCode() map[int]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int]uint64, len(s.requestsByCode))
	for k, v := range s.requestsByCode {
		result[k] = v
	}
	return result
}
