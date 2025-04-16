package system

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type netInfo struct {
	initialized bool

	rx_bytes   int64
	tx_bytes   int64
	rx_packets int64
	tx_packets int64
	rx_errors  int64
	tx_errors  int64
	rx_dropped int64
	tx_dropped int64
}

var re = regexp.MustCompile(`^(\S+)\s+(\S+)`)

func getDefaultInterface() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("open route list: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		if matches[2] == "00000000" {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading route list: %w", err)
	}

	return "", fmt.Errorf("default interface not found")
}

func netReadStat(iface string, key string) int64 {
	path := filepath.Join("/sys/class/net", iface, "statistics", key)

	if bval, err := os.ReadFile(path); err == nil {
		sval := strings.TrimSpace(string(bval))
		if nval, err := strconv.ParseInt(sval, 10, 64); err == nil {
			return nval
		}
	}

	return 0
}

func (ni *netInfo) collect(data map[string]any) error {
	iface, err := getDefaultInterface()
	if err != nil {
		return fmt.Errorf("get network usage: %w", err)
	}

	data["net_if"] = iface

	fields := []struct {
		name string
		ptr  *int64
	}{
		{"rx_bytes", &ni.rx_bytes},
		{"tx_bytes", &ni.tx_bytes},
		{"rx_packets", &ni.rx_packets},
		{"tx_packets", &ni.tx_packets},
		{"rx_errors", &ni.rx_errors},
		{"tx_errors", &ni.tx_errors},
		{"rx_dropped", &ni.rx_dropped},
		{"tx_dropped", &ni.tx_dropped},
	}

	for _, field := range fields {
		val := netReadStat(iface, field.name)
		delta := int64(0)
		if ni.initialized {
			delta = val - *field.ptr
		}
		*field.ptr = val

		key := "net_" + field.name
		data[key] = delta
	}

	if !ni.initialized {
		ni.initialized = true
	}

	return nil
}
