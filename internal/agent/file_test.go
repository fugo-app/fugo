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
			},
			line:    "Test message",
			want:    nil,
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
				Path:    "/var/log/test.log",
				Format:  "json",
				Include: []string{"level", "message"},
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
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
			name: "template and exclude",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
				Timestamp: &TimestampFormat{
					Source: "time",
					Format: "2006-01-02 15:04:05",
				},
				Templates: map[string]string{
					"formatted": "{{.level}}: {{.message}}",
				},
				Exclude: []string{"level", "message"},
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

func TestFileAgent_parseJson(t *testing.T) {
	tests := []struct {
		name    string
		agent   *FileAgent
		line    string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "parse simple JSON",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
			},
			line:    `{"key":"value"}`,
			want:    map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name: "parse JSON with numeric values",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "json",
			},
			line: `{"int":123,"float":123.456,"bool":true}`,
			want: map[string]string{
				"int":   "123",
				"float": "123.456",
				"bool":  "true",
			},
			wantErr: false,
		},
		{
			name: "parse JSON with only requested fields",
			agent: &FileAgent{
				Path:    "/var/log/test.log",
				Format:  "json",
				Include: []string{"a", "c"},
			},
			line: `{"a":1,"b":2,"c":3,"d":4}`,
			want: map[string]string{
				"a": "1",
				"c": "3",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.agent.Init(), "Failed to initialize FileAgent")
			got, err := tt.agent.parseJSON(tt.line)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
			}
			require.Equal(t, tt.want, got, "Map not equal", tt.name)
		})
	}
}

func TestFileAgent_parsePlain(t *testing.T) {
	tests := []struct {
		name    string
		agent   *FileAgent
		line    string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "parse with simple regex",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `(?P<word>\w+) (?P<rest>.*)`,
			},
			line: "first second third",
			want: map[string]string{
				"word": "first",
				"rest": "second third",
			},
			wantErr: false,
		},
		{
			name: "parse with complex log format",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `\[(?P<timestamp>[^\]]+)\] \[(?P<level>[^\]]+)\] \[(?P<module>[^\]]+)\] (?P<message>.*)`,
			},
			line: "[2023-01-01 12:00:00] [INFO] [auth] User login successful",
			want: map[string]string{
				"timestamp": "2023-01-01T12:00:00.000Z",
				"level":     "INFO",
				"module":    "auth",
				"message":   "User login successful",
			},
			wantErr: false,
		},
		{
			name: "parse with regex that doesn't match",
			agent: &FileAgent{
				Path:   "/var/log/test.log",
				Format: "plain",
				Regex:  `\[(?P<timestamp>[^\]]+)\] \[(?P<level>[^\]]+)\]`,
			},
			line:    "This doesn't match the regex pattern",
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.agent.Init(), "Failed to initialize FileAgent")
			got, err := tt.agent.parsePlain(tt.line)
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
			}
			require.Equal(t, tt.want, got, "Map not equal", tt.name)
		})
	}
}
