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

	inner SinkDriver
}

func (s *SinkConfig) Open() error {
	if s.SQLite != nil {
		if err := s.SQLite.Open(); err != nil {
			return err
		}
		s.inner = s.SQLite
	} else {
		s.inner = &DummySink{}
	}

	return nil
}

func (s *SinkConfig) Close() error {
	return s.inner.Close()
}

func (s *SinkConfig) Migrate(table string, fields []*field.Field) error {
	return s.inner.Migrate(table, fields)
}

func (s *SinkConfig) Write(table string, data map[string]any) {
	s.inner.Write(table, data)
}
