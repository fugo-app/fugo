package system

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

type diskInfo struct {
	path   string
	device string

	ok      bool
	ioRead  uint64
	ioWrite uint64
}

func (di *diskInfo) init(path string) error {
	if path == "" {
		path = "/var/lib"
	}
	di.path = path

	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("get partitions: %w", err)
	}

	sort.Slice(partitions, func(i, j int) bool {
		return len(partitions[i].Mountpoint) > len(partitions[j].Mountpoint)
	})

	for _, p := range partitions {
		if strings.HasPrefix(di.path, p.Mountpoint) {
			di.device = filepath.Base(p.Device)
			break
		}
	}

	return nil
}

func (di *diskInfo) getIO(data map[string]any) error {
	data["disk_read_bytes"] = int64(0)
	data["disk_write_bytes"] = int64(0)

	if di.device == "" {
		return nil
	}

	counters, err := disk.IOCounters(di.device)
	if err != nil {
		return fmt.Errorf("get disk io counters: %w", err)
	}

	diskIO, ok := counters[di.device]
	if !ok {
		return nil
	}

	if di.ok {
		data["disk_read_bytes"] = int64(diskIO.ReadCount - di.ioRead)
		data["disk_write_bytes"] = int64(diskIO.WriteCount - di.ioWrite)
	} else {
		di.ok = true
	}

	di.ioRead = diskIO.ReadCount
	di.ioWrite = diskIO.WriteCount

	return nil
}

func (di *diskInfo) collect(data map[string]any) error {
	diskStat, err := disk.Usage(di.path)
	if err != nil {
		return fmt.Errorf("get disk usage: %w", err)
	}

	data["disk_usage"] = math.Round(diskStat.UsedPercent*100) / 100
	data["disk_total"] = int64(diskStat.Total)

	if err := di.getIO(data); err != nil {
		return fmt.Errorf("get disk io: %w", err)
	}

	return nil
}
