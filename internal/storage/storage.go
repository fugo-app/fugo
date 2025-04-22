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

func (sc *StorageConfig) GetDriver() StorageDriver {
	return sc.inner
}
