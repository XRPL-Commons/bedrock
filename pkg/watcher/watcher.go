package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors file changes and triggers callbacks
type Watcher struct {
	dirs       []string
	extensions []string
	debounce   time.Duration
}

// New creates a new file Watcher
func New(dirs []string, extensions []string) *Watcher {
	return &Watcher{
		dirs:       dirs,
		extensions: extensions,
		debounce:   500 * time.Millisecond,
	}
}

// Watch starts watching for file changes and calls onChange when detected.
// It blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context, onChange func()) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer fsw.Close()

	// Add directories recursively
	for _, dir := range w.dirs {
		if err := w.addRecursive(fsw, dir); err != nil {
			return fmt.Errorf("failed to watch %s: %w", dir, err)
		}
	}

	// Debounce timer
	var timer *time.Timer

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}

			// Only react to writes and creates
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			// Check file extension
			if !w.matchesExtension(event.Name) {
				continue
			}

			// If a new directory was created, watch it too
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addRecursive(fsw, event.Name)
				}
			}

			// Debounce: reset timer on each change
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, onChange)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

func (w *Watcher) addRecursive(fsw *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}
		if info.IsDir() {
			// Skip hidden directories and target/build dirs
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "target" || base == "node_modules" {
				return filepath.SkipDir
			}
			return fsw.Add(path)
		}
		return nil
	})
}

func (w *Watcher) matchesExtension(path string) bool {
	if len(w.extensions) == 0 {
		return true
	}
	ext := filepath.Ext(path)
	for _, e := range w.extensions {
		if ext == e {
			return true
		}
	}
	return false
}
