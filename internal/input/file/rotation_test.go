package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRotationConfig_Init(t *testing.T) {
	tests := []struct {
		name      string
		config    RotationConfig
		expectErr bool
		size      int64
	}{
		{
			name: "valid bytes value",
			config: RotationConfig{
				Method: "truncate",
				Size:   "100",
			},
			expectErr: false,
			size:      100,
		},
		{
			name: "valid KB value",
			config: RotationConfig{
				Method: "truncate",
				Size:   "15KB",
			},
			expectErr: false,
			size:      15 * 1024,
		},
		{
			name: "valid MB value",
			config: RotationConfig{
				Method: "truncate",
				Size:   "2MB",
			},
			expectErr: false,
			size:      2 * 1024 * 1024,
		},
		{
			name: "case insensitive unit",
			config: RotationConfig{
				Method: "truncate",
				Size:   "15kb",
			},
			expectErr: false,
			size:      15 * 1024,
		},
		{
			name: "invalid size format",
			config: RotationConfig{
				Method: "truncate",
				Size:   "invalid",
			},
			expectErr: true,
		},
		{
			name: "invalid number value",
			config: RotationConfig{
				Method: "truncate",
				Size:   "1.5KB",
			},
			expectErr: true,
		},
		{
			name: "missing method",
			config: RotationConfig{
				Size: "100",
			},
			expectErr: true,
		},
		{
			name: "unsupported method",
			config: RotationConfig{
				Method: "unsupported",
				Size:   "100",
			},
			expectErr: true,
		},
		{
			name: "rename method without run",
			config: RotationConfig{
				Method: "rename",
				Size:   "100",
				Run:    "true",
			},
			expectErr: false,
			size:      100,
		},
		{
			name: "rename method without run",
			config: RotationConfig{
				Method: "rename",
				Size:   "100",
			},
			expectErr: true,
			size:      100,
		},
		{
			name: "case insensitive method",
			config: RotationConfig{
				Method: "TRUNCATE",
				Size:   "100",
			},
			expectErr: false,
			size:      100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Init()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.size, tt.config.size)

				// Check correct rotator type was assigned
				if tt.config.Method != "" {
					switch strings.ToLower(tt.config.Method) {
					case "truncate":
						_, ok := tt.config.rotator.(*truncateFile)
						require.True(t, ok, "expected truncateFile rotator")
					case "rename":
						_, ok := tt.config.rotator.(*renameFile)
						require.True(t, ok, "expected renameFile rotator")
					}
				}
			}
		})
	}
}

func TestRotationConfig_CheckSize(t *testing.T) {
	tests := []struct {
		name     string
		config   *RotationConfig
		size     int64
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			size:     100,
			expected: false,
		},
		{
			name: "size less than max",
			config: &RotationConfig{
				size: 100,
			},
			size:     50,
			expected: false,
		},
		{
			name: "size equal to max",
			config: &RotationConfig{
				size: 100,
			},
			size:     100,
			expected: true,
		},
		{
			name: "size greater than max",
			config: &RotationConfig{
				size: 100,
			},
			size:     150,
			expected: true,
		},
		{
			name: "size zero",
			config: &RotationConfig{
				size: 0,
			},
			size:     100,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.CheckSize(tt.size)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRotationConfig_truncateFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "truncate_test.txt")

	// Create file with content
	content := "This is test content\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Check file size
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))

	// Test truncation
	rotator := &truncateFile{}
	require.True(t, rotator.CheckSize(info.Size()))

	require.NoError(t, rotator.Rotate(testFile))

	// Verify file was truncated
	info, err = os.Stat(testFile)
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())
}

func TestRotationConfig_renameFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "rename_test.txt")
	tempFile := testFile + ".remove"

	// Create file with content and set permissions
	content := "This is test content\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Get original file info
	originalInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Greater(t, originalInfo.Size(), int64(0))

	// Test rename rotation
	rotator := &renameFile{}
	require.True(t, rotator.CheckSize(originalInfo.Size()))

	err = rotator.Rotate(testFile)
	require.NoError(t, err)

	// Verify new file was created and is empty
	newInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Equal(t, int64(0), newInfo.Size())
	require.Equal(t, originalInfo.Mode().Perm(), newInfo.Mode().Perm())

	// Wait a bit for the goroutine to run
	time.Sleep(100 * time.Millisecond)

	// Check if the temp file was removed
	_, err = os.Stat(tempFile)
	require.True(t, os.IsNotExist(err), "temp file should be removed")
}

func TestRotationConfig_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "rotation_test.txt")

	outputContent := "Processed file\n"
	outputFile := filepath.Join(tempDir, "script_output.txt")

	// Create test file
	content := "This is test content\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Configure rotation with a shell script to run
	config := &RotationConfig{
		Method: "truncate",
		Size:   "10",
		Run:    `printf "` + outputContent + `" > ` + outputFile,
	}
	err = config.Init()
	require.NoError(t, err)

	// Perform rotation
	require.True(t, config.CheckSize(21))
	err = config.Rotate(testFile)
	require.NoError(t, err)

	// Verify file was truncated
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Equal(t, int64(0), info.Size())

	// Wait for the script to execute
	time.Sleep(100 * time.Millisecond)

	// Check script output file
	result, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Script output file should exist")
	require.Equal(t, outputContent, string(result), "Output content should match")
}
