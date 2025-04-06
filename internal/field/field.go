package field

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

type fieldConverter interface {
	Default() any
	Convert(map[string]string) (any, error)
}

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
	Timestamp *TimestampFormat `yaml:"timestamp,omitempty"`

	converter fieldConverter
}

func (f *Field) Init() error {
	source := f.Source
	if source == "" {
		source = f.Name
	}

	if f.Timestamp != nil {
		f.Type = "time"

		if err := f.Timestamp.Init(); err != nil {
			return fmt.Errorf("invalid timestamp format: %w", err)
		}

		f.converter = &timestampConverter{
			source:    source,
			timestamp: f.Timestamp,
		}

		return nil
	}

	if f.Template != "" {
		f.Type = "string"

		if tpl, err := template.New(f.Name).Parse(f.Template); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", f.Name, err)
		} else {
			f.converter = &templateConverter{tpl}
		}

		return nil
	}

	switch f.Type {
	case "", "string":
		f.converter = &stringConverter{source}
	case "int":
		f.converter = &intConverter{source}
	case "float":
		f.converter = &floatConverter{source}
	default:
		return fmt.Errorf("invalid field type '%s' for field '%s'", f.Type, f.Name)
	}

	return nil
}

func (f *Field) Default() any {
	return f.converter.Default()
}

// Convert converts the field value from the source data.
func (f *Field) Convert(data map[string]string) (any, error) {
	return f.converter.Convert(data)
}

type templateConverter struct {
	tpl *template.Template
}

func (t *templateConverter) Default() any {
	return ""
}

func (t *templateConverter) Convert(data map[string]string) (any, error) {
	var str strings.Builder
	if err := t.tpl.Execute(&str, data); err == nil {
		return str.String(), nil
	} else {
		return nil, err
	}
}

type timestampConverter struct {
	source    string
	timestamp *TimestampFormat
}

func (t *timestampConverter) Default() any {
	return int64(0)
}

func (t *timestampConverter) Convert(data map[string]string) (any, error) {
	if val, ok := data[t.source]; ok {
		if t.timestamp != nil {
			return t.timestamp.Convert(val)
		}
	}

	return nil, nil
}

type stringConverter struct {
	source string
}

func (s *stringConverter) Default() any {
	return ""
}

func (s *stringConverter) Convert(data map[string]string) (any, error) {
	if val, ok := data[s.source]; ok {
		return val, nil
	}
	return nil, nil
}

type intConverter struct {
	source string
}

func (i *intConverter) Default() any {
	return int64(0)
}

func (i *intConverter) Convert(data map[string]string) (any, error) {
	if val, ok := data[i.source]; ok {
		if intVal, err := strconv.ParseInt(val, 0, 64); err == nil {
			return intVal, nil
		} else {
			return nil, err
		}
	}

	return nil, nil
}

type floatConverter struct {
	source string
}

func (f *floatConverter) Default() any {
	return float64(0)
}

func (f *floatConverter) Convert(data map[string]string) (any, error) {
	if val, ok := data[f.source]; ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal, nil
		} else {
			return nil, err
		}
	}

	return nil, nil
}
