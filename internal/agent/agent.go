package agent

import "fmt"

type Agent struct {
	Name string `yaml:"name"`

	// Fields to include in the final log record.
	Fields []Field `yaml:"fields"`

	// File-based log source.
	File *FileAgent `yaml:"file,omitempty"`
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
