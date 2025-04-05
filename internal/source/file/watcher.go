package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type fileWatcher struct {
	dir     string
	re      *regexp.Regexp
	parser  fileParser
	workers map[string]*fileWorker

	stop chan struct{}
}

func newFileWatcher(path string, parser fileParser) (*fileWatcher, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	dir, pattern := filepath.Split(path)

	pattern = "^" + pattern + "$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	return &fileWatcher{
		dir:     dir,
		re:      re,
		parser:  parser,
		workers: make(map[string]*fileWorker),
		stop:    make(chan struct{}),
	}, nil
}

func (fw *fileWatcher) startWorker(path string, watcher *fsnotify.Watcher) {
	name := filepath.Base(path)
	match := fw.re.FindStringSubmatch(name)
	if match == nil {
		return
	}

	data := make(map[string]string)
	for i, name := range fw.re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		data[name] = match[i]
	}

	worker, err := newFileWorker(path, data)
	if err != nil {
		slog.Error("failed to create worker", "path", path, "error", err)
		return
	}

	fw.workers[name] = worker
	worker.Start(fw.parser)
	watcher.Add(path)
}

func (fw *fileWatcher) stopWorker(path string, watcher *fsnotify.Watcher) {
	name := filepath.Base(path)

	if worker, ok := fw.workers[name]; ok {
		delete(fw.workers, name)
		worker.Stop()
		watcher.Remove(path)
	}
}

func (fw *fileWatcher) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("failed to start watcher", "dir", fw.dir, "error", err)
		return
	}
	defer watcher.Close()

	entries, err := os.ReadDir(fw.dir)
	if err != nil {
		slog.Error("failed to read directory", "dir", fw.dir, "error", err)
		return
	}

	watcher.Add(fw.dir)

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(fw.dir, name)
		fw.startWorker(path, watcher)
	}

	for {
		select {
		case <-fw.stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				continue
			}

			if event.Has(fsnotify.Write) {
				name := filepath.Base(event.Name)
				if worker, ok := fw.workers[name]; ok {
					worker.Handle()
				}
			} else if event.Has(fsnotify.Create) {
				stat, err := os.Stat(event.Name)
				if err != nil || !stat.Mode().IsRegular() {
					continue
				}

				fw.startWorker(event.Name, watcher)
			} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				fw.stopWorker(event.Name, watcher)
			}
		}
	}
}

// Start begins monitoring log files specified by the path pattern.
// For each matched file, it launches a goroutine that watches for changes.
func (fw *fileWatcher) Start() {
	go fw.watch()
}

// Stop stops monitoring the log files and closes the watcher.
func (fw *fileWatcher) Stop() {
	if fw.stop != nil {
		close(fw.stop)
	}

	for _, worker := range fw.workers {
		worker.Stop()
	}
}
