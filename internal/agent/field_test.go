package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestField_Process(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		data    map[string]string
		want    any
		wantErr bool
	}{
		{
			name: "process time field",
			field: Field{
				Name:       "time",
				TimeFormat: "2006-01-02 15:04:05",
			},
			data: map[string]string{
				"time": "2023-01-01 12:00:00",
			},
			want:    int64(1672574400000),
			wantErr: false,
		},
		{
			name: "process time field with source",
			field: Field{
				Name:       "time",
				Source:     "timestamp",
				TimeFormat: "2006-01-02 15:04:05",
			},
			data: map[string]string{
				"timestamp": "2023-01-01 12:00:00",
			},
			want:    int64(1672574400000),
			wantErr: false,
		},
		{
			name: "process time field with invalid timestamp",
			field: Field{
				Name:       "time",
				TimeFormat: "2006-01-02 15:04:05",
			},
			data: map[string]string{
				"time": "invalid-timestamp",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "process field with template",
			field: Field{
				Name:     "formatted",
				Template: "{{.level}}: {{.message}}",
			},
			data: map[string]string{
				"level":   "INFO",
				"message": "Test message",
			},
			want:    "INFO: Test message",
			wantErr: false,
		},
		{
			name: "process string field",
			field: Field{
				Name: "message",
			},
			data: map[string]string{
				"message": "Test message",
			},
			want:    "Test message",
			wantErr: false,
		},
		{
			name: "process string field with source",
			field: Field{
				Name:   "msg",
				Source: "message",
			},
			data: map[string]string{
				"message": "Test message",
			},
			want:    "Test message",
			wantErr: false,
		},
		{
			name: "process int field",
			field: Field{
				Name: "count",
				Type: "int",
			},
			data: map[string]string{
				"count": "123",
			},
			want:    int64(123),
			wantErr: false,
		},
		{
			name: "process float field",
			field: Field{
				Name: "value",
				Type: "float",
			},
			data: map[string]string{
				"value": "123.456",
			},
			want:    float64(123.456),
			wantErr: false,
		},
		{
			name: "process missing field",
			field: Field{
				Name: "missing",
			},
			data:    map[string]string{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "process invalid int",
			field: Field{
				Name: "count",
				Type: "int",
			},
			data: map[string]string{
				"count": "not-a-number",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "process invalid float",
			field: Field{
				Name: "value",
				Type: "float",
			},
			data: map[string]string{
				"value": "not-a-number",
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.field.Init(), "Failed to initialize Field")

			got, err := tt.field.Process(tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
