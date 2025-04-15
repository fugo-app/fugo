package system

import (
	"fmt"
	"math"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

type cpuUsage struct {
	usage float64
	idle  float64
}

func (c *cpuUsage) collect(data map[string]any) error {
	// Load average
	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Errorf("get load average: %w", err)
	}
	data["la_1"] = int64(math.Round(loadAvg.Load1))
	data["la_5"] = int64(math.Round(loadAvg.Load5))
	data["la_15"] = int64(math.Round(loadAvg.Load15))

	// Calculate CPU usage
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		return fmt.Errorf("get cpu times: %w", err)
	}

	cpuTime := cpuTimes[0]
	cpuUsage := cpuTime.User +
		cpuTime.Nice +
		cpuTime.System +
		cpuTime.Irq +
		cpuTime.Softirq +
		cpuTime.Steal +
		cpuTime.Guest +
		cpuTime.GuestNice
	cpuIdle := cpuTime.Idle + cpuTime.Iowait

	if c.usage != 0 {
		deltaUsage := cpuUsage - c.usage
		deltaIdle := cpuIdle - c.idle
		total := deltaUsage + deltaIdle
		data["cpu_usage"] = int64(math.Round(deltaUsage * 100.0 / total))
	} else {
		data["cpu_usage"] = int64(0)
	}

	data["cpu_cores"] = int64(runtime.NumCPU())

	c.usage = cpuUsage
	c.idle = cpuIdle

	return nil
}
