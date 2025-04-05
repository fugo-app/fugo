package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/runcitrus/fugo/internal/source"
)

// FileWatcher is an implementation of the file-based log agent.
// It watches log files with inotify for changes and processes new log entries.
type FileWatcher struct {
	// Path to the log file or regex pattern to match multiple files.
	// A named capture group can be used in the fields.
	// For example: `/var/log/nginx/access_(?P<host>.*)\.log`
	Path string `yaml:"path"`

	// Log format to parse the log file: "plain" or "json"
	// Default: "plain"
	Format string `yaml:"format"`

	// Regex to parse the plain log lines
	// Example: `(?P<time>[^ ]+) (?P<level>[^ ]+) (?P<message>.*)`
	Regex string `yaml:"regex,omitempty"`

	dir       string         // Base directory for the path
	re        *regexp.Regexp // Regex to match the file name
	parser    fileParser     // Line parser
	processor source.Processor
	workers   map[string]*fileWorker

	stop chan struct{}
}

func (fw *FileWatcher) Init(processor source.Processor) error {
	if fw.Path == "" {
		return fmt.Errorf("path is required")
	}

	if !strings.HasPrefix(fw.Path, "/") {
		return fmt.Errorf("path must be absolute: %s", fw.Path)
	}

	if fw.Format == "" {
		fw.Format = "plain"
	}

	if fw.Format == "plain" {
		if fw.Regex == "" {
			return fmt.Errorf("regex is required for plain format")
		}

		p, err := newPlainParser(fw.Regex)
		if err != nil {
			return fmt.Errorf("plain parser: %w", err)
		} else {
			fw.parser = p
		}
	} else if fw.Format == "json" {
		p, err := newJsonParser()
		if err != nil {
			return fmt.Errorf("json parser: %w", err)
		} else {
			fw.parser = p
		}
	} else {
		return fmt.Errorf("unsupported format: %s", fw.Format)
	}

	dir, pattern := filepath.Split(fw.Path)
	pattern = "^" + pattern + "$"
	if re, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	} else {
		fw.re = re
	}

	fw.dir = dir
	fw.processor = processor
	fw.workers = make(map[string]*fileWorker)

	return nil
}

// Start begins monitoring log files specified by the path pattern.
// For each matched file, it launches a goroutine that watches for changes.
func (fw *FileWatcher) Start() {
	fw.stop = make(chan struct{})
	go fw.watch()
}

// Stop stops monitoring the log files and closes the watcher.
func (fw *FileWatcher) Stop() {
	if fw.stop != nil {
		close(fw.stop)
	}

	for _, worker := range fw.workers {
		worker.Stop()
	}
}

func (fw *FileWatcher) startWorker(path string, watcher *fsnotify.Watcher) {
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

	worker, err := newFileWorker(path, data, fw.parser, fw.processor)
	if err != nil {
		slog.Error("failed to create worker", "path", path, "error", err)
		return
	}

	fw.workers[name] = worker
	worker.Start()
	watcher.Add(path)
}

func (fw *FileWatcher) stopWorker(path string, watcher *fsnotify.Watcher) {
	name := filepath.Base(path)

	if worker, ok := fw.workers[name]; ok {
		delete(fw.workers, name)
		worker.Stop()
		watcher.Remove(path)
	}
}

func (fw *FileWatcher) watch() {
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
