package utils

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
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
		return true
	}
	return false
}

func (fw *FileWatcher) watch() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("issue %w while creating a watcher for these files: %v", err, fw.files)
	}
	defer w.Close()

	go fw.watchLoop(w)

	for _, p := range fw.files {
		err = w.Add(p)
		if err != nil {
			return fmt.Errorf("%q: %w", p, err)
		}
	}

	<-make(chan struct{}) // Block forever
	return nil
}

// watchLoop is the inner function which loops until a change is noticed and then runs callBack func
func (fw *FileWatcher) watchLoop(w *fsnotify.Watcher) {
	for {
		select {
		case err, ok := <-w.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			log.Printf("ERROR: %s", err)
		case e, ok := <-w.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			// Just print the event nicely aligned, and keep track how many
			// events we've seen.
			fw.count += 1
			log.Printf("secret notification: %3d/%3d %s", fw.count, fw.lastCount, e)
		}
	}
}
