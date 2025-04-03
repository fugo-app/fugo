package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// FileAgent is an implementation of the file-based log agent.
// It watches log files with inotify for changes and processes new log entries.
type FileAgent struct {
	// Path to the log file or regex pattern to match multiple files.
	// A named capture group can be used in the regex.
	// For example: `/var/log/nginx/access_(?P<host>.*)\.log`
	Path string `yaml:"path"`

	// Log format to parse the log file: "plain" or "json"
	// Default: "plain"
	Format string `yaml:"format"`

	// Regex to parse the plain log lines
	// Example: `(?P<time>[^ ]+) (?P<level>[^ ]+) (?P<message>.*)`
	Regex string `yaml:"regex,omitempty"`

	// Time format to parse the timestamp field in log lines
	Timestamp *TimestampFormat `yaml:"timestamp,omitempty"`

	// Fields to include in the final log record.
	// Variants:
	// - empty value: include fields as is
	// - template: convert log fields into new field using Go template
	Fields map[string]string `yaml:"templates,omitempty"`

	templates map[string]*template.Template
	fields    map[string]struct{}

	regexPattern *regexp.Regexp
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

		if pattern, err := regexp.Compile(f.Regex); err != nil {
			return fmt.Errorf("failed to compile regex pattern: %w", err)
		} else {
			f.regexPattern = pattern
		}
	}

	if len(f.Fields) == 0 {
		return fmt.Errorf("fields are required")
	}

	f.fields = make(map[string]struct{})
	f.templates = make(map[string]*template.Template)

	for key, value := range f.Fields {
		if value == "" {
			f.fields[key] = struct{}{}
		} else {
			if tpl, err := template.New(key).Parse(value); err != nil {
				return fmt.Errorf("failed to parse template %s: %w", key, err)
			} else {
				f.fields[key] = struct{}{}
				f.templates[key] = tpl
			}
		}
	}

	if f.Timestamp != nil {
		if err := f.Timestamp.Init(); err != nil {
			return fmt.Errorf("failed to initialize timestamp format: %w", err)
		}

		f.fields["time"] = struct{}{}
	}

	return nil
}

// Parse processes a log line based on the configured format (JSON or plain)
// and returns a map of field names to values extracted from the log line.
func (f *FileAgent) Parse(logLine string) (map[string]string, error) {
	if logLine == "" {
		return nil, nil
	}

	var result map[string]string
	var err error

	switch f.Format {
	case "json":
		result, err = f.parseJSON(logLine)
	default: // "plain" or any other format defaults to plain
		result, err = f.parsePlain(logLine)
	}

	if result == nil || err != nil {
		return nil, err
	}

	if f.Timestamp != nil {
		if err := f.Timestamp.Convert(result); err != nil {
			return nil, fmt.Errorf("failed to convert timestamp: %w", err)
		}
	}

	f.renderTemplates(result)

	for key := range result {
		if _, ok := f.fields[key]; !ok {
			delete(result, key) // Remove fields not in the configured fields
		}
	}

	return result, nil
}

// parseJSON extracts fields from a JSON-formatted log line.
// If JSON fields are specified, only those fields are extracted.
func (f *FileAgent) parseJSON(line string) (map[string]string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON log line: %w", err)
	}

	result := make(map[string]string)

	for key, val := range data {
		result[key] = fmt.Sprintf("%v", val)
	}

	return result, nil
}

func (f *FileAgent) renderTemplates(record map[string]string) {
	for key, tpl := range f.templates {
		var str strings.Builder
		if err := tpl.Execute(&str, record); err != nil {
			continue // Ignore errors in template rendering
		}
		record[key] = str.String()
	}
}

// parsePlain extracts fields from a plain-text log line using the configured regex pattern.
func (f *FileAgent) parsePlain(line string) (map[string]string, error) {
	match := f.regexPattern.FindStringSubmatch(line)
	if match == nil {
		return nil, nil
	}

	// Extract named capture groups
	result := make(map[string]string)
	for i, name := range f.regexPattern.SubexpNames() {
		if i == 0 {
			continue // Skip the full match
		}

		if name == "" {
			continue // Skip unnamed groups
		}

		result[name] = match[i]
	}

	// If no named groups were matched, return the log line as a message
	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}
