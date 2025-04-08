package agent

import (
	"encoding/json"
	"fmt"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/sink"
	"github.com/fugo-app/fugo/internal/source/file"
)

type Agent struct {
	name string

	// Fields to include in the final log record.
	Fields []*field.Field `yaml:"fields"`

	// File-based log source.
	File *file.FileWatcher `yaml:"file,omitempty"`

	sink sink.SinkDriver
}

func (a *Agent) Init(name string, sink sink.SinkDriver) error {
	a.sink = sink

	if name == "" {
		return fmt.Errorf("name is required")
	}
	a.name = name

	if len(a.Fields) == 0 {
		return fmt.Errorf("fields are required")
	}

	for i := range a.Fields {
		field := a.Fields[i]
		if err := field.Init(); err != nil {
			return fmt.Errorf("field %s init: %w", field.Name, err)
		}
	}

	if a.File != nil {
		if err := a.File.Init(a); err != nil {
			return fmt.Errorf("file agent init: %w", err)
		}
	}

	return nil
}

func (a *Agent) Start() {
	if a.File != nil {
		a.File.Start()
	}
}

func (a *Agent) Stop() {
	if a.File != nil {
		a.File.Stop()
	}
}

// Process receives raw data from source and converts to the logs record.
func (a *Agent) Process(data map[string]string) {
	if len(data) == 0 {
		return
	}

	result := make(map[string]any)

	for i := range a.Fields {
		field := a.Fields[i]
		if val, err := field.Convert(data); err == nil && val != nil {
			result[field.Name] = val
		} else {
			result[field.Name] = field.Default()
		}
	}

	line, _ := json.Marshal(result)
	fmt.Println(a.name, string(line))

	// TODO: send data to sink
}
