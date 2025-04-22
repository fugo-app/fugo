package agent

import (
	"fmt"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/input/file"
	"github.com/fugo-app/fugo/internal/input/system"
	"github.com/fugo-app/fugo/internal/storage"
)

type Agent struct {
	name string

	// Fields to include in the final log record.
	Fields []*field.Field `yaml:"fields"`

	// File-based input.
	File *file.FileWatcher `yaml:"file,omitempty"`

	// System telemetry input.
	System *system.SystemWatcher `yaml:"system,omitempty"`

	// Retention configuration
	Retention storage.RetentionConfig `yaml:"retention,omitempty"`

	fields []*field.Field
	app    AppHandler
}

func (a *Agent) Init(name string, app AppHandler) error {
	a.app = app

	if name == "" {
		return fmt.Errorf("name is required")
	}
	a.name = name

	if len(a.Fields) == 0 {
		if a.System != nil {
			a.fields = a.System.Fields()
		}
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

	if a.System != nil {
		if err := a.System.Init(a); err != nil {
			return fmt.Errorf("system agent init: %w", err)
		}
	}

	if err := a.Retention.Init(name, timefield, a.app.GetStorage()); err != nil {
		return fmt.Errorf("retention init: %w", err)
	}

	if err := a.app.GetStorage().Migrate(name, a.fields); err != nil {
		return fmt.Errorf("migrate agent (%s): %w", name, err)
	}

	return nil
}

func (a *Agent) Start() {
	if a.File != nil {
		a.File.Start()
	}

	if a.System != nil {
		a.System.Start()
	}

	a.Retention.Start()
}

func (a *Agent) Stop() {
	if a.File != nil {
		a.File.Stop()
	}

	if a.System != nil {
		a.System.Stop()
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

// Write writes the serialized data to the storage.
func (a *Agent) Write(data map[string]any) {
	if len(data) == 0 {
		return
	}

	a.app.GetStorage().Write(a.name, data)
}

// GetFields returns the list of initialized fields for the agent.
func (a *Agent) GetFields() []*field.Field {
	return a.fields
}
