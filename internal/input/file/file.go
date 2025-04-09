package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/fugo-app/fugo/pkg/debounce"
)

type FileConfig struct {
	// Path to the offset storage file.
	// Example: "/var/lib/fugo/offsets.yaml"
	Offsets string `yaml:"offsets,omitempty"`

	mutex   sync.Mutex
	offsets map[string]int64

	debounce *debounce.Debounce
}

var globalFileConfig *FileConfig

func (fc *FileConfig) Open() error {
	globalFileConfig = fc

	if fc.Offsets != "" {
		dir := filepath.Dir(fc.Offsets)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create offsets directory: %w", err)
		}

		// Load existing offsets
		data, err := os.ReadFile(fc.Offsets)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("read offsets file: %w", err)
			}
		} else {
			if err := yaml.Unmarshal(data, &fc.offsets); err != nil {
				return fmt.Errorf("unmarshal offsets: %w", err)
			}
		}
	}

	if fc.offsets == nil {
		fc.offsets = make(map[string]int64)
	}

	fc.debounce = debounce.NewDebounce(fc.save, time.Second, false)
	fc.debounce.Start()

	return nil
}

func (fc *FileConfig) Close() error {
	fc.debounce.Stop()
	fc.save()

	return nil
}

func (fc *FileConfig) getOffset(path string) int64 {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if offset, ok := fc.offsets[path]; ok {
		return offset
	}

	return 0
}

func (fc *FileConfig) setOffset(path string, offset int64) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	fc.offsets[path] = offset
	fc.debounce.Emit()
}

func (fc *FileConfig) prepare() []byte {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	out, err := yaml.Marshal(&fc.offsets)
	if err != nil {
		log.Printf("Error marshalling offsets: %v", err)
		out = nil
	}

	return out
}

func (fc *FileConfig) save() {
	if fc.Offsets == "" {
		return
	}

	data := fc.prepare()
	if data == nil {
		return
	}

	if err := os.WriteFile(fc.Offsets, data, 0644); err != nil {
		log.Printf("Error writing offsets to file: %v", err)
	}
}

func getOffset(path string) int64 {
	return globalFileConfig.getOffset(path)
}

func setOffset(path string, offset int64) {
	globalFileConfig.setOffset(path, offset)
}
