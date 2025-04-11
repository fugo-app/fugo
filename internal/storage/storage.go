package storage

import (
	"io"
	"path/filepath"
	"time"

	"github.com/fugo-app/fugo/internal/field"
)

type StorageDriver interface {
	Open() error
	Close() error
	Migrate(string, []*field.Field) error
	Cleanup(string, string, time.Duration) error
	Write(string, map[string]any)
	Query(io.Writer, *Query) error
}

type StorageConfig struct {
	SQLite *SQLiteStorage `yaml:"sqlite,omitempty"`

	inner StorageDriver
}

// InitDefault initializes the default storage configuration
func (sc *StorageConfig) InitDefault(dir string) {
	sc.SQLite = &SQLiteStorage{
		Path: filepath.Join(dir, "fugo.db"),
	}
}

func (sc *StorageConfig) Open() error {
	if sc.SQLite != nil {
		if err := sc.SQLite.Open(); err != nil {
			return err
		}
		sc.inner = sc.SQLite
	} else {
		sc.inner = &DummyStorage{}
	}

	return nil
}

func (sc *StorageConfig) Close() error {
	return sc.inner.Close()
}

func (sc *StorageConfig) Migrate(table string, fields []*field.Field) error {
	return sc.inner.Migrate(table, fields)
}

func (sc *StorageConfig) Cleanup(table, field string, duration time.Duration) error {
	return sc.inner.Cleanup(table, field, duration)
}

func (sc *StorageConfig) Write(table string, data map[string]any) {
	sc.inner.Write(table, data)
}

func (sc *StorageConfig) Query(w io.Writer, q *Query) error {
	return sc.inner.Query(w, q)
}
