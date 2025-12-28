package webFs

import (
	"io/fs"
	"strings"
	"time"

	"github.com/go-xlite/wbx/comm"
)

// WebFs provides base filesystem functionality that can be embedded by other providers
type WebFs struct {
	basePath    string
	readOnly    bool
	customMimes map[string]string
}

// NewWebFs creates a new WebFs instance
func NewWebFs() *WebFs {
	return &WebFs{
		basePath:    "",
		readOnly:    false,
		customMimes: make(map[string]string),
	}
}

// NewWebFsReadOnly creates a read-only WebFs instance
func NewWebFsReadOnly() *WebFs {
	return &WebFs{
		basePath:    "",
		readOnly:    true,
		customMimes: make(map[string]string),
	}
}

// GetBasePath returns the base path
func (w *WebFs) GetBasePath() string {
	return w.basePath
}

// SetBasePath sets the base path
func (w *WebFs) SetBasePath(basePath string) {
	w.basePath = strings.TrimSuffix(basePath, "/")
}

// IsReadOnly returns whether the filesystem is read-only
func (w *WebFs) IsReadOnly() bool {
	return w.readOnly
}

// SetReadOnly sets the read-only flag
func (w *WebFs) SetReadOnly(readOnly bool) {
	w.readOnly = readOnly
}

// AddCustomMimeType adds a custom MIME type mapping
func (w *WebFs) AddCustomMimeType(extension, mimeType string) {
	ext := strings.ToLower(extension)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	w.customMimes[ext] = mimeType
}

// GetMimeType returns the MIME type for a file path
func (w *WebFs) GetMimeType(path string) string {
	// Extract file extension
	dotIndex := strings.LastIndex(path, ".")
	if dotIndex == -1 {
		return "application/octet-stream"
	}

	ext := strings.ToLower(path[dotIndex:])

	// Check custom MIME types first
	if mimeType, exists := w.customMimes[ext]; exists {
		return mimeType
	}

	// Fall back to common MIME types
	return comm.Mime.GetType(ext)
}

// makePath constructs the full path by combining base path and relative path
func (w *WebFs) makePath(path string) string {
	if w.basePath == "" {
		return path
	}

	// Remove leading slash from path if present
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		return w.basePath
	}

	return w.basePath + "/" + path
}

// Base implementations that can be overridden by specific providers

// WriteFile - base implementation returns error for read-only filesystems
func (w *WebFs) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if w.readOnly {
		return &fs.PathError{Op: "write", Path: path, Err: fs.ErrPermission}
	}
	return &fs.PathError{Op: "write", Path: path, Err: fs.ErrInvalid}
}

// Close - base implementation does nothing
func (w *WebFs) Close() error {
	return nil
}

// Helper function to convert fs.FileInfo to comm.FileInfo
func ConvertFileInfo(info fs.FileInfo) comm.FileInfo {
	return comm.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
}

// Helper function to create a FileInfo from basic parameters
func NewFileInfo(name string, size int64, mode fs.FileMode, modTime time.Time, isDir bool) comm.FileInfo {
	return comm.FileInfo{
		Name:    name,
		Size:    size,
		Mode:    mode,
		ModTime: modTime,
		IsDir:   isDir,
	}
}
