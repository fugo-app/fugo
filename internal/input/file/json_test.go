package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "parse json log",
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
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
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := newJsonParser()
			require.NoError(t, err, "Failed to initialize FileAgent")
			got, err := parser.Parse(tt.line)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
				require.Equal(t, tt.want, got, "Map not equal", tt.name)
			}
		})
	}
}
