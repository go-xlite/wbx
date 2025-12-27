package compressor

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

// CompressionLevel controls how aggressively to compress
type CompressionLevel int

const (
	CompressionDefault       CompressionLevel = gzip.DefaultCompression
	CompressionBestSpeed     CompressionLevel = gzip.BestSpeed
	CompressionBestSize      CompressionLevel = gzip.BestCompression
	CompressionNoCompression CompressionLevel = gzip.NoCompression
)

// Config holds compression configuration
type Config struct {
	Level             CompressionLevel
	MinSize           int  // Minimum size in bytes to compress (default: 1024)
	Enabled           bool // Whether compression is enabled
	CompressibleTypes map[string]bool
}

// DefaultConfig returns a default compression configuration
func DefaultConfig() *Config {
	return &Config{
		Level:             CompressionDefault,
		MinSize:           1024,
		Enabled:           true,
		CompressibleTypes: defaultCompressibleTypes(),
	}
}

// defaultCompressibleTypes returns the default map of compressible content types
func defaultCompressibleTypes() map[string]bool {
	return map[string]bool{
		"text/html":                true,
		"text/plain":               true,
		"text/css":                 true,
		"text/javascript":          true,
		"text/xml":                 true,
		"application/javascript":   true,
		"application/x-javascript": true,
		"application/json":         true,
		"application/xml":          true,
		"application/xhtml+xml":    true,
		"application/rss+xml":      true,
		"application/atom+xml":     true,
		"image/svg+xml":            true,
	}
}

// gzipResponseWriter wraps http.ResponseWriter to provide gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	config         *Config
	gzipWriter     *gzip.Writer
	headerWritten  bool
	shouldCompress bool
	closed         bool
}

// Write implements io.Writer
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		// Set content type if not already set
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}

		// Determine if we should compress based on content type
		w.shouldCompress = w.isCompressible()

		if w.shouldCompress {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Del("Content-Length") // Length will change with compression
		}

		w.headerWritten = true
	}

	if w.shouldCompress {
		return w.gzipWriter.Write(b)
	}

	return w.ResponseWriter.Write(b)
}

// WriteHeader implements http.ResponseWriter
func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// Flush implements http.Flusher
func (w *gzipResponseWriter) Flush() {
	if w.shouldCompress && w.gzipWriter != nil {
		w.gzipWriter.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker
func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter does not support Hijack")
}

// Close closes the gzip writer
func (w *gzipResponseWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if w.shouldCompress && w.gzipWriter != nil {
		return w.gzipWriter.Close()
	}
	return nil
}

// isCompressible checks if the response should be compressed based on content type
func (w *gzipResponseWriter) isCompressible() bool {
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		return false
	}

	// Extract MIME type without parameters
	mimeType := strings.TrimSpace(strings.Split(contentType, ";")[0])

	// Check if it's in the configured list
	if w.config.CompressibleTypes[mimeType] {
		return true
	}

	// Check for prefixes
	return strings.HasPrefix(mimeType, "text/") ||
		strings.HasPrefix(mimeType, "application/json") ||
		strings.HasPrefix(mimeType, "application/xml")
}

// Compressor provides compression middleware
type Compressor struct {
	config *Config
}

// New creates a new compressor with default configuration
func New() *Compressor {
	return &Compressor{
		config: DefaultConfig(),
	}
}

// NewWithConfig creates a new compressor with custom configuration
func NewWithConfig(config *Config) *Compressor {
	return &Compressor{
		config: config,
	}
}

// SetLevel sets the compression level
func (c *Compressor) SetLevel(level CompressionLevel) *Compressor {
	c.config.Level = level
	return c
}

// SetMinSize sets the minimum size for compression
func (c *Compressor) SetMinSize(size int) *Compressor {
	c.config.MinSize = size
	return c
}

// Enable enables compression
func (c *Compressor) Enable() *Compressor {
	c.config.Enabled = true
	return c
}

// Disable disables compression
func (c *Compressor) Disable() *Compressor {
	c.config.Enabled = false
	return c
}

// Handler returns an HTTP middleware handler for compression
func (c *Compressor) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if compression is disabled
		if !c.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip if client doesn't accept gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip if already compressed
		if w.Header().Get("Content-Encoding") != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip writer
		gz, err := gzip.NewWriterLevel(w, int(c.config.Level))
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		// Wrap response writer
		gzw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
			config:         c.config,
			gzipWriter:     gz,
			headerWritten:  false,
			shouldCompress: false,
			closed:         false,
		}
		defer gzw.Close()

		// Call next handler
		next.ServeHTTP(gzw, r)
	})
}

// HandlerFunc returns an HTTP middleware handler func for compression
func (c *Compressor) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.Handler(next).ServeHTTP(w, r)
	}
}

// Wrap wraps a response writer with compression support
func (c *Compressor) Wrap(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, func() error) {
	// Skip if compression is disabled
	if !c.config.Enabled {
		return w, func() error { return nil }
	}

	// Skip if client doesn't accept gzip
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return w, func() error { return nil }
	}

	// Skip if already compressed
	if w.Header().Get("Content-Encoding") != "" {
		return w, func() error { return nil }
	}

	// Create gzip writer
	gz, err := gzip.NewWriterLevel(w, int(c.config.Level))
	if err != nil {
		return w, func() error { return nil }
	}

	// Wrap response writer
	gzw := &gzipResponseWriter{
		Writer:         gz,
		ResponseWriter: w,
		config:         c.config,
		gzipWriter:     gz,
		headerWritten:  false,
		shouldCompress: false,
		closed:         false,
	}

	return gzw, gzw.Close
}

// Utility functions

// IsCompressibleType checks if a content type should be compressed
func IsCompressibleType(contentType string) bool {
	if contentType == "" {
		return false
	}

	mimeType := strings.TrimSpace(strings.Split(contentType, ";")[0])

	compressibleTypes := defaultCompressibleTypes()
	if compressibleTypes[mimeType] {
		return true
	}

	return strings.HasPrefix(mimeType, "text/") ||
		strings.HasPrefix(mimeType, "application/json") ||
		strings.HasPrefix(mimeType, "application/xml")
}

// AcceptsGzip checks if the request accepts gzip encoding
func AcceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}
