package duration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Parse(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			input: "1s",
			want:  time.Second,
		},
		{
			input: "2m",
			want:  2 * time.Minute,
		},
		{
			input: "3h",
			want:  3 * time.Hour,
		},
		{
			input: "1d",
			want:  24 * time.Hour,
		},
		{
			input: "1h30m",
			want:  time.Hour + (30 * time.Minute),
		},
		{
			input: "2D3H",
			want:  (24 + 24 + 3) * time.Hour,
		},
		{
			input: "1d 14h 30m",
			want:  ((24 + 14) * time.Hour) + (30 * time.Minute),
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   "10x",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := Parse(test.input)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.want, result)
			}
		})
	}
}

func Test_Match(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1s", true},
		{"2m", true},
		{"3h", true},
		{"1d", true},
		{"1h30m", true},
		{"2d3h", true},
		{"1d 14h 30m", false},
		{"invalid", false},
		{"10x", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Match(test.input)
			assert.Equal(t, test.want, result)
		})
	}
}
