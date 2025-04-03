package agent

import (
	"reflect"
	"testing"
	"time"
)

func TestTimestampFormat_Init(t *testing.T) {
	tests := []struct {
		name       string
		timestamp  *TimestampFormat
		wantLayout string
		wantFormat string
	}{
		{
			name: "default format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
			},
			wantLayout: time.RFC3339,
			wantFormat: "rfc3339",
		},
		{
			name: "rfc3339 format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
			},
			wantLayout: time.RFC3339,
			wantFormat: "rfc3339",
		},
		{
			name: "unix format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "unix",
			},
			wantLayout: "unix",
			wantFormat: "unix",
		},
		{
			name: "custom format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "2006-01-02 15:04:05",
			},
			wantLayout: "2006-01-02 15:04:05",
			wantFormat: "2006-01-02 15:04:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.timestamp.Init()

			// Check that Format is set correctly
			if tt.timestamp.Format != tt.wantFormat {
				t.Errorf("TimestampFormat.Init() Format = %v, want %v",
					tt.timestamp.Format, tt.wantFormat)
			}

			// Check that layout is set correctly
			if tt.timestamp.layout != tt.wantLayout {
				t.Errorf("TimestampFormat.Init() layout = %v, want %v",
					tt.timestamp.layout, tt.wantLayout)
			}
		})
	}
}

func TestTimestampFormat_Convert(t *testing.T) {
	// Helper function to create a record with a timestamp
	createRecord := func(field, value string) map[string]string {
		return map[string]string{field: value, "other": "value"}
	}

	tests := []struct {
		name      string
		timestamp *TimestampFormat
		record    map[string]string
		want      map[string]string
		wantErr   bool
	}{
		{
			name:      "nil timestamp",
			timestamp: nil,
			record:    createRecord("timestamp", "2023-01-01T12:00:00Z"),
			want:      createRecord("timestamp", "2023-01-01T12:00:00Z"),
			wantErr:   false,
		},
		{
			name: "empty source field",
			timestamp: &TimestampFormat{
				Source: "",
				Format: "rfc3339",
			},
			record:  createRecord("timestamp", "2023-01-01T12:00:00Z"),
			want:    createRecord("timestamp", "2023-01-01T12:00:00Z"),
			wantErr: false,
		},
		{
			name: "source field not found",
			timestamp: &TimestampFormat{
				Source: "missing",
				Format: "rfc3339",
			},
			record:  createRecord("timestamp", "2023-01-01T12:00:00Z"),
			want:    createRecord("timestamp", "2023-01-01T12:00:00Z"),
			wantErr: true,
		},
		{
			name: "rfc3339 format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
				layout: time.RFC3339,
			},
			record: createRecord("timestamp", "2023-01-01T12:00:00Z"),
			want: map[string]string{
				"time":  "2023-01-01T12:00:00.000Z",
				"other": "value",
			},
			wantErr: false,
		},
		{
			name: "invalid timestamp format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
				layout: time.RFC3339,
			},
			record:  createRecord("timestamp", "invalid-date"),
			want:    createRecord("timestamp", "invalid-date"),
			wantErr: true,
		},
		{
			name: "unix timestamp integer",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "unix",
				layout: "unix",
			},
			record: createRecord("timestamp", "1672574400"),
			want: map[string]string{
				"time":  "2023-01-01T12:00:00.000Z",
				"other": "value",
			},
			wantErr: false,
		},
		{
			name: "unix timestamp with milliseconds",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "unix",
				layout: "unix",
			},
			record: createRecord("timestamp", "1672574400.123"),
			want: map[string]string{
				"time":  "2023-01-01T12:00:00.123Z",
				"other": "value",
			},
			wantErr: false,
		},
		{
			name: "invalid unix timestamp",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "unix",
				layout: "unix",
			},
			record:  createRecord("timestamp", "not-a-number"),
			want:    createRecord("timestamp", "not-a-number"),
			wantErr: true,
		},
		{
			name: "custom format",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "2006-01-02 15:04:05",
				layout: "2006-01-02 15:04:05",
			},
			record: createRecord("timestamp", "2023-01-01 12:00:00"),
			want: map[string]string{
				"time":  "2023-01-01T12:00:00.000Z",
				"other": "value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the record so we can compare with the original
			recordCopy := make(map[string]string)
			for k, v := range tt.record {
				recordCopy[k] = v
			}

			// Initialize timestamp if not nil (to make sure layout is set)
			if tt.timestamp != nil {
				if tt.timestamp.Source != "" || tt.timestamp.Format != "" {
					tt.timestamp.Init()
				}
			}

			err := tt.timestamp.Convert(recordCopy)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("TimestampFormat.Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If error is expected, don't check the result
			if tt.wantErr {
				return
			}

			// For non-error case, check that the record matches what we expect
			// Note: for UTC time zones, we need to normalize the times because the test might
			// run in different time zones
			if recordCopy["time"] != "" {
				parsed, _ := time.Parse(time.RFC3339, recordCopy["time"])
				recordCopy["time"] = parsed.UTC().Format("2006-01-02T15:04:05.000Z")
			}
			if tt.want["time"] != "" {
				parsed, _ := time.Parse(time.RFC3339, tt.want["time"])
				tt.want["time"] = parsed.UTC().Format("2006-01-02T15:04:05.000Z")
			}

			if !reflect.DeepEqual(recordCopy, tt.want) {
				t.Errorf("TimestampFormat.Convert() result = %v, want %v", recordCopy, tt.want)
			}
		})
	}
}

func TestTimestampFormat_Convert_TimeZones(t *testing.T) {
	tests := []struct {
		name      string
		timestamp *TimestampFormat
		input     string
		want      string
	}{
		{
			name: "UTC timezone",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
			},
			input: "2023-01-01T12:00:00Z",
			want:  "2023-01-01T12:00:00.000Z",
		},
		{
			name: "Positive timezone offset",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
			},
			input: "2023-01-01T17:00:00+05:00",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
		{
			name: "Negative timezone offset",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "rfc3339",
			},
			input: "2023-01-01T07:00:00-05:00",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
		{
			name: "Custom format with timezone",
			timestamp: &TimestampFormat{
				Source: "timestamp",
				Format: "2006-01-02 15:04:05 -0700",
			},
			input: "2023-01-01 07:00:00 -0500",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize timestamp
			tt.timestamp.Init()

			// Create test record
			record := map[string]string{"timestamp": tt.input}

			// Convert the timestamp
			err := tt.timestamp.Convert(record)
			if err != nil {
				t.Errorf("TimestampFormat.Convert() unexpected error: %v", err)
				return
			}

			// Normalize times to UTC for comparison
			parsed, _ := time.Parse(time.RFC3339Nano, record["time"])
			got := parsed.UTC().Format("2006-01-02T15:04:05.000Z")

			if got != tt.want {
				t.Errorf("TimestampFormat.Convert() time = %v, want %v", got, tt.want)
			}
		})
	}
}
