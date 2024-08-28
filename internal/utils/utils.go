package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func PathToFileList(paths []string) ([]string, error) {
	files := make(map[string]bool)
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error while walking the path: %w", err)
			} else if resolvedPath, err := filepath.EvalSymlinks(path); err != nil {
				return fmt.Errorf("failed to resolve symlink %s: %w", path, err)
			} else if absPath, err := filepath.Abs(resolvedPath); err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", resolvedPath, err)
			} else if absMode, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("failed to get filemode for %s: %w", absPath, err)
			} else if absMode.Mode().IsRegular() {
				files[absPath] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	var fileList []string
	for key := range files {
		fileList = append(fileList, key)
	}
	return fileList, nil
}
