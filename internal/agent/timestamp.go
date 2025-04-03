package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimestampFormat defines how to parse timestamps in log lines.
// It sets the source field name and the format used.
// The parsed result is stored in the "time" field in RFC3339 format with milliseconds.
type TimestampFormat struct {
	// Field name containing the timestamp.
	Source string `yaml:"source"`

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
	// Default to RFC3339 if format is not specified
	if t.Format == "" {
		t.Format = "rfc3339"
	}

	// Convert named formats to Go time layout format
	switch t.Format {
	case "rfc3339":
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
func (t *TimestampFormat) Convert(record map[string]string) error {
	// If no timestamp format is configured or no source field is specified
	if t == nil || t.Source == "" {
		return nil
	}

	// Check if the source field exists in the record
	sourceValue, exists := record[t.Source]
	if !exists || sourceValue == "" {
		return fmt.Errorf("field '%s' not found in log record", t.Source)
	}

	var parsedTime time.Time
	var err error

	// Parse the timestamp based on the configured format
	if t.layout == "unix" {
		sec, frac, ok := strings.Cut(sourceValue, ".")

		seconds, err := strconv.ParseInt(sec, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid timestamp '%s' (seconds): %w", sourceValue, err)
		}

		nanoseconds := int64(0)
		if ok {
			frac = frac + "000000000"
			frac = frac[:9]
			nanoseconds, err = strconv.ParseInt(frac, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp '%s' (fraction): %w", sourceValue, err)
			}
		}

		parsedTime = time.Unix(seconds, nanoseconds)
	} else {
		// Parse using the configured layout
		parsedTime, err = time.Parse(t.layout, sourceValue)
		if err != nil {
			return fmt.Errorf("invalid timestamp '%s' (%s): %w", sourceValue, t.Format, err)
		}
	}

	// Remove the original timestamp field
	delete(record, t.Source)

	// Format the parsed time as RFC3339 with millisecond precision
	record["time"] = parsedTime.Format("2006-01-02T15:04:05.000Z07:00")

	return nil
}
