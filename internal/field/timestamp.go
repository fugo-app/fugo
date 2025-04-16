package field

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var stdTimeNow = time.Now

type timeParser interface {
	Parse(string) (int64, error)
}

// TimestampFormat defines how to parse timestamps in log lines.
// Returns time in RFC3339 format with milliseconds.
type TimestampFormat struct {
	// Format of the timestamp.
	// Default: "rfc3339"
	// Supported: "rfc3339", "common", "unix", or custom Go layout (e.g. "2006-01-02 15:04:05")
	Format string `yaml:"format,omitempty"`

	// Go time layout used in time.Parse
	parser timeParser
}

func (t *TimestampFormat) Clone() *TimestampFormat {
	if t == nil {
		return nil
	}

	return &TimestampFormat{
		Format: t.Format,
	}
}

// Init initializes the TimestampFormat by converting the Format field
// to the corresponding Go time layout format stored in layout.
func (t *TimestampFormat) Init() error {
	format := strings.ToLower(t.Format)
	if format == "" {
		format = "rfc3339"
	}

	// Convert named formats to Go time layout format
	switch format {
	case "rfc3339":
		t.parser = &stdTimeParser{
			layout: time.RFC3339,
		}
	case "rfc3339nano":
		t.parser = &stdTimeParser{
			layout: time.RFC3339Nano,
		}
	case "common":
		// Common log format used by web servers
		t.parser = &stdTimeParser{
			layout: "02/Jan/2006:15:04:05 -0700",
		}
	case "stamp":
		t.parser = &stdTimeParser{
			layout: time.Stamp,
		}
	case "unix":
		// Unix timestamp doesn't need a layout as it will be parsed differently
		t.parser = &unixTimeParser{}
	default:
		// Assume Format is already in Go's time layout syntax
		t.parser = &stdTimeParser{
			layout: t.Format,
		}
	}

	return nil
}

// Convert processes a log record and converts the timestamp field specified by Source
// to unix timestamp format with millisecond precision.
func (t *TimestampFormat) Convert(source string) (int64, error) {
	return t.parser.Parse(source)
}

type stdTimeParser struct {
	layout string
}

func (p *stdTimeParser) Parse(source string) (int64, error) {
	parsedTime, err := time.Parse(p.layout, source)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp '%s' (%s): %w", source, p.layout, err)
	}

	if parsedTime.Year() == 0 {
		now := stdTimeNow()

		// Handle new year case
		if parsedTime.Month() == 12 && now.Month() == 1 {
			year := now.Year() - 1
			parsedTime = parsedTime.AddDate(year, 0, 0)
		} else {
			year := now.Year()
			parsedTime = parsedTime.AddDate(year, 0, 0)
		}
	}

	return parsedTime.UnixMilli(), nil
}

type unixTimeParser struct{}

func (unixTimeParser) Parse(source string) (int64, error) {
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
