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

	// Path to check the disk usage. Default is "/var/lib"
	DiskPath string `yaml:"disk_path,omitempty"`

	interval  time.Duration
	processor input.Processor

	cpu  cpuInfo
	disk diskInfo
	net  netInfo

	stop chan struct{}
}

func (sw *SystemWatcher) Fields() []*field.Field {
	return []*field.Field{
		{Name: "time", Type: "time"},
		{Name: "uptime", Type: "int", Description: "System uptime in seconds"},
		// CPU
		{Name: "la_1", Type: "float", Description: "Load average for 1 minute"},
		{Name: "la_5", Type: "float", Description: "Load average for 5 minutes"},
		{Name: "la_15", Type: "float", Description: "Load average for 15 minutes"},
		{Name: "cpu_usage", Type: "float", Description: "CPU usage percentage"},
		{Name: "cpu_cores", Type: "int", Description: "Number of CPU cores"},
		// MEM
		{Name: "mem_usage", Type: "float", Description: "Memory usage percentage"},
		{Name: "mem_total", Type: "int", Description: "Memory total size in bytes"},
		// DISK
		{Name: "disk_usage", Type: "float", Description: "Disk usage percentage"},
		{Name: "disk_total", Type: "int", Description: "Disk total size in bytes"},
		{Name: "disk_read_bytes", Type: "int", Description: "Disk I/O read bytes"},
		{Name: "disk_write_bytes", Type: "int", Description: "Disk I/O write bytes"},
		// NET
		{Name: "net_if", Type: "string", Description: "Network interface name"},
		{Name: "net_rx_bytes", Type: "int", Description: "Network receive bytes"},
		{Name: "net_tx_bytes", Type: "int", Description: "Network transmit bytes"},
		{Name: "net_rx_packets", Type: "int", Description: "Network receive packets"},
		{Name: "net_tx_packets", Type: "int", Description: "Network transmit packets"},
		{Name: "net_rx_errors", Type: "int", Description: "Network receive errors"},
		{Name: "net_tx_errors", Type: "int", Description: "Network transmit errors"},
		{Name: "net_rx_dropped", Type: "int", Description: "Network dropped incoming packets"},
		{Name: "net_tx_dropped", Type: "int", Description: "Network dropped outgoing packets"},
	}
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

	sw.disk.init(sw.DiskPath)

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

	if err := sw.disk.collect(data); err != nil {
		return err
	}

	if err := sw.net.collect(data); err != nil {
		return err
	}

	sw.processor.Write(data)

	return nil
}
