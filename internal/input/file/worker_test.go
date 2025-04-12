package file

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockParser struct {
	mu    sync.Mutex
	calls int
}

func (p *mockParser) Parse(text string) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++

	return map[string]string{"line": text}, nil
}

type mockProcessor struct {
	mu        sync.Mutex
	processed []map[string]string
}

func (p *mockProcessor) Process(data map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processed = append(p.processed, data)
}

func TestFileWorker_tail(t *testing.T) {
	// Create mocks
	mockParser := &mockParser{}
	mockProcessor := &mockProcessor{}

	// Create a temporary directory and file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.log")

	// Config for file-based input
	globalFileConfig := &FileConfig{
		Offsets: filepath.Join(tempDir, "offsets.yaml"),
	}
	globalFileConfig.InitDefault(tempDir)
	require.NoError(t, globalFileConfig.Open(), "failed to open file config")
	defer globalFileConfig.Close()

	// Write test data to the file
	testData := "line1\nline2\nline3\n"
	{
		err := os.WriteFile(tempFile, []byte(testData), 0644)
		require.NoError(t, err, "Failed to write test file")
	}

	// Create a file worker
	worker, err := newFileWorker(
		tempFile,
		map[string]string{
			"source": "test",
		},
		mockParser,
		nil,
		mockProcessor,
	)
	require.NoError(t, err, "Failed to create file worker")

	// Call tail() to process the file
	worker.tail()

	// Verify that each line was processed
	require.Equal(t, 3, mockParser.calls, "Parser should be called 3 times")
	require.Len(t, mockProcessor.processed, 3, "Processor should process 3 lines")

	// Verify the content of processed data
	expected := []map[string]string{
		{"line": "line1", "source": "test"},
		{"line": "line2", "source": "test"},
		{"line": "line3", "source": "test"},
	}
	require.Equal(t, expected, mockProcessor.processed, "Processed data doesn't match")

	// Verify offset was updated
	require.Equal(t, int64(len(testData)), getOffset(tempFile), "Offset should be updated")

	// Test that tail() only processes new content on subsequent calls
	{
		data := "line4\nline5\n"
		f, err := os.OpenFile(tempFile, os.O_APPEND|os.O_WRONLY, 0644)
		require.NoError(t, err, "Failed to open file for appending")
		_, err = f.WriteString(data)
		require.NoError(t, err, "Failed to append to file")
		f.Close()
	}

	// Reset mock processor data for clarity
	mockProcessor.mu.Lock()
	mockProcessor.processed = []map[string]string{}
	mockProcessor.mu.Unlock()

	// Call tail() again
	worker.tail()

	// Verify that only new lines were processed
	require.Equal(t, 5, mockParser.calls, "Parser should be called 2 more times")
	require.Len(t, mockProcessor.processed, 2, "Processor should process 2 new lines")

	expected = []map[string]string{
		{"line": "line4", "source": "test"},
		{"line": "line5", "source": "test"},
	}
	require.Equal(t, expected, mockProcessor.processed, "Processed data doesn't match")

	// Test file truncation case
	{
		data := "truncated\n"
		f, err := os.OpenFile(tempFile, os.O_TRUNC|os.O_WRONLY, 0644)
		require.NoError(t, err, "Failed to open file for truncating")
		_, err = f.WriteString(data)
		require.NoError(t, err, "Failed to write to file")
		f.Close()
	}

	// Reset mock processor data again
	mockProcessor.mu.Lock()
	mockProcessor.processed = []map[string]string{}
	mockProcessor.mu.Unlock()

	// Call tail() again
	worker.tail()

	// Verify truncation was handled correctly
	require.Equal(t, 6, mockParser.calls, "Parser should be called once more")
	require.Len(t, mockProcessor.processed, 1, "Processor should process 1 line from truncated file")

	expected = []map[string]string{
		{"line": "truncated", "source": "test"},
	}
	require.Equal(t, expected, mockProcessor.processed, "Processed data doesn't match")
}
