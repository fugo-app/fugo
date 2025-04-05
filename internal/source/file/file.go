package file

import (
	"fmt"
)

type FileParser interface {
	Parse(line string, data map[string]string) (map[string]string, error)
}

// FileAgent is an implementation of the file-based log agent.
// It watches log files with inotify for changes and processes new log entries.
type FileAgent struct {
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

	parser FileParser
}

func (f *FileAgent) Init() error {
	if f.Path == "" {
		return fmt.Errorf("path is required")
	}

	if f.Format == "" {
		f.Format = "plain"
	}

	if f.Format == "plain" {
		if f.Regex == "" {
			return fmt.Errorf("regex is required for plain format")
		}

		p, err := newPlainParser(f.Regex)
		if err != nil {
			return fmt.Errorf("failed to create plain parser: %w", err)
		} else {
			f.parser = p
		}
	} else if f.Format == "json" {
		p, err := newJsonParser()
		if err != nil {
			return fmt.Errorf("failed to create JSON parser: %w", err)
		} else {
			f.parser = p
		}
	} else {
		return fmt.Errorf("unsupported format: %s", f.Format)
	}

	return nil
}
