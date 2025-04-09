package storage

import (
	"path/filepath"

	"github.com/fugo-app/fugo/internal/field"
)

type StorageDriver interface {
	Open() error
	Close() error
	Migrate(string, []*field.Field) error
	Write(string, map[string]any)
}

type StorageConfig struct {
	SQLite *SQLiteStorage `yaml:"sqlite,omitempty"`

	inner StorageDriver
}

// InitDefault initializes the default storage configuration
func (s *StorageConfig) InitDefault(dir string) {
	s.SQLite = &SQLiteStorage{
		Path: filepath.Join(dir, "fugo.db"),
	}
}

func (s *StorageConfig) Open() error {
	if s.SQLite != nil {
		if err := s.SQLite.Open(); err != nil {
			return err
		}
		s.inner = s.SQLite
	} else {
		s.inner = &DummyStorage{}
	}

	return nil
}

func (s *StorageConfig) Close() error {
	return s.inner.Close()
}

func (s *StorageConfig) Migrate(table string, fields []*field.Field) error {
	return s.inner.Migrate(table, fields)
}

func (s *StorageConfig) Write(table string, data map[string]any) {
	s.inner.Write(table, data)
}
