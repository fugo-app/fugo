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

	fields  []*field.Field
	storage storage.StorageDriver
}

func (a *Agent) Init(name string, storage storage.StorageDriver) error {
	a.storage = storage

	if name == "" {
		return fmt.Errorf("name is required")
	}
	a.name = name

	if len(a.Fields) == 0 {
		// Try to set default fields
	} else {
		a.fields = make([]*field.Field, len(a.Fields))
		for i := range a.Fields {
			a.fields[i] = a.Fields[i].Clone()
		}
	}

	var timefield string

	for i := range a.fields {
		field := a.fields[i]
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

	if err := a.storage.Migrate(name, a.fields); err != nil {
		return fmt.Errorf("migrate agent (%s): %w", name, err)
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

func (a *Agent) Serialize(data map[string]string) map[string]any {
	if len(data) == 0 {
		return nil
	}

	result := make(map[string]any)

	for i := range a.fields {
		field := a.fields[i]
		if val, err := field.Convert(data); err == nil && val != nil {
			result[field.Name] = val
		} else {
			result[field.Name] = field.Default()
		}
	}

	return result
}

func (a *Agent) Write(data map[string]any) {
	if len(data) == 0 {
		return
	}

	a.storage.Write(a.name, data)
}
