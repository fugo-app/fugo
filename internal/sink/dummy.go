package sink

import (
	"encoding/json"
	"fmt"

	"github.com/fugo-app/fugo/internal/field"
)

type DummySink struct{}

func (DummySink) Open() error {
	return nil
}

func (DummySink) Close() {}

func (DummySink) Migrate(name string, fields []*field.Field) error {
	return nil
}

func (DummySink) Write(name string, data map[string]any) {
	line, _ := json.Marshal(data)
	fmt.Println(name, string(line))
}
