package embedfs

import (
	"embed"
	"io"
	"io/fs"
	"path"
	"strings"

	webFs "github.com/go-xlite/wbx/adapter_fs/web_fs"
	"github.com/go-xlite/wbx/comm"
)

// EmbedFS provides filesystem operations using Go's embedded filesystem
type EmbedFS struct {
	*webFs.WebFs
	fs        *embed.FS
	EmbedPath string
}

// NewEmbedFS creates a new embedded filesystem provider
func NewEmbedFS(embedFS *embed.FS) *EmbedFS {
	return &EmbedFS{
		WebFs:     webFs.NewWebFsReadOnly(), // Embedded FS is always read-only
		fs:        embedFS,
		EmbedPath: "",
	}
}

// NewEmbedFSWithBasePath creates a new embedded filesystem provider with a base path
func NewEmbedFSWithBasePath(embedFS *embed.FS, basePath string) *EmbedFS {
	efs := &EmbedFS{
		WebFs:     webFs.NewWebFsReadOnly(),
		fs:        embedFS,
		EmbedPath: basePath,
	}
	efs.SetBasePath(basePath)
	return efs
}

// ReadFile reads a file from the embedded filesystem
func (e *EmbedFS) ReadFile(path string) ([]byte, error) {
	if e.fs == nil {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrInvalid}
	}

	fullPath := e.makePath(path)
	return e.fs.ReadFile(fullPath)
}

// WriteFile always returns an error since embedded FS is read-only
func (e *EmbedFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return &fs.PathError{Op: "write", Path: path, Err: fs.ErrPermission}
}

// Open opens a file for reading from the embedded filesystem
func (e *EmbedFS) Open(path string) (io.ReadCloser, error) {
	if e.fs == nil {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrInvalid}
	}

	fullPath := e.makePath(path)
	return e.fs.Open(fullPath)
}

// Exists checks if a file or directory exists in the embedded filesystem
func (e *EmbedFS) Exists(path string) bool {
	if e.fs == nil {
		return false
	}

	fullPath := e.makePath(path)
	_, err := fs.Stat(e.fs, fullPath)
	return err == nil
}

// Stat returns file information from the embedded filesystem
func (e *EmbedFS) Stat(path string) (comm.FileInfo, error) {
	if e.fs == nil {
		return comm.FileInfo{}, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrInvalid}
	}

	fullPath := e.makePath(path)
	info, err := fs.Stat(e.fs, fullPath)
	if err != nil {
		return comm.FileInfo{}, err
	}
	return webFs.ConvertFileInfo(info), nil
}

// ListDir returns a list of files and directories from the embedded filesystem
func (e *EmbedFS) ListDir(path string) ([]comm.FileInfo, error) {
	if e.fs == nil {
		return nil, &fs.PathError{Op: "readdir", Path: path, Err: fs.ErrInvalid}
	}

	fullPath := e.makePath(path)
	entries, err := fs.ReadDir(e.fs, fullPath)
	if err != nil {
		return nil, err
	}

	var result []comm.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}
		result = append(result, webFs.ConvertFileInfo(info))
	}

	return result, nil
}

// IsDir checks if the path is a directory in the embedded filesystem
func (e *EmbedFS) IsDir(path string) bool {
	if e.fs == nil {
		return false
	}

	fullPath := e.makePath(path)
	info, err := fs.Stat(e.fs, fullPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// makePath constructs the full path by combining EmbedPath and relative path
// Uses forward slashes as required by embed.FS
func (e *EmbedFS) makePath(filePath string) string {
	// Use EmbedPath if set, otherwise use BasePath from WebFs
	basePath := e.EmbedPath
	if basePath == "" {
		basePath = e.GetBasePath()
	}

	if basePath == "" {
		return strings.TrimPrefix(filePath, "/")
	}

	// Ensure forward slashes for embed.FS
	basePath = strings.ReplaceAll(basePath, "\\", "/")
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// Remove leading slashes
	basePath = strings.TrimPrefix(basePath, "/")
	filePath = strings.TrimPrefix(filePath, "/")

	if filePath == "" {
		return basePath
	}

	if basePath == "" {
		return filePath
	}

	return path.Join(basePath, filePath)
}

// GetEmbedFS returns the underlying embed.FS (for compatibility)
func (e *EmbedFS) GetEmbedFS() *embed.FS {
	return e.fs
}

// SetEmbedFS sets the underlying embed.FS
func (e *EmbedFS) SetEmbedFS(embedFS *embed.FS) {
	e.fs = embedFS
}
