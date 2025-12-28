package debug_fs

import (
	"fmt"
	"io/fs"
)

// PrintEmbeddedFiles walks through a filesystem and prints all files
func PrintEmbeddedFiles(fsys fs.FS, message string) {
	if message != "" {
		fmt.Println(message)
	}
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Printf("  - %s\n", path)
		}
		return nil
	})
}
