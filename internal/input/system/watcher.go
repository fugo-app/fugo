package system

import (
	"fmt"
	"log"
	"time"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/input"
	"github.com/fugo-app/fugo/pkg/duration"
	"github.com/shirou/gopsutil/v4/host"
)

type SystemWatcher struct {
	// Interval to check the system status. Default is 60s
	Interval string `yaml:"interval,omitempty"`

	Disk *diskInfo `yaml:"disk,omitempty"`
	Net  *netInfo  `yaml:"net,omitempty"`

	interval  time.Duration
	processor input.Processor

	cpu cpuInfo

	stop chan struct{}
}

var baseFields = []*field.Field{
	{
		Name: "time",
		Type: "time",
	},
	{
		Name:        "uptime",
		Type:        "int",
		Description: "System uptime in seconds",
	},
}

func (sw *SystemWatcher) Fields() []*field.Field {
	fields := make([]*field.Field, 0)

	fields = append(fields, baseFields...)
	fields = append(fields, cpuFields...)
	fields = append(fields, memFields...)

	if sw.Disk != nil {
		fields = append(fields, diskFields...)
	}

	if sw.Net != nil {
		fields = append(fields, netFields...)
	}

	return fields
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

	if sw.Disk != nil {
		if err := sw.Disk.init(); err != nil {
			return fmt.Errorf("init disk info: %w", err)
		}
	}

	if sw.Net != nil {
		if err := sw.Net.init(); err != nil {
			return fmt.Errorf("init net info: %w", err)
		}
	}

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
	sw.collect()

	ticker := time.NewTicker(sw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sw.collect()
		case <-sw.stop:
			return
		}
	}
}

func (sw *SystemWatcher) collect() {
	if err := sw._collect(); err != nil {
		log.Printf("Error on collecting system status: %v\n", err)
	}
}

func (sw *SystemWatcher) _collect() error {
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

	if err := sw.Disk.collect(data); err != nil {
		return err
	}

	if err := sw.Net.collect(data); err != nil {
		return err
	}

	sw.processor.Write(data)

	return nil
}
