/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathToFileList can be fed multiple paths, which it will walk and return a fill list of all files in the path /
// subdirectories
func PathToFileList(paths []string) ([]string, error) {
	files := make(map[string]bool)
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("error while walking the path: %w", walkErr)
			}

			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("failed to resolve symlink %s: %w", path, err)
			}

			absPath, err := filepath.Abs(resolvedPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", resolvedPath, err)
			}

			absMode, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("failed to get filemode for %s: %w", absPath, err)
			}

			if absMode.Mode().IsRegular() {
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
