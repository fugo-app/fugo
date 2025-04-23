package system

import (
	"fmt"
	"math"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

type cpuInfo struct {
	used float64
	idle float64
}

func (ci *cpuInfo) collect(data map[string]any) error {
	// Load average
	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Errorf("get load average: %w", err)
	}
	data["la_1"] = loadAvg.Load1
	data["la_5"] = loadAvg.Load5
	data["la_15"] = loadAvg.Load15

	// Calculate CPU usage
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		return fmt.Errorf("get cpu times: %w", err)
	}

	cpuTime := cpuTimes[0]
	cpuUsed := cpuTime.User +
		cpuTime.Nice +
		cpuTime.System +
		cpuTime.Irq +
		cpuTime.Softirq +
		cpuTime.Steal +
		cpuTime.Guest +
		cpuTime.GuestNice
	cpuIdle := cpuTime.Idle + cpuTime.Iowait

	if ci.used != 0 {
		deltaUsed := cpuUsed - ci.used
		deltaIdle := cpuIdle - ci.idle
		total := deltaUsed + deltaIdle
		data["cpu_usage"] = math.Round(deltaUsed/total*100*100) / 100
	} else {
		data["cpu_usage"] = float64(0)
	}

	data["cpu_cores"] = int64(runtime.NumCPU())

	ci.used = cpuUsed
	ci.idle = cpuIdle

	return nil
}
