package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimestampFormat_Init(t *testing.T) {
	tests := []struct {
		name       string
		timestamp  *TimestampFormat
		wantLayout string
	}{
		{
			name:       "default format",
			timestamp:  &TimestampFormat{},
			wantLayout: time.RFC3339,
		},
		{
			name: "rfc3339 format",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			wantLayout: time.RFC3339,
		},
		{
			name: "unix format",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			wantLayout: "unix",
		},
		{
			name: "custom format",
			timestamp: &TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
			wantLayout: "2006-01-02 15:04:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.timestamp.Init(), "Failed to initialize timestamp format")
			require.Equal(t, tt.wantLayout, tt.timestamp.layout, "Unexpected layout after initialization")
		})
	}
}

func TestTimestampFormat_Convert(t *testing.T) {
	tests := []struct {
		name      string
		timestamp *TimestampFormat
		input     string
		want      string
		wantErr   bool
	}{
		{
			name: "rfc3339 format",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input:   "2023-01-01T12:00:00Z",
			want:    "2023-01-01T12:00:00.000Z",
			wantErr: false,
		},
		{
			name: "common format",
			timestamp: &TimestampFormat{
				Format: "common",
			},
			input:   "10/Oct/2000:13:55:36 -0700",
			want:    "2000-10-10T13:55:36.000-07:00",
			wantErr: false,
		},
		{
			name: "invalid timestamp format",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input:   "invalid-date",
			want:    "invalid-date",
			wantErr: true,
		},
		{
			name: "unix timestamp integer",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "1672574400",
			want:    "2023-01-01T12:00:00.000Z",
			wantErr: false,
		},
		{
			name: "unix timestamp with milliseconds",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "1672574400.123",
			want:    "2023-01-01T12:00:00.123Z",
			wantErr: false,
		},
		{
			name: "invalid unix timestamp",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "not-a-number",
			want:    "not-a-number",
			wantErr: true,
		},
		{
			name: "custom format",
			timestamp: &TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
			input:   "2023-01-01 12:00:00",
			want:    "2023-01-01T12:00:00.000Z",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.timestamp.Init(), "Failed to initialize timestamp format")

			result, err := tt.timestamp.Convert(tt.input)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Failed to convert timestamp")
				require.Equal(t, tt.want, result, "Unexpected result for valid timestamp")
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
				Format: "rfc3339",
			},
			input: "2023-01-01T12:00:00Z",
			want:  "2023-01-01T12:00:00.000Z",
		},
		{
			name: "Positive timezone offset",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input: "2023-01-01T17:00:00+05:00",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
		{
			name: "Negative timezone offset",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input: "2023-01-01T07:00:00-05:00",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
		{
			name: "Custom format with timezone",
			timestamp: &TimestampFormat{
				Format: "2006-01-02 15:04:05 -0700",
			},
			input: "2023-01-01 07:00:00 -0500",
			want:  "2023-01-01T12:00:00.000Z", // UTC equivalent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize timestamp
			require.NoError(t, tt.timestamp.Init(), "Failed to initialize timestamp format")

			// Convert the timestamp
			result, err := tt.timestamp.Convert(tt.input)
			require.NoError(t, err, "Failed to convert timestamp")

			// Normalize times to UTC for comparison
			parsed, _ := time.Parse(time.RFC3339Nano, result)
			got := parsed.UTC().Format("2006-01-02T15:04:05.000Z")
			require.Equal(t, tt.want, got, "Unexpected result for timestamp conversion")
		})
	}
}
