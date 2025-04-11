package storage

import (
	"fmt"
	"log"
	"time"
)

type RetentionConfig struct {
	// Period is the retention period for the log records.
	// Value in the format of "1h", "1d", etc.
	Period string `yaml:"period,omitempty"`

	// Interval is the interval for the retention cleanup.
	// Value in the format of "1h", "1d", etc.
	Interval string `yaml:"interval,omitempty"`

	name    string
	field   string
	storage StorageDriver

	period   time.Duration
	interval time.Duration

	stop chan struct{}
}

func (rc *RetentionConfig) Init(name string, field string, storage StorageDriver) error {
	rc.name = name
	rc.field = field
	rc.storage = storage

	rc.period = time.Hour * 24 * 3 // Default to 3 days
	if rc.Period != "" {
		d, err := time.ParseDuration(rc.Period)
		if err != nil {
			return fmt.Errorf("invalid period value: %w", err)
		}
		rc.period = d
	}

	rc.interval = time.Hour // Default to 1 hour
	if rc.Interval != "" {
		d, err := time.ParseDuration(rc.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval value: %w", err)
		}
		rc.interval = d
	}

	return nil
}

func (rc *RetentionConfig) Start() {
	rc.stop = make(chan struct{})
	go rc.run()
}

func (rc *RetentionConfig) Stop() {
	close(rc.stop)
}

func (rc *RetentionConfig) run() {
	ticker := time.NewTicker(rc.interval)
	defer ticker.Stop()

	rc.storage.Cleanup(rc.name, rc.field, rc.period)

	for {
		select {
		case <-ticker.C:
			if err := rc.storage.Cleanup(rc.name, rc.field, rc.period); err != nil {
				log.Printf("retention cleanup for %s: %v", rc.name, err)
			}
		case <-rc.stop:
			return
		}
	}
}
