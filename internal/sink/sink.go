package sink

import "github.com/fugo-app/fugo/internal/field"

type SinkDriver interface {
	Open() error
	Close()
	Migrate(string, []*field.Field) error
	Write(string, map[string]any)
}
