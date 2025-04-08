package sink

import "github.com/fugo-app/fugo/internal/field"

type SinkDriver interface {
	Migrate(string, []*field.Field) error
}
