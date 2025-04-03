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
	// Supported: "rfc3339", "unix", or custom Go layout (e.g. "2006-01-02 15:04:05")
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
// to RFC3339 format with millisecond precision, storing it in the "time" field.
// The original timestamp field is removed from the map.
func (t *TimestampFormat) Convert(source string) (string, error) {

	var parsedTime time.Time
	var err error

	// Parse the timestamp based on the configured format
	if t.layout == "unix" {
		sec, frac, ok := strings.Cut(source, ".")

		seconds, err := strconv.ParseInt(sec, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid timestamp '%s' (seconds): %w", source, err)
		}

		nanoseconds := int64(0)
		if ok {
			frac = frac + "000000000"
			frac = frac[:9]
			nanoseconds, err = strconv.ParseInt(frac, 10, 64)
			if err != nil {
				return "", fmt.Errorf("invalid timestamp '%s' (fraction): %w", source, err)
			}
		}

		parsedTime = time.Unix(seconds, nanoseconds).UTC()
	} else {
		// Parse using the configured layout
		parsedTime, err = time.Parse(t.layout, source)
		if err != nil {
			return "", fmt.Errorf("invalid timestamp '%s' (%s): %w", source, t.Format, err)
		}
	}

	// Format the parsed time as RFC3339 with millisecond precision
	return parsedTime.Format("2006-01-02T15:04:05.000Z07:00"), nil
}
