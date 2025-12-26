package osfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-xlite/wbx/comm"
	webFs "github.com/go-xlite/wbx/fs_provider/web_fs"
)

// OsFs provides filesystem operations using the OS filesystem
type OsFs struct {
	*webFs.WebFs
}

// NewOsFs creates a new OS filesystem provider
func NewOsFs() *OsFs {
	return &OsFs{
		WebFs: webFs.NewWebFs(),
	}
}

// NewOsFsWithBasePath creates a new OS filesystem provider with a base path
func NewOsFsWithBasePath(basePath string) *OsFs {
	osFs := &OsFs{
		WebFs: webFs.NewWebFs(),
	}
	osFs.SetBasePath(basePath)
	return osFs
}

// ReadFile reads a file from the OS filesystem
func (o *OsFs) ReadFile(path string) ([]byte, error) {
	fullPath := o.makePath(path)
	return os.ReadFile(fullPath)
}

// WriteFile writes data to a file in the OS filesystem
func (o *OsFs) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if o.IsReadOnly() {
		return &fs.PathError{Op: "write", Path: path, Err: fs.ErrPermission}
	}

	fullPath := o.makePath(path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, perm)
}

// Open opens a file for reading
func (o *OsFs) Open(path string) (io.ReadCloser, error) {
	fullPath := o.makePath(path)
	return os.Open(fullPath)
}

// Exists checks if a file or directory exists
func (o *OsFs) Exists(path string) bool {
	fullPath := o.makePath(path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Stat returns file information
func (o *OsFs) Stat(path string) (comm.FileInfo, error) {
	fullPath := o.makePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return comm.FileInfo{}, err
	}
	return webFs.ConvertFileInfo(info), nil
}

// ListDir returns a list of files and directories
func (o *OsFs) ListDir(path string) ([]comm.FileInfo, error) {
	fullPath := o.makePath(path)
	entries, err := os.ReadDir(fullPath)
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

// IsDir checks if the path is a directory
func (o *OsFs) IsDir(path string) bool {
	fullPath := o.makePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// makePath constructs the full path by combining base path and relative path
func (o *OsFs) makePath(path string) string {
	basePath := o.GetBasePath()
	if basePath == "" {
		return path
	}

	// Clean and join paths
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return basePath
	}

	return filepath.Join(basePath, path)
}
