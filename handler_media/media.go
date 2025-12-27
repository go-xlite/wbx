package handlermedia

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// MediaHandler handles video and audio streaming with range request support
type MediaHandler struct {
	*handler_role.HandlerRole
	MediaDir          string
	BufferSize        int
	EnableCaching     bool
	CacheDuration     time.Duration
	AllowedExtensions map[string]bool
}

// MediaInfo contains metadata about a media file
type MediaInfo struct {
	Path        string
	Size        int64
	ModTime     time.Time
	ContentType string
	Extension   string
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(handler handler_role.IHandler, mediaDir string) *MediaHandler {
	return &MediaHandler{
		HandlerRole:   &handler_role.HandlerRole{Handler: handler, PathPrefix: "/media"},
		MediaDir:      mediaDir,
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
}

// SetBufferSize sets the streaming buffer size
func (mh *MediaHandler) SetBufferSize(size int) *MediaHandler {
	mh.BufferSize = size
	return mh
}

// SetCaching enables or disables caching
func (mh *MediaHandler) SetCaching(enabled bool, duration time.Duration) *MediaHandler {
	mh.EnableCaching = enabled
	mh.CacheDuration = duration
	return mh
}

// AddAllowedExtension adds an allowed file extension
func (mh *MediaHandler) AddAllowedExtension(ext string) *MediaHandler {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	mh.AllowedExtensions[strings.ToLower(ext)] = true
	return mh
}

// ServeMedia serves a media file with range request support
func (mh *MediaHandler) ServeMedia(w http.ResponseWriter, r *http.Request, filePath string) {
	// Get absolute path
	absPath := filepath.Join(mh.MediaDir, filepath.Clean(filePath))

	// Security check - ensure path is within media directory
	if !strings.HasPrefix(absPath, filepath.Clean(mh.MediaDir)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists and get info
	info, err := mh.getMediaInfo(absPath)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Check if extension is allowed
	if !mh.AllowedExtensions[strings.ToLower(info.Extension)] {
		http.Error(w, "Media type not allowed", http.StatusForbidden)
		return
	}

	// Open the file
	file, err := os.Open(info.Path)
	if err != nil {
		http.Error(w, "Cannot open media file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set common headers
	mh.setMediaHeaders(w, info)

	// Handle range requests
	if r.Header.Get("Range") != "" {
		mh.serveRangeRequest(w, r, file, info)
	} else {
		mh.serveFullContent(w, r, file, info)
	}
}

// getMediaInfo retrieves information about a media file
func (mh *MediaHandler) getMediaInfo(path string) (*MediaInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}

	ext := strings.ToLower(filepath.Ext(path))
	contentType := mh.getContentType(ext)

	return &MediaInfo{
		Path:        path,
		Size:        stat.Size(),
		ModTime:     stat.ModTime(),
		ContentType: contentType,
		Extension:   ext,
	}, nil
}

// setMediaHeaders sets common headers for media responses
func (mh *MediaHandler) setMediaHeaders(w http.ResponseWriter, info *MediaInfo) {
	// Set content type
	w.Header().Set("Content-Type", info.ContentType)

	// Enable range requests
	w.Header().Set("Accept-Ranges", "bytes")

	// Set caching headers
	if mh.EnableCaching {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(mh.CacheDuration.Seconds())))
		w.Header().Set("Last-Modified", info.ModTime.UTC().Format(http.TimeFormat))
		w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, info.ModTime.Unix(), info.Size))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	// Prevent browser from trying to execute the file
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// serveFullContent serves the entire media file
func (mh *MediaHandler) serveFullContent(w http.ResponseWriter, r *http.Request, file *os.File, info *MediaInfo) {
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.WriteHeader(http.StatusOK)

	// Use efficient copying with buffer
	if r.Method != http.MethodHead {
		buf := make([]byte, mh.BufferSize)
		io.CopyBuffer(w, file, buf)
	}
}

// serveRangeRequest handles HTTP range requests for partial content
func (mh *MediaHandler) serveRangeRequest(w http.ResponseWriter, r *http.Request, file *os.File, info *MediaInfo) {
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

	// Seek to start position
	_, err = file.Seek(rangeSpec.Start, io.SeekStart)
	if err != nil {
		http.Error(w, "Seek error", http.StatusInternalServerError)
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
		buf := make([]byte, mh.BufferSize)
		limitedReader := io.LimitReader(file, contentLength)
		io.CopyBuffer(w, limitedReader, buf)
	}
}

// HandleMedia creates an HTTP handler for serving media
func (mh *MediaHandler) HandleMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract file path from URL
		filePath := strings.TrimPrefix(r.URL.Path, mh.PathPrefix)
		filePath = strings.TrimPrefix(filePath, "/")

		if filePath == "" {
			http.Error(w, "No media file specified", http.StatusBadRequest)
			return
		}

		mh.ServeMedia(w, r, filePath)
	}
}

// RegisterRoutes registers media serving routes
func (mh *MediaHandler) RegisterRoutes(pathPrefix string) {
	mh.PathPrefix = pathPrefix
	mh.Handler.GetRoutes().HandlePathPrefixFn(pathPrefix, mh.HandleMedia())
}

// getContentType returns the MIME type for a file extension
func (mh *MediaHandler) getContentType(ext string) string {
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

// RangeSpec represents a byte range
type RangeSpec struct {
	Start int64
	End   int64
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

// StreamInfo provides streaming statistics
type StreamInfo struct {
	FilePath       string
	FileSize       int64
	BytesSent      int64
	StartTime      time.Time
	IsRangeRequest bool
	RangeStart     int64
	RangeEnd       int64
}

// MediaConfig provides configuration for media handler
type MediaConfig struct {
	MediaDir          string
	BufferSize        int
	EnableCaching     bool
	CacheDuration     time.Duration
	AllowedExtensions []string
}

// NewMediaFromConfig creates a media handler from configuration
func NewMediaFromConfig(handler handler_role.IHandler, config MediaConfig) *MediaHandler {
	mh := NewMediaHandler(handler, config.MediaDir)

	if config.BufferSize > 0 {
		mh.SetBufferSize(config.BufferSize)
	}

	mh.SetCaching(config.EnableCaching, config.CacheDuration)

	if len(config.AllowedExtensions) > 0 {
		mh.AllowedExtensions = make(map[string]bool)
		for _, ext := range config.AllowedExtensions {
			mh.AddAllowedExtension(ext)
		}
	}

	return mh
}
