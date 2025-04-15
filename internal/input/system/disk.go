package system

import (
	"fmt"
	"math"

	"github.com/shirou/gopsutil/v4/disk"
)

func collectDisk(data map[string]any) error {
	diskStat, err := disk.Usage("/var/lib")
	if err != nil {
		return fmt.Errorf("get disk usage: %w", err)
	}

	data["disk_usage"] = int64(math.Round(diskStat.UsedPercent))
	data["disk_total"] = int64(diskStat.Total)

	return nil
}
