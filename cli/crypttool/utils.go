package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func pathToFileList(paths []string) ([]string, error) {
	files := make(map[string]bool)
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error while walking the path: %e", err)
			} else if info.Mode().IsRegular() {
				files[path] = true
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
