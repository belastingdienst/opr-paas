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
	timeout = 10 * time.Millisecond
)

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
		path := filepath.Join(tmpDir, filename)
		t.Logf("writing file %s", path)
		if err := os.WriteFile(path, []byte(filename), 0o600); err != nil {
			panic(fmt.Errorf("unable to create temp file: %w", err))
		}
	}

	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after creating 3 files")

	// add file
	path := filepath.Join(tmpDir, "extra")
	t.Logf("writing file %s", path)
	if err := os.WriteFile(path, []byte("extra file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to create extra temp file: %w", err))
	}

	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after adding file")

	// change file
	path = filepath.Join(tmpDir, "extra")
	t.Logf("writing file %s", path)
	if err := os.WriteFile(path, []byte("other extra file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to create extra temp file: %w", err))
	}
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
	t.Logf("writing file %s", filePath)
	if err := os.WriteFile(filePath, []byte("initial file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to create extra temp file: %w", err))
	}
	fw := NewFileWatcher(filePath)
	time.Sleep(timeout)
	require.False(t, fw.WasTriggered(), "fileWatcher was not triggered after init")

	t.Logf("writing file %s", filePath)
	if err := os.WriteFile(filePath, []byte("other file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to write to extra temp file: %w", err))
	}
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after writing to file")

	os.Remove(filePath)
	if err := os.WriteFile(filePath, []byte("recreated file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to write to extra temp file: %w", err))
	}
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after removing file")

	if err := os.WriteFile(filePath, []byte("recreated file data again"), 0o600); err != nil {
		panic(fmt.Errorf("unable to write to extra temp file: %w", err))
	}
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
	t.Logf("writing file %s", filePath)
	if err := os.WriteFile(filePath, []byte("initial file data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to create extra temp file: %w", err))
	}
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(filePath, symlinkPath); err != nil {
		panic(fmt.Errorf("unable to create symlink: %w", err))
	}

	fw := NewFileWatcher(symlinkPath)
	time.Sleep(timeout)
	require.False(t, fw.WasTriggered(), "fileWatcher was not triggered after init")

	t.Logf("writing to symlink %s", symlinkPath)
	if err := os.WriteFile(symlinkPath, []byte("other symlink data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to write to symlink: %w", err))
	}
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

	if err := os.WriteFile(filePath, []byte("recreated symlink data"), 0o600); err != nil {
		panic(fmt.Errorf("unable to write to extra temp file: %w", err))
	}
	time.Sleep(timeout)
	require.True(t, fw.WasTriggered(), "fileWatcher was triggered after recreating symlink")
}
