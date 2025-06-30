/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathDoesNotExist(t *testing.T) {
	paths := []string{"path1", "path2"}
	// expectedFiles := []string{"file1", "file2"}

	_, err := PathToFileList(paths)
	assert.NotNil(t, err)
	// assert.Equal(t, expectedFiles, files)
	assert.ErrorContains(t, err, "error while walking the path: lstat path1: no such file or directory")
}

func TestPathHappyFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Create some directories and files in the temporary directory
	for i := 0; i < 3; i++ {
		err = os.Mkdir(filepath.Join(tempDir, fmt.Sprintf("path%d", i)), 0o755) // revive:disable-line:add-constant
		if err != nil {
			t.Fatalf("Error creating directory: %v", err)
		}
		err = os.WriteFile(filepath.Join(tempDir,
			fmt.Sprintf("path%d/file%d", i, i)),
			[]byte(fmt.Sprintf("content%d", i)), 0o644) // revive:disable-line:add-constant
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
	}

	paths := []string{filepath.Join(tempDir, "path1"), filepath.Join(tempDir, "path2")}
	expectedFiles := []string{filepath.Join(tempDir, "path1", "file1"), filepath.Join(tempDir, "path2", "file2")}

	files, err := PathToFileList(paths)

	// Check if files are prefixed with "/private", meaning we're on macOS
	for _, file := range files {
		if strings.HasPrefix(file, "/private") {
			// Update expectedFiles to include the correct path prefix for macOS
			expectedFiles = []string{
				filepath.Join("/private", tempDir, "path1", "file1"),
				filepath.Join("/private", tempDir, "path2", "file2"),
			}
		}
	}

	assert.Nil(t, err)
	slices.Sort(expectedFiles)
	slices.Sort(files)
	assert.Equal(t, expectedFiles, files)
}
