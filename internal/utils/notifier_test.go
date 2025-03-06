package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	timeout               = 10 * time.Millisecond
	fileModeUserReadWrite = 0o600
)

func writeFile(t *testing.T, path string, data string) {
	t.Logf("writing file %s", path)
	if err := os.WriteFile(path, []byte(data), fileModeUserReadWrite); err != nil {
		panic(fmt.Errorf("unable to write to file: %w", err))
	}
}

func Test_FolderChanged(t *testing.T) {
	// Create folder
	tmpDir, err := os.MkdirTemp("", "notifierFolderTest")
	if err != nil {
		panic(fmt.Errorf("unable to create temp dir: %w", err))
	}
	fw := NewFileWatcher(tmpDir)
	time.Sleep(timeout)
	require.False(t, fw.WasTriggered(), "fileWatcher was not triggered after init")

	// create 3 files
	for _, filename := range []string{"f1", "f2", "f3"} {
		writeFile(t, filepath.Join(tmpDir, filename), filename)
	}

	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after creating 3 files")

	// add file
	writeFile(t, filepath.Join(tmpDir, "extra"), "extra file data")

	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after adding file")

	// change file
	writeFile(t, filepath.Join(tmpDir, "extra"), "other extra file data")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after adding file")
}

func Test_FileChanged(t *testing.T) {
	// Create folder
	tmpDir, err := os.MkdirTemp("", "notifierFileTest")
	if err != nil {
		panic(fmt.Errorf("unable to create temp dir: %w", err))
	}

	// add file
	filePath := filepath.Join(tmpDir, "extra")
	writeFile(t, filePath, "initial file data")
	fw := NewFileWatcher(filePath)
	time.Sleep(timeout)
	require.False(t, fw.WasTriggered(), "fileWatcher was not triggered after init")

	writeFile(t, filePath, "other file data")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after writing to file")

	os.Remove(filePath)
	writeFile(t, filePath, "recreated file data")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after removing file")

	writeFile(t, filePath, "recreated file data again")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after recreating file")
}

func Test_LinkChanged(t *testing.T) {
	// Create folder
	tmpDir, err := os.MkdirTemp("", "notifierSymlinkTest")
	if err != nil {
		panic(fmt.Errorf("unable to create temp dir: %w", err))
	}

	// add file
	filePath := filepath.Join(tmpDir, "extra")
	writeFile(t, filePath, "initial file data")
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(filePath, symlinkPath); err != nil {
		panic(fmt.Errorf("unable to create symlink: %w", err))
	}

	fw := NewFileWatcher(symlinkPath)
	time.Sleep(timeout)
	require.False(t, fw.WasTriggered(), "fileWatcher was not triggered after init")

	writeFile(t, filePath, "other symlink data")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after writing to symlink")

	os.Remove(symlinkPath)
	time.Sleep(timeout)
	if err := os.Symlink(filePath, symlinkPath); err != nil {
		panic(fmt.Errorf("unable to create symlink: %w", err))
	}
	time.Sleep(timeout)
	// !!! known behavior. fsnotifier does not track symlinks themselves, but files they point to
	require.False(t, fw.WasTriggered(), "fileWatcher is not triggered after removing symlink")

	writeFile(t, filePath, "recreated symlink data")
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after recreating symlink")
}
