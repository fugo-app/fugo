package file

import (
	"bytes"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getFileOffset(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		lines    int
		expected int64
	}{
		{
			name:     "empty file",
			content:  "",
			lines:    10,
			expected: 0,
		},
		{
			name:     "single line file",
			content:  "single line",
			lines:    1,
			expected: 0,
		},
		{
			name:     "single line file with newline",
			content:  "single line\n",
			lines:    1,
			expected: 0,
		},
		{
			name:     "multi line file with fewer lines than requested",
			content:  "line1\nline2\nline3\n",
			lines:    5,
			expected: 0,
		},
		{
			name:     "multi line file with exact lines",
			content:  "line1\nline2\nline3\nline4\nline5\n",
			lines:    5,
			expected: 0,
		},
		{
			name:     "multi line file with more lines than requested",
			content:  "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n",
			lines:    3,
			expected: 30, // offset after 'line5\n'
		},
		{
			name:     "check buffer",
			content:  "",
			lines:    100,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file with content
			fileName := tt.name + ".txt"
			testFile := filepath.Join(tempDir, fileName)

			if tt.name == "check buffer" {
				linesBefore := 10000
				linesTotal := linesBefore + tt.lines
				lineMinSize := int32(60)
				lineMaxSize := int32(100)

				file, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				require.NoError(t, err)
				defer file.Close()

				rand := rand.New(rand.NewPCG(0, 0))
				for i := 0; i < linesTotal; i++ {
					length := rand.Int32N(lineMaxSize-lineMinSize) + lineMinSize
					line := bytes.Repeat([]byte{'a'}, int(length))
					line[length-1] = '\n'

					_, err := file.Write(line)
					require.NoError(t, err)

					if i < linesBefore {
						tt.expected += int64(length)
					}
				}
				require.NotZero(t, tt.expected, "expected offset should not be zero")
			} else {
				err := os.WriteFile(testFile, []byte(tt.content), 0644)
				require.NoError(t, err)
			}

			// Test the function
			result := getFileOffset(testFile, tt.lines)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileOffset_NonExistentFile(t *testing.T) {
	result := getFileOffset("/non/existent/file.txt", 10)
	require.Equal(t, int64(0), result)
}
