package system

import (
	"fmt"
	"log"
	"time"

	"github.com/fugo-app/fugo/internal/input"
	"github.com/fugo-app/fugo/pkg/duration"
	"github.com/shirou/gopsutil/v4/host"
)

type SystemWatcher struct {
	Interval string `yaml:"interval,omitempty"` // Interval to check the system status. Default is 60s

	interval  time.Duration
	processor input.Processor

	cpu cpuInfo
	net netInfo

	stop chan struct{}
}

func (sw *SystemWatcher) Init(processor input.Processor) error {
	sw.interval = 60 * time.Second // Default to 60 seconds
	if sw.Interval != "" {
		d, err := duration.Parse(sw.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval value: %w", err)
		}
		sw.interval = d
	}
	sw.processor = processor

	return nil
}

func (sw *SystemWatcher) Start() {
	sw.stop = make(chan struct{})
	go sw.watch()
}

func (sw *SystemWatcher) Stop() {
	if sw.stop != nil {
		close(sw.stop)
	}
}

func (sw *SystemWatcher) watch() {
	ticker := time.NewTicker(sw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sw.collect(); err != nil {
				log.Printf("Error on collecting system status: %v\n", err)
			}
		case <-sw.stop:
			return
		}
	}
}

func (sw *SystemWatcher) collect() error {
	data := make(map[string]any)

	data["time"] = time.Now().UnixMilli()

	// Uptime
	uptime, err := host.Uptime()
	if err != nil {
		return fmt.Errorf("get host uptime: %w", err)
	}
	data["uptime"] = int64(uptime)

	if err := sw.cpu.collect(data); err != nil {
		return err
	}

	if err := collectMem(data); err != nil {
		return err
	}

	if err := collectDisk(data); err != nil {
		return err
	}

	if err := sw.net.collect(data); err != nil {
		return err
	}

	sw.processor.Write(data)

	return nil
}
