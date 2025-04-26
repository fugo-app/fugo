package system

import (
	"fmt"
	"math"
	"runtime"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

var cpuFields = []*field.Field{
	{
		Name:        "la_1",
		Type:        "float",
		Description: "Load average for 1 minute",
	},
	{
		Name:        "la_5",
		Type:        "float",
		Description: "Load average for 5 minutes",
	},
	{
		Name:        "la_15",
		Type:        "float",
		Description: "Load average for 15 minutes",
	},
	{
		Name:        "cpu_usage",
		Type:        "float",
		Description: "CPU usage percentage",
	},
	{
		Name:        "cpu_cores",
		Type:        "int",
		Description: "Number of CPU cores",
	},
}

type cpuInfo struct {
	ok   bool
	used float64
	idle float64
}

func (ci *cpuInfo) collect(data map[string]any) error {
	// Load average
	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Errorf("get load average: %w", err)
	}
	data["la_1"] = math.Round(loadAvg.Load1*100) / 100
	data["la_5"] = math.Round(loadAvg.Load5*100) / 100
	data["la_15"] = math.Round(loadAvg.Load15*100) / 100

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

	if ci.ok {
		deltaUsed := cpuUsed - ci.used
		deltaIdle := cpuIdle - ci.idle
		total := deltaUsed + deltaIdle
		data["cpu_usage"] = math.Round(deltaUsed/total*100*100) / 100
	} else {
		data["cpu_usage"] = float64(0)
		ci.ok = true
	}

	data["cpu_cores"] = int64(runtime.NumCPU())

	ci.used = cpuUsed
	ci.idle = cpuIdle

	return nil
}
