package system

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/shirou/gopsutil/v4/disk"
)

var diskFields = []*field.Field{
	{
		Name:        "disk_dev",
		Type:        "string",
		Description: "Disk device name",
	},
	{
		Name:        "disk_usage",
		Type:        "float",
		Description: "Disk usage percentage",
	},
	{
		Name:        "disk_total",
		Type:        "int",
		Description: "Disk total size in bytes",
	},
	{
		Name:        "disk_read_bytes",
		Type:        "int",
		Description: "Delta of read bytes",
	},
	{
		Name:        "disk_write_bytes",
		Type:        "int",
		Description: "Delta of write bytes",
	},
}

type diskInfo struct {
	Path string `yaml:"path"`

	dev     string
	ok      bool
	ioRead  uint64
	ioWrite uint64
}

func (di *diskInfo) init() error {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("get partitions: %w", err)
	}

	sort.Slice(partitions, func(i, j int) bool {
		return len(partitions[i].Mountpoint) > len(partitions[j].Mountpoint)
	})

	for _, p := range partitions {
		if strings.HasPrefix(di.Path, p.Mountpoint) {
			di.dev = filepath.Base(p.Device)
			break
		}
	}

	return nil
}

func (di *diskInfo) getIO(data map[string]any) error {
	data["disk_read_bytes"] = int64(0)
	data["disk_write_bytes"] = int64(0)

	if di.dev == "" {
		return nil
	}

	counters, err := disk.IOCounters(di.dev)
	if err != nil {
		return fmt.Errorf("get disk io counters: %w", err)
	}

	diskIO, ok := counters[di.dev]
	if !ok {
		return nil
	}

	if di.ok {
		data["disk_read_bytes"] = int64(diskIO.ReadBytes - di.ioRead)
		data["disk_write_bytes"] = int64(diskIO.WriteBytes - di.ioWrite)
	} else {
		di.ok = true
	}

	di.ioRead = diskIO.ReadBytes
	di.ioWrite = diskIO.WriteBytes

	return nil
}

func (di *diskInfo) collect(data map[string]any) error {
	diskStat, err := disk.Usage(di.Path)
	if err != nil {
		return fmt.Errorf("get disk usage: %w", err)
	}

	data["disk_dev"] = di.dev

	data["disk_usage"] = math.Round(diskStat.UsedPercent*100) / 100
	data["disk_total"] = int64(diskStat.Total)

	if err := di.getIO(data); err != nil {
		return fmt.Errorf("get disk io: %w", err)
	}

	return nil
}
