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
	// Template to convert source fields into new record field.
	Template string `yaml:"template,omitempty"`
	// Source field name to extract the value from.
	Source string `yaml:"source,omitempty"`
	// Time format only for the "time" field.
	TimeFormat string `yaml:"time_format,omitempty"`
	// Feild type: "string", "int", "float". Default: "string"
	Type string `yaml:"type,omitempty"`

	source    string
	template  *template.Template
	timestamp *TimestampFormat
}

func (f *Field) Init() error {
	f.source = f.Source
	if f.source == "" {
		f.source = f.Name
	}

	if f.Name == "time" {
		f.timestamp = &TimestampFormat{
			Format: f.TimeFormat,
		}

		if err := f.timestamp.Init(); err != nil {
			return fmt.Errorf("failed to initialize timestamp format: %w", err)
		}
		return nil
	}

	if f.Template != "" {
		if tpl, err := template.New(f.Name).Parse(f.Template); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", f.Name, err)
		} else {
			f.template = tpl
		}
	}

	return nil
}

func (f *Field) Process(data map[string]string) (any, error) {
	if f.Name == "time" {
		if t, err := f.timestamp.Convert(data[f.source]); err == nil {
			return t, nil
		} else {
			return nil, fmt.Errorf("failed to convert timestamp: %w", err)
		}
	}

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
