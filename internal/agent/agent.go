package agent

import (
	"fmt"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/input/file"
	"github.com/fugo-app/fugo/internal/storage"
)

type Agent struct {
	name string

	// Fields to include in the final log record.
	Fields []*field.Field `yaml:"fields"`

	// File-based input.
	File *file.FileWatcher `yaml:"file,omitempty"`

	// Retention configuration
	Retention storage.RetentionConfig `yaml:"retention,omitempty"`

	storage storage.StorageDriver
}

func (a *Agent) Init(name string, storage storage.StorageDriver) error {
	a.storage = storage

	if name == "" {
		return fmt.Errorf("name is required")
	}
	a.name = name

	if len(a.Fields) == 0 {
		return fmt.Errorf("fields are required")
	}

	var timefield string

	for i := range a.Fields {
		field := a.Fields[i]
		if err := field.Init(); err != nil {
			return fmt.Errorf("field %s init: %w", field.Name, err)
		}

		if timefield == "" && field.Type == "time" {
			timefield = field.Name
		}
	}

	if timefield == "" {
		return fmt.Errorf("time field is required")
	}

	if a.File != nil {
		if err := a.File.Init(a); err != nil {
			return fmt.Errorf("file agent init: %w", err)
		}
	}

	if err := a.Retention.Init(name, timefield, storage); err != nil {
		return fmt.Errorf("retention init: %w", err)
	}

	return nil
}

func (a *Agent) Start() {
	if a.File != nil {
		a.File.Start()
	}

	a.Retention.Start()
}

func (a *Agent) Stop() {
	if a.File != nil {
		a.File.Stop()
	}

	a.Retention.Stop()
}

// Process receives raw data from input and converts to the logs record.
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

	a.Write(result)
}

func (a *Agent) Write(data map[string]any) {
	if len(data) == 0 {
		return
	}

	a.storage.Write(a.name, data)
}
