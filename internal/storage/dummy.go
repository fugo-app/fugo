package storage

import (
	"encoding/json"
	"fmt"

	"github.com/fugo-app/fugo/internal/field"
)

type DummyStorage struct{}

func (DummyStorage) Open() error  { return nil }
func (DummyStorage) Close() error { return nil }

func (DummyStorage) Migrate(name string, fields []*field.Field) error { return nil }

func (DummyStorage) Write(name string, data map[string]any) {
	line, _ := json.Marshal(data)
	fmt.Println(name, string(line))
}
