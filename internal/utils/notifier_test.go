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
	timeout = 1 * time.Millisecond
)

func Test_FileChanged(t *testing.T) {
	// Create folder
	tmpDir, err := os.MkdirTemp("", "sampledir")
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
