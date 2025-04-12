package file

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type RotationConfig struct {
	// Rotation method. Options: "truncate", "rename"
	Method string `yaml:"method,omitempty"`
	Size   string `yaml:"size,omitempty"`
	Run    string `yaml:"run,omitempty"`

	rotator fileRotator
	size    int64
}

type fileRotator interface {
	CheckSize(size int64) bool
	Rotate(string) error
}

var reSize = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)([km]b)?$`)

func (rc *RotationConfig) Init() error {
	matches := reSize.FindStringSubmatch(rc.Size)
	if len(matches) == 0 {
		return fmt.Errorf("invalid size format: %s", rc.Size)
	}

	val, unit := matches[1], strings.ToLower(matches[2])
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("invalid number: %w", err)
	}

	multiplier := float64(1)
	switch unit {
	case "kb":
		multiplier = 1024
	case "mb":
		multiplier = 1024 * 1024
	}

	rc.size = int64(num * multiplier)

	method := strings.ToLower(rc.Method)
	switch method {
	case "":
		return fmt.Errorf("rotation method is required")
	case "truncate":
		rc.rotator = &truncateFile{}
	case "rename":
		if rc.Run == "" {
			return fmt.Errorf("run command is required for rename method")
		}
		rc.rotator = &renameFile{}
	default:
		return fmt.Errorf("unsupported rotation method: %s", method)
	}

	return nil
}

func (rc *RotationConfig) CheckSize(size int64) bool {
	return (rc != nil) && (rc.size > 0) && (size >= rc.size)
}

func (rc *RotationConfig) Rotate(path string) error {
	if err := rc.rotator.Rotate(path); err != nil {
		return err
	}

	if rc.Run != "" {
		go rc.run(path)
	}

	return nil
}

func (rc *RotationConfig) run(path string) {
	cmd := exec.Command("sh", "-c", rc.Run)

	// Capture stdout and stderr
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to start script for %s: %v", path, err)
		text := stderr.String()
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			log.Print(line)
		}
	}
}

type truncateFile struct{}

func (t *truncateFile) CheckSize(size int64) bool {
	return size > 0
}

func (t *truncateFile) Rotate(path string) error {
	if err := os.Truncate(path, 0); err != nil {
		return fmt.Errorf("truncate file: %w", err)
	}

	return nil
}

type renameFile struct{}

func (r *renameFile) CheckSize(size int64) bool {
	return size > 0
}

func (r *renameFile) Rotate(path string) error {
	tmp := path + ".remove"

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat file: %w", err)
	}

	// Preserve file mode and ownership
	mode := info.Mode()
	stat := info.Sys().(*syscall.Stat_t)
	uid := int(stat.Uid)
	gid := int(stat.Gid)

	if err := os.Rename(path, tmp); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create new file: %w", err)
	}
	defer file.Close()

	if err := file.Chown(uid, gid); err != nil {
		return fmt.Errorf("change file ownership: %w", err)
	}

	go os.Remove(tmp)

	return nil
}
