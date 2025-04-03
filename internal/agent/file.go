package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// Field represents a field in the log record.
type Field struct {
	// Name of the field in the log record.
	Name string `yaml:"name"`
	// Template to convert source fields into new record field.
	Template string `yaml:"template,omitempty"`
	// Source field name to extract the value from.
	Source string `yaml:"source,omitempty"`
	// Time format only for the "time" field.
	TimeFormat string `yaml:"time_format,omitempty"`

	source   string
	template *template.Template
}

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

	// Fields to include in the final log record.
	Fields []Field `yaml:"fields"`

	timestamp    *TimestampFormat `yaml:"timestamp,omitempty"`
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

	for i := range f.Fields {
		field := &f.Fields[i]
		field.source = field.Source
		if field.source == "" {
			field.source = field.Name
		}

		if field.Name == "time" {
			f.timestamp = &TimestampFormat{
				Format: field.TimeFormat,
			}
			if err := f.timestamp.Init(); err != nil {
				return fmt.Errorf("failed to initialize timestamp format: %w", err)
			}
			continue
		}

		if field.Template != "" {
			if tpl, err := template.New(field.Name).Parse(field.Template); err != nil {
				return fmt.Errorf("failed to parse template %s: %w", field.Name, err)
			} else {
				field.template = tpl
			}
		}
	}

	return nil
}

// Parse processes a log line based on the configured format (JSON or plain)
// and returns a map of field names to values extracted from the log line.
func (f *FileAgent) Parse(logLine string) (map[string]string, error) {
	if logLine == "" {
		return nil, nil
	}

	var raw map[string]string
	var err error

	switch f.Format {
	case "json":
		raw, err = f.parseJSON(logLine)
	default: // "plain" or any other format defaults to plain
		raw, err = f.parsePlain(logLine)
	}

	if raw == nil || err != nil {
		return nil, err
	}

	record := make(map[string]string)

	for i := range f.Fields {
		field := &f.Fields[i]
		if field.Name == "time" {
			if t, err := f.timestamp.Convert(raw[field.source]); err != nil {
				return nil, fmt.Errorf("failed to convert timestamp: %w", err)
			} else {
				record[field.Name] = t
			}
			continue
		}

		if field.template != nil {
			var str strings.Builder
			if err := field.template.Execute(&str, raw); err != nil {
				return nil, fmt.Errorf("failed to execute template: %w", err)
			}
			record[field.Name] = str.String()
			continue
		}

		if val, ok := raw[field.source]; ok {
			record[field.Name] = val
		} else {
			record[field.Name] = ""
		}
	}

	return record, nil
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
