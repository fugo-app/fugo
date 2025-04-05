package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		data    map[string]string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "parse json log",
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
			data: nil,
			want: map[string]string{
				"time":    "2023-01-01 12:00:00",
				"level":   "INFO",
				"message": "Test message",
			},
			wantErr: false,
		},
		{
			name: "type conversion",
			line: `{"time":"2023-01-01 12:00:00","int":123,"float":123.456,"bool":true}`,
			data: nil,
			want: map[string]string{
				"time":  "2023-01-01 12:00:00",
				"int":   "123",
				"float": "123.456",
				"bool":  "true",
			},
			wantErr: false,
		},
		{
			name:    "non-matching json",
			line:    `plain text log`,
			data:    nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "join external data",
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message","host":"invalid-host"}`,
			data: map[string]string{
				"source": "test_source",
				"host":   "test_host",
			},
			want: map[string]string{
				"time":    "2023-01-01 12:00:00",
				"level":   "INFO",
				"message": "Test message",
				"source":  "test_source",
				"host":    "test_host",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := newJsonParser()
			require.NoError(t, err, "Failed to initialize FileAgent")
			got, err := parser.Parse(tt.line, tt.data)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
				require.Equal(t, tt.want, got, "Map not equal", tt.name)
			}
		})
	}
}
