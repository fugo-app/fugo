package agent

import (
	"fmt"

	"github.com/runcitrus/fugo/internal/field"
	"github.com/runcitrus/fugo/internal/source/file"
)

type Agent struct {
	Name string `yaml:"name"`

	// Fields to include in the final log record.
	Fields []field.Field `yaml:"fields"`

	// File-based log source.
	File *file.FileAgent `yaml:"file,omitempty"`
}

func (a *Agent) Init() error {
	if a.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(a.Fields) == 0 {
		return fmt.Errorf("fields are required")
	}

	for i := range a.Fields {
		field := &a.Fields[i]
		if err := field.Init(); err != nil {
			return fmt.Errorf("field %s init: %w", field.Name, err)
		}
	}

	if a.File != nil {
		if err := a.File.Init(); err != nil {
			return fmt.Errorf("file agent init: %w", err)
		}
	}

	return nil
}

// Convert converts the parsed log data into a map of key-value pairs.
func (a *Agent) Convert(data map[string]string) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	result := make(map[string]any)

	for i := range a.Fields {
		field := &a.Fields[i]
		if val, err := field.Convert(data); err == nil {
			if val != nil {
				result[field.Name] = val
			}
		} else {
			return nil, fmt.Errorf("failed to process field %s: %w", field.Name, err)
		}
	}

	return result, nil
}
