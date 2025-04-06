package field

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
		want      int64
		wantErr   bool
	}{
		{
			name: "rfc3339 format",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input:   "2023-01-01T12:00:00Z",
			want:    1672574400000,
			wantErr: false,
		},
		{
			name: "common format",
			timestamp: &TimestampFormat{
				Format: "common",
			},
			input:   "10/Oct/2000:13:55:36 -0700",
			want:    971211336000,
			wantErr: false,
		},
		{
			name: "invalid timestamp format",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input:   "invalid-date",
			want:    0,
			wantErr: true,
		},
		{
			name: "unix timestamp integer",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "1672574400",
			want:    1672574400000,
			wantErr: false,
		},
		{
			name: "unix timestamp with milliseconds",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "1672574400.123",
			want:    1672574400123,
			wantErr: false,
		},
		{
			name: "invalid unix timestamp",
			timestamp: &TimestampFormat{
				Format: "unix",
			},
			input:   "not-a-number",
			want:    0,
			wantErr: true,
		},
		{
			name: "custom format",
			timestamp: &TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
			input:   "2023-01-01 12:00:00",
			want:    1672574400000,
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
		want      int64
	}{
		{
			name: "UTC timezone",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input: "2023-01-01T12:00:00Z",
			want:  1672574400000,
		},
		{
			name: "Positive timezone offset",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input: "2023-01-01T17:00:00+05:00",
			want:  1672574400000, // UTC equivalent
		},
		{
			name: "Negative timezone offset",
			timestamp: &TimestampFormat{
				Format: "rfc3339",
			},
			input: "2023-01-01T07:00:00-05:00",
			want:  1672574400000, // UTC equivalent
		},
		{
			name: "Custom format with timezone",
			timestamp: &TimestampFormat{
				Format: "2006-01-02 15:04:05 -0700",
			},
			input: "2023-01-01 07:00:00 -0500",
			want:  1672574400000, // UTC equivalent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize timestamp
			require.NoError(t, tt.timestamp.Init(), "Failed to initialize timestamp format")

			// Convert the timestamp
			result, err := tt.timestamp.Convert(tt.input)
			require.NoError(t, err, "Failed to convert timestamp")

			require.Equal(t, tt.want, result, "Unexpected result for timestamp conversion")
		})
	}
}

func TestTimestampFormat_Convert_WithoutYear(t *testing.T) {
	tests := []struct {
		name      string
		timestamp *TimestampFormat
		input     string
		now       time.Time
		want      int64
	}{
		{
			name: "same year",
			timestamp: &TimestampFormat{
				Format: "Jan 02 15:04:05",
			},
			input: "Jun 15 12:00:00",
			now:   time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
			want:  time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC).UnixMilli(),
		},
		{
			name: "earlier month same year",
			timestamp: &TimestampFormat{
				Format: "Jan 02 15:04:05",
			},
			input: "Mar 10 12:00:00",
			now:   time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
			want:  time.Date(2023, 3, 10, 12, 0, 0, 0, time.UTC).UnixMilli(),
		},
		{
			name: "later month same year",
			timestamp: &TimestampFormat{
				Format: "Jan 02 15:04:05",
			},
			input: "Oct 20 12:00:00",
			now:   time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
			want:  time.Date(2023, 10, 20, 12, 0, 0, 0, time.UTC).UnixMilli(),
		},
		{
			name: "New Year edge case",
			timestamp: &TimestampFormat{
				Format: "Jan 02 15:04:05",
			},
			input: "Dec 31 23:59:59",
			now:   time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			want:  time.Date(2022, 12, 31, 23, 59, 59, 0, time.UTC).UnixMilli(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize timestamp
			require.NoError(t, tt.timestamp.Init(), "Failed to initialize timestamp format")

			// Convert the timestamp
			stdTimeNow = func() time.Time {
				return tt.now
			}
			defer func() {
				stdTimeNow = time.Now
			}()
			result, err := tt.timestamp.Convert(tt.input)
			require.NoError(t, err, "Failed to convert timestamp")

			require.Equal(t, tt.want, result, "Unexpected result for timestamp conversion")
		})
	}
}
