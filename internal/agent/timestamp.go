package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimestampFormat defines how to parse timestamps in log lines.
// Returns time in RFC3339 format with milliseconds.
type TimestampFormat struct {
	// Format of the timestamp.
	// Default: "rfc3339"
	// Supported: "rfc3339", "common", "unix", or custom Go layout (e.g. "2006-01-02 15:04:05")
	Format string `yaml:"format,omitempty"`

	// Go time layout used in time.Parse
	layout string
}

// Init initializes the TimestampFormat by converting the Format field
// to the corresponding Go time layout format stored in layout.
func (t *TimestampFormat) Init() error {
	// Convert named formats to Go time layout format
	switch t.Format {
	case "", "rfc3339":
		t.layout = time.RFC3339
	case "common":
		// Common log format used by web servers
		t.layout = "02/Jan/2006:15:04:05 -0700"
	case "unix":
		// Unix timestamp doesn't need a layout as it will be parsed differently
		t.layout = "unix"
	default:
		// Assume Format is already in Go's time layout syntax
		t.layout = t.Format
	}

	return nil
}

// Convert processes a log record and converts the timestamp field specified by Source
// to unix timestamp format with millisecond precision.
func (t *TimestampFormat) Convert(source string) (int64, error) {
	// Parse the timestamp based on the configured format
	if t.layout == "unix" {
		return convertUnix(source)
	}

	// Parse using the configured layout
	parsedTime, err := time.Parse(t.layout, source)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp '%s' (%s): %w", source, t.Format, err)
	}

	return parsedTime.UnixMilli(), nil
}

func convertUnix(source string) (int64, error) {
	sec, frac, ok := strings.Cut(source, ".")
	seconds, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp '%s' (seconds): %w", source, err)
	}

	milliseconds := seconds * 1000
	if !ok {
		return milliseconds, nil
	}

	limit := len(frac)
	if limit > 3 {
		limit = 3
	}

	multiply := int64(100)
	for i := 0; i < limit; i++ {
		ch := frac[i] - '0'
		milliseconds = milliseconds + (int64(ch) * multiply)
		multiply = multiply / 10
	}

	return milliseconds, nil
}
