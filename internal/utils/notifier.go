package utils

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher   *fsnotify.Watcher
	files     []string
	count     int
	lastCount int
}

// watch keeps track of changes on paths and runs callbackFunc when they change
func NewFileWatcher(paths ...string) *FileWatcher {
	fw := &FileWatcher{
		files: paths,
		count: 0,
	}
	go func() {
		_ = fw.watch()
	}()
	return fw
}

func (fw *FileWatcher) WasTriggered() bool {
	if fw.lastCount != fw.count {
		fw.lastCount = fw.count
		// kubernetes removes and creates a file when a mounted secret or configmap is changed
		// refresh will re-add the newly created files after they have been changed
		_ = fw.Refresh()
		return true
	}
	return false
}

func (fw *FileWatcher) Refresh() (err error) {
	// Notes from fsnotify.Watcher.Add():
	// - A path can only be watched once; watching it more than once is a no-op and will not return an error.
	// - Paths that do not yet exist on the filesystem cannot be watched.
	// - A watch will be automatically removed if the watched path is deleted or renamed. T
	for _, p := range fw.files {
		err = fw.watcher.Add(p)
		if err != nil {
			return fmt.Errorf("%q: %w", p, err)
		}
	}
	return nil
}

func (fw *FileWatcher) watch() (err error) {
	fw.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("issue %w while creating a watcher for these files: %v", err, fw.files)
	}
	defer fw.watcher.Close()

	go fw.watchLoop()
	if err = fw.Refresh(); err != nil {
		return err
	}

	<-make(chan struct{}) // Block forever
	return nil
}

// watchLoop is the inner function which loops until a change is noticed and then runs callBack func
func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case err, ok := <-fw.watcher.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			log.Printf("ERROR: %s", err)
		case e, ok := <-fw.watcher.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			// Just print the event nicely aligned, and keep track how many
			// events we've seen.
			fw.count++
			log.Printf("secret notification: %3d/%3d %s", fw.count, fw.lastCount, e)
		}
	}
}
