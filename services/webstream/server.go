package webstream

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-xlite/wbx/comm"
)

// MediaInfo contains metadata about a media file
type MediaInfo struct {
	Path        string
	Size        int64
	ModTime     time.Time
	ContentType string
	Extension   string
}

// RangeSpec represents a byte range
type RangeSpec struct {
	Start int64
	End   int64
}

// StreamConfig provides configuration for streaming
type StreamConfig struct {
	BufferSize        int
	EnableCaching     bool
	CacheDuration     time.Duration
	AllowedExtensions map[string]bool
}

// WebStream represents a media streaming server for video/audio with range request support
type WebStream struct {
	*comm.ServerCore
	PathBase          string
	NotFound          http.HandlerFunc
	FsAdapter         comm.IFsAdapter
	BufferSize        int
	EnableCaching     bool
	CacheDuration     time.Duration
	AllowedExtensions map[string]bool
}

// NewWebStream creates a new WebStream instance
func NewWebStream(fsAdapter comm.IFsAdapter) *WebStream {
	ws := &WebStream{
		ServerCore:    comm.NewServerCore(),
		PathBase:      "",
		FsAdapter:     fsAdapter,
		BufferSize:    32 * 1024, // 32KB buffer for streaming
		EnableCaching: true,
		CacheDuration: 24 * time.Hour,
		AllowedExtensions: map[string]bool{
			".mp4":  true,
			".webm": true,
			".ogg":  true,
			".mp3":  true,
			".wav":  true,
			".m4v":  true,
			".mkv":  true,
			".avi":  true,
			".mov":  true,
			".flac": true,
			".aac":  true,
			".m4a":  true,
		},
	}
	ws.NotFound = http.NotFound
	return ws
}

// NewWebStreamFromConfig creates a WebStream from configuration
func NewWebStreamFromConfig(fsAdapter comm.IFsAdapter, config StreamConfig) *WebStream {
	ws := NewWebStream(fsAdapter)

	if config.BufferSize > 0 {
		ws.BufferSize = config.BufferSize
	}

	ws.EnableCaching = config.EnableCaching
	ws.CacheDuration = config.CacheDuration

	if len(config.AllowedExtensions) > 0 {
		ws.AllowedExtensions = config.AllowedExtensions
	}

	return ws
}

// OnRequest handles an incoming HTTP request using the registered routes
func (ws *WebStream) OnRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[WebStream] OnRequest: %s %s\n", r.Method, r.URL.Path)
	ws.Mux.ServeHTTP(w, r)
}

// SetNotFoundHandler sets a custom 404 handler
func (ws *WebStream) SetNotFoundHandler(handler http.HandlerFunc) {
	ws.NotFound = handler
	ws.Mux.NotFoundHandler = handler
}

// AddAllowedExtension adds an allowed file extension
func (ws *WebStream) AddAllowedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ws.AllowedExtensions[strings.ToLower(ext)] = true
}

// ServeMedia serves a media file with range request support
func (ws *WebStream) ServeMedia(w http.ResponseWriter, r *http.Request, filePath string) {
	// Clean the file path
	cleanPath := filepath.Clean(filePath)

	// Check if file exists
	if !ws.FsAdapter.Exists(cleanPath) {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Check if it's a directory
	if ws.FsAdapter.IsDir(cleanPath) {
		http.Error(w, "Path is a directory", http.StatusForbidden)
		return
	}

	// Get file info
	info, err := ws.getMediaInfo(cleanPath)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Check if extension is allowed
	if !ws.AllowedExtensions[strings.ToLower(info.Extension)] {
		http.Error(w, "Media type not allowed", http.StatusForbidden)
		return
	}

	// Open the file
	file, err := ws.FsAdapter.Open(cleanPath)
	if err != nil {
		http.Error(w, "Cannot open media file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set common headers
	ws.setMediaHeaders(w, info)

	// Handle range requests
	if r.Header.Get("Range") != "" {
		ws.serveRangeRequest(w, r, file, info)
	} else {
		ws.serveFullContent(w, r, file, info)
	}
}

// getMediaInfo retrieves information about a media file
func (ws *WebStream) getMediaInfo(path string) (*MediaInfo, error) {
	fileInfo, err := ws.FsAdapter.Stat(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	contentType := ws.getContentType(ext)

	return &MediaInfo{
		Path:        path,
		Size:        fileInfo.Size,
		ModTime:     fileInfo.ModTime,
		ContentType: contentType,
		Extension:   ext,
	}, nil
}

// setMediaHeaders sets common headers for media responses
func (ws *WebStream) setMediaHeaders(w http.ResponseWriter, info *MediaInfo) {
	// Set content type
	w.Header().Set("Content-Type", info.ContentType)

	// Enable range requests
	w.Header().Set("Accept-Ranges", "bytes")

	// Set caching headers
	if ws.EnableCaching {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ws.CacheDuration.Seconds())))
		w.Header().Set("Last-Modified", info.ModTime.UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, info.ModTime.Unix(), info.Size))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	// Prevent browser from trying to execute the file
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// serveFullContent serves the entire media file
func (ws *WebStream) serveFullContent(w http.ResponseWriter, r *http.Request, file io.ReadCloser, info *MediaInfo) {
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.WriteHeader(http.StatusOK)

	// Use efficient copying with buffer
	if r.Method != http.MethodHead {
		buf := make([]byte, ws.BufferSize)
		io.CopyBuffer(w, file, buf)
	}
}

// serveRangeRequest handles HTTP range requests for partial content
func (ws *WebStream) serveRangeRequest(w http.ResponseWriter, r *http.Request, file io.ReadCloser, info *MediaInfo) {
	rangeHeader := r.Header.Get("Range")

	// Parse range header
	ranges, err := parseRange(rangeHeader, info.Size)
	if err != nil || len(ranges) == 0 {
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.Size))
		return
	}

	// For simplicity, only handle single range requests
	// Multi-range requests would require multipart/byteranges
	if len(ranges) > 1 {
		http.Error(w, "Multiple ranges not supported", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := ranges[0]

	// For range requests, we need to reopen the file as a seeker
	// This is a limitation of using io.ReadCloser - we need io.ReadSeeker
	file.Close()

	// Reopen as bytes for seeking (load into memory for range support)
	data, err := ws.FsAdapter.ReadFile(info.Path)
	if err != nil {
		http.Error(w, "Cannot read file for range request", http.StatusInternalServerError)
		return
	}

	// Validate range
	if rangeSpec.Start >= info.Size || rangeSpec.End >= info.Size {
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Calculate content length for this range
	contentLength := rangeSpec.End - rangeSpec.Start + 1

	// Set range response headers
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeSpec.Start, rangeSpec.End, info.Size))
	w.WriteHeader(http.StatusPartialContent)

	// Stream the requested range
	if r.Method != http.MethodHead {
		w.Write(data[rangeSpec.Start : rangeSpec.End+1])
	}
}

// getContentType returns the MIME type for a file extension
func (ws *WebStream) getContentType(ext string) string {
	contentTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".ogv":  "video/ogg",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".m4v":  "video/x-m4v",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
		".oga":  "audio/ogg",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// parseRange parses HTTP Range header
func parseRange(rangeHeader string, fileSize int64) ([]RangeSpec, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("invalid range header")
	}

	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := []RangeSpec{}

	// Split multiple ranges (though we'll only support one)
	for _, part := range strings.Split(rangeStr, ",") {
		part = strings.TrimSpace(part)

		// Parse start-end format
		parts := strings.Split(part, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format")
		}

		var start, end int64
		var err error

		// Handle different range formats
		if parts[0] == "" {
			// Suffix range: "-500" means last 500 bytes
			end = fileSize - 1
			start, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			start = fileSize - start
			if start < 0 {
				start = 0
			}
		} else if parts[1] == "" {
			// Open-ended range: "500-" means from byte 500 to end
			start, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return nil, err
			}
			end = fileSize - 1
		} else {
			// Standard range: "500-999"
			start, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return nil, err
			}
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
		}

		// Validate range
		if start < 0 || end >= fileSize || start > end {
			return nil, fmt.Errorf("invalid range values")
		}

		ranges = append(ranges, RangeSpec{Start: start, End: end})
	}

	return ranges, nil
}
