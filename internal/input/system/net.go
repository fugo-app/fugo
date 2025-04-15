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

func collectNet(data map[string]any) error {
	iface, err := getDefaultInterface()
	if err != nil {
		return fmt.Errorf("get network usage: %w", err)
	}

	data["net_if"] = iface
	data["net_rx_kb"] = netReadStat(iface, "rx_bytes") / 1024
	data["net_tx_kb"] = netReadStat(iface, "tx_bytes") / 1024
	data["net_rx_packets"] = netReadStat(iface, "rx_packets")
	data["net_tx_packets"] = netReadStat(iface, "tx_packets")
	data["net_rx_errors"] = netReadStat(iface, "rx_errors")
	data["net_tx_errors"] = netReadStat(iface, "tx_errors")
	data["net_rx_dropped"] = netReadStat(iface, "rx_dropped")
	data["net_tx_dropped"] = netReadStat(iface, "tx_dropped")

	return nil
}
