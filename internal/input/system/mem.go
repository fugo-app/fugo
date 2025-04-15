package system

import (
	"fmt"
	"math"

	"github.com/shirou/gopsutil/v4/mem"
)

func collectMem(data map[string]any) error {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("get memory status: %w", err)
	}

	used := float64(memStat.Total - memStat.Available)
	total := float64(memStat.Total)
	data["mem_usage"] = int64(math.Round(used * 100.0 / total))

	data["mem_total"] = int64(memStat.Total)

	return nil
}
