package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fugo-app/fugo/internal/field"
)

type DummyStorage struct{}

func (DummyStorage) Open() error  { return nil }
func (DummyStorage) Close() error { return nil }

func (DummyStorage) Migrate(name string, fields []*field.Field) error {
	return nil
}

func (DummyStorage) Cleanup(name string, field string, retention time.Duration) error {
	return nil
}

func (DummyStorage) Write(name string, data map[string]any) {
	line, _ := json.Marshal(data)
	fmt.Println(name, string(line))
}

func (DummyStorage) Query(w io.Writer, q *Query) error {
	return nil
}
