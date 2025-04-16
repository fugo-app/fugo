package system

import (
	"fmt"
	"math"

	"github.com/shirou/gopsutil/v4/disk"
)

type diskInfo struct {
	Path string
}

func (di *diskInfo) collect(data map[string]any) error {
	diskStat, err := disk.Usage(di.Path)
	if err != nil {
		return fmt.Errorf("get disk usage: %w", err)
	}

	data["disk_usage"] = math.Round(diskStat.UsedPercent*100) / 100
	data["disk_total"] = int64(diskStat.Total)

	return nil
}
