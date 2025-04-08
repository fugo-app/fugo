package sink

import (
	"github.com/fugo-app/fugo/internal/field"
)

type SinkDriver interface {
	Open() error
	Close() error
	Migrate(string, []*field.Field) error
	Write(string, map[string]any)
}

type SinkConfig struct {
	SQLite *SQLiteSink `yaml:"sqlite,omitempty"`
}
