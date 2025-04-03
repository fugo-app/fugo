package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileAgent_Parse(t *testing.T) {
	tests := []struct {
		name    string
		agent   *FileAgent
		line    string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "parse empty log line",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `(?P<message>.*)`,
				Fields: map[string]string{
					"message": "",
				},
			},
			line:    "",
			want:    nil,
			wantErr: false,
		},
		{
			name: "parse plain log",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `(?P<time>[^ ]+ [^ ]+) (?P<level>\w+) (?P<message>.*)`,
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"level":   "",
					"message": "",
				},
			},
			line: "2023-01-01 12:00:00 INFO Test message",
			want: map[string]string{
				"time":    "2023-01-01T12:00:00.000Z",
				"level":   "INFO",
				"message": "Test message",
			},
			wantErr: false,
		},
		{
			name: "parse plain log with non-matching regex",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `(?P<time>[^ ]+ [^ ]+) (?P<level>\w+) (?P<message>.*)`,
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"level":   "",
					"message": "",
				},
			},
			line:    "Test message",
			want:    nil,
			wantErr: false,
		},
		{
			name: "parse with complex log format",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `\[(?P<timestamp>[^\]]+)\] \[(?P<level>[^\]]+)\] \[(?P<module>[^\]]+)\] (?P<message>.*)`,
				Timestamp: &TimestampFormat{
					Source: "timestamp",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"level":   "",
					"module":  "",
					"message": "",
				},
			},
			line: "[2023-01-01 12:00:00] [INFO] [auth] User login successful",
			want: map[string]string{
				"time":    "2023-01-01T12:00:00.000Z",
				"level":   "INFO",
				"module":  "auth",
				"message": "User login successful",
			},
			wantErr: false,
		},
		{
			name: "parse json log",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"level":   "",
					"message": "",
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
			want: map[string]string{
				"time":    "2023-01-01T12:00:00.000Z",
				"level":   "INFO",
				"message": "Test message",
			},
			wantErr: false,
		},
		{
			name: "parse valid JSON with specific fields",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"level":   "",
					"message": "",
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message","other":"should-not-be-included"}`,
			want: map[string]string{
				"time":    "2023-01-01T12:00:00.000Z",
				"level":   "INFO",
				"message": "Test message",
			},
			wantErr: false,
		},
		{
			name: "parse valid JSON with type conversion",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"int":   "",
					"float": "",
					"bool":  "",
				},
			},
			line: `{"time":"2023-01-01 12:00:00","int":123,"float":123.456,"bool":true}`,
			want: map[string]string{
				"time":  "2023-01-01T12:00:00.000Z",
				"int":   "123",
				"float": "123.456",
				"bool":  "true",
			},
			wantErr: false,
		},
		{
			name: "template and exclude",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Fields: map[string]string{
					"formatted": "{{.level}}: {{.message}}",
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
			want: map[string]string{
				"time":      "2023-01-01T12:00:00.000Z",
				"formatted": "INFO: Test message",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.agent.Init(), "Failed to initialize FileAgent")
			got, err := tt.agent.Parse(tt.line)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
			}
			require.Equal(t, tt.want, got, "Map not equal", tt.name)
		})
	}
}
