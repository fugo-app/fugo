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

	// Limit the number of lines to read from the file on first read.
	Limit int `yaml:"limit,omitempty"`

	mutex   sync.Mutex
	offsets map[string]int64

	debounce *debounce.Debounce
}

var globalFileConfig *FileConfig

func (fc *FileConfig) InitDefault(dir string) {
	fc.Offsets = filepath.Join(dir, "offsets.yaml")
	fc.Limit = 100
}

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

	if fc.Limit == 0 {
		return 0
	}

	// File not found so get limited offset
	return getFileOffset(path, fc.Limit)
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

func getFileOffset(path string, lines int) int64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return 0 // Empty file
	}

	const bufferSize int64 = 4096
	buffer := make([]byte, bufferSize)

	newlineCount := 0
	offset := fileSize - 1

	if _, err := file.ReadAt(buffer[:1], offset); err == nil {
		if buffer[0] == '\n' {
			newlineCount = 1
		}
	}

	for offset > 0 && newlineCount <= lines {
		readSize := bufferSize
		if offset < bufferSize {
			readSize = offset
		}

		start := offset - readSize
		_, err := file.ReadAt(buffer[:readSize], start)
		if err != nil {
			return 0
		}

		offset -= readSize

		for i := readSize - 1; i >= 0; i-- {
			if buffer[i] == '\n' {
				newlineCount += 1
				if newlineCount > lines {
					return offset + i + 1
				}
			}
		}
	}

	return 0
}
