package comm

import (
	"io"
	"io/fs"
	"time"
)

// FileInfo represents file metadata
type FileInfo struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    fs.FileMode `json:"mode"`
	ModTime time.Time   `json:"mod_time"`
	IsDir   bool        `json:"is_dir"`
}

// IFsProvider defines the interface for filesystem operations
// This interface provides flexibility to support different filesystem backends:
// - OS filesystem (osfs)
// - Embedded filesystem (embedfs)
// - Web-based filesystem (webfs)
// - Custom implementations
type IFsProvider interface {
	// ReadFile reads the contents of a file and returns it as a byte slice
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file at the specified path
	// For read-only filesystems (like embed.FS), this should return an error
	WriteFile(path string, data []byte, perm fs.FileMode) error

	// Open opens a file for reading and returns an io.ReadCloser
	Open(path string) (io.ReadCloser, error)

	// Exists checks if a file or directory exists at the given path
	Exists(path string) bool

	// Stat returns file information for the given path
	Stat(path string) (FileInfo, error)

	// ListDir returns a list of files and directories in the specified directory
	ListDir(path string) ([]FileInfo, error)

	// IsDir checks if the path is a directory
	IsDir(path string) bool

	// GetMimeType returns the MIME type for a file based on its extension
	GetMimeType(path string) string

	// GetBasePath returns the base path for this filesystem provider
	GetBasePath() string

	// SetBasePath sets the base path for this filesystem provider
	SetBasePath(basePath string)

	// IsReadOnly returns true if the filesystem is read-only
	IsReadOnly() bool

	// Close cleans up any resources used by the provider
	Close() error
}
