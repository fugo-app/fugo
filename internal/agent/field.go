package agent

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// Field represents a field in the log record.
type Field struct {
	// Name of the field in the log record.
	Name string `yaml:"name"`
	// Source field name to extract the value from.
	Source string `yaml:"source,omitempty"`
	// Feild type: "string" (default), "int", "float", "time" (default for field with time_format).
	Type string `yaml:"type,omitempty"`
	// Template to convert source fields into new record field.
	Template string `yaml:"template,omitempty"`
	// Layout to parse the time string. Only for the "time" field.
	// Formats: "rfc3339" (default), "common", "unix",
	// or custom Go layout (e.g. "2006-01-02 15:04:05")
	TimeFormat string `yaml:"time_format,omitempty"`

	source    string
	template  *template.Template
	timestamp *TimestampFormat
}

func (f *Field) Init() error {
	f.source = f.Source
	if f.source == "" {
		f.source = f.Name
	}

	if f.TimeFormat != "" {
		f.Type = "time"

		f.timestamp = &TimestampFormat{
			Format: f.TimeFormat,
		}

		if err := f.timestamp.Init(); err != nil {
			return fmt.Errorf("failed to initialize timestamp format: %w", err)
		}

		return nil
	}

	if f.Template != "" {
		f.Type = "string"

		if tpl, err := template.New(f.Name).Parse(f.Template); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", f.Name, err)
		} else {
			f.template = tpl
		}
	}

	return nil
}

// Convert converts the field value from the source data.
func (f *Field) Convert(data map[string]string) (any, error) {
	if f.template != nil {
		var str strings.Builder
		if err := f.template.Execute(&str, data); err == nil {
			return str.String(), nil
		} else {
			return nil, nil
		}
	}

	if val, ok := data[f.source]; ok {
		switch f.Type {
		case "", "string":
			return val, nil
		case "time":
			if t, err := f.timestamp.Convert(val); err == nil {
				return t, nil
			} else {
				return nil, fmt.Errorf("failed to convert timestamp: %w", err)
			}
		case "int":
			if val, err := strconv.ParseInt(val, 0, 64); err == nil {
				return val, nil
			} else {
				return nil, nil
			}
		case "float":
			if val, err := strconv.ParseFloat(val, 64); err == nil {
				return val, nil
			} else {
				return nil, nil
			}
		}
	}

	return nil, nil
}
