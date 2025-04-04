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
		want    map[string]any
		wantErr bool
	}{
		{
			name: "parse empty log line",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `(?P<message>.*)`,
				Fields: []Field{
					{
						Name: "message",
					},
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
				Regex:  `^(?P<time>[^ ]+ [^ ]+) (?P<level>\w+) (?P<message>.*)`,
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
					{
						Name: "message",
					},
				},
			},
			line: "2023-01-01 12:00:00 INFO Test message",
			want: map[string]any{
				"time":    int64(1672574400000),
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
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
					{
						Name: "message",
					},
				},
			},
			line:    "Test message",
			want:    nil,
			wantErr: false,
		},
		{
			name: "parse plain log with partial mathching regex",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `^(?P<time>[^ ]+ [^ ]+) (?P<level>\w+)`,
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
				},
			},
			line: "2023-01-01 12:00:00 INFO Test message",
			want: map[string]any{
				"time":  int64(1672574400000),
				"level": "INFO",
			},
			wantErr: false,
		},
		{
			name: "parse with complex log format",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `\[(?P<timestamp>[^\]]+)\] \[(?P<level>[^\]]+)\] \[(?P<module>[^\]]+)\] (?P<message>.*)`,
				Fields: []Field{
					{
						Name:       "time",
						Source:     "timestamp",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
					{
						Name: "module",
					},
					{
						Name: "message",
					},
				},
			},
			line: "[2023-01-01 12:00:00] [INFO] [auth] User login successful",
			want: map[string]any{
				"time":    int64(1672574400000),
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
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
					{
						Name: "message",
					},
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
			want: map[string]any{
				"time":    int64(1672574400000),
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
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "level",
					},
					{
						Name: "message",
					},
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message","other":"should-not-be-included"}`,
			want: map[string]any{
				"time":    int64(1672574400000),
				"level":   "INFO",
				"message": "Test message",
			},
			wantErr: false,
		},
		{
			name: "parse valid JSON without type conversion",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "int",
					},
					{
						Name: "float",
					},
					{
						Name: "bool",
					},
				},
			},
			line: `{"time":"2023-01-01 12:00:00","int":123,"float":123.456,"bool":true}`,
			want: map[string]any{
				"time":  int64(1672574400000),
				"int":   "123",
				"float": "123.456",
				"bool":  "true",
			},
			wantErr: false,
		},
		{
			name: "parse valid JSON with type conversion",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name: "int",
						Type: "int",
					},
					{
						Name: "float",
						Type: "float",
					},
				},
			},
			line: `{"time":"2023-01-01 12:00:00","int":123,"float":123.456}`,
			want: map[string]any{
				"time":  int64(1672574400000),
				"int":   int64(123),
				"float": float64(123.456),
			},
			wantErr: false,
		},
		{
			name: "template and exclude",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Fields: []Field{
					{
						Name:       "time",
						TimeFormat: "2006-01-02 15:04:05",
					},
					{
						Name:     "formatted",
						Template: "{{.level}}: {{.message}}",
					},
				},
			},
			line: `{"time":"2023-01-01 12:00:00","level":"INFO","message":"Test message"}`,
			want: map[string]any{
				"time":      int64(1672574400000),
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
				require.Equal(t, tt.want, got, "Map not equal", tt.name)
			}
		})
	}
}
