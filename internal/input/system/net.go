package system

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fugo-app/fugo/internal/field"
)

var netFields = []*field.Field{
	{
		Name:        "net_if",
		Type:        "string",
		Description: "Network interface name",
	},
	{
		Name:        "net_rx_bytes",
		Type:        "int",
		Description: "Delta of received bytes",
	},
	{
		Name:        "net_tx_bytes",
		Type:        "int",
		Description: "Delta of transmitted bytes",
	},
	{
		Name:        "net_rx_packets",
		Type:        "int",
		Description: "Delta of received packets",
	},
	{
		Name:        "net_tx_packets",
		Type:        "int",
		Description: "Delta of transmitted packets",
	},
	{
		Name:        "net_rx_errors",
		Type:        "int",
		Description: "Delta of receive errors",
	},
	{
		Name:        "net_tx_errors",
		Type:        "int",
		Description: "Delta of transmit errors",
	},
	{
		Name:        "net_rx_dropped",
		Type:        "int",
		Description: "Delta of dropped incoming packets",
	},
	{
		Name:        "net_tx_dropped",
		Type:        "int",
		Description: "Delta of dropped outgoing packets",
	},
}

type netInfo struct {
	Interface string `yaml:"interface,omitempty"`

	initialized bool

	ifname     string
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

func (ni *netInfo) init() error {
	if ni.Interface != "default" {
		ni.ifname = ni.Interface
	} else {
		iface, err := getDefaultInterface()
		if err != nil {
			return fmt.Errorf("get default interface: %w", err)
		}
		ni.ifname = iface
	}

	return nil
}

func (ni *netInfo) collect(data map[string]any) error {
	data["net_if"] = ni.ifname

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
		val := netReadStat(ni.ifname, field.name)
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
