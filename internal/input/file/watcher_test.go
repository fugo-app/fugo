package file

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dummyProcessor struct{}

func (d *dummyProcessor) Process(data map[string]string) {}
func (d *dummyProcessor) Write(data map[string]any)      {}

func TestFileWatcher_WorkerManagement(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Config for file-based input
	globalFileConfig := &FileConfig{
		Offsets: filepath.Join(tempDir, "offsets.yaml"),
	}
	globalFileConfig.InitDefault(tempDir)
	require.NoError(t, globalFileConfig.Open(), "failed to open file config")
	defer globalFileConfig.Close()

	var ok bool

	// Create a watcher instance
	watcher := &FileWatcher{
		Path:   filepath.Join(tempDir, "(?P<host>.*)\\.log"),
		Format: "plain",
		Regex:  `(?P<message>.*)`,
	}
	processor := &dummyProcessor{}
	err := watcher.Init(processor)
	require.NoError(t, err, "failed to create file watcher")

	// Start the watcher with a pattern that will match files with .log extension
	watcher.Start()
	defer watcher.Stop()

	// Wait a bit for the watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Initially there should be no workers
	require.Empty(t, watcher.workers, "expected no workers initially")

	// Create a new file that matches the pattern
	testFile := filepath.Join(tempDir, "test.log")
	err = os.WriteFile(testFile, []byte("test data"), 0644)
	require.NoError(t, err, "failed to create test file")

	// Wait for the watcher to detect the new file
	time.Sleep(200 * time.Millisecond)

	// There should be a worker for the new file
	require.Len(t, watcher.workers, 1, "expected 1 worker after file creation")

	// Check if the worker for the specific file exists
	_, ok = watcher.workers["test.log"]
	require.True(t, ok, "worker for test.log not found")

	// Remove the file
	err = os.Remove(testFile)
	require.NoError(t, err, "failed to remove test file")

	// Wait for the watcher to detect the removal
	time.Sleep(200 * time.Millisecond)

	// The worker should be removed
	require.Empty(t, watcher.workers, "expected no workers after file removal")
}

func TestFileWatcher_MultipleWorkers(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Config for file-based input
	globalFileConfig := &FileConfig{
		Offsets: filepath.Join(tempDir, "offsets.yaml"),
	}
	globalFileConfig.InitDefault(tempDir)
	require.NoError(t, globalFileConfig.Open(), "failed to open file config")
	defer globalFileConfig.Close()

	var ok bool

	// Create a watcher instance
	watcher := &FileWatcher{
		Path:   filepath.Join(tempDir, "(?P<host>.*)\\.log"),
		Format: "plain",
		Regex:  `(?P<message>.*)`,
	}
	processor := &dummyProcessor{}
	err := watcher.Init(processor)
	require.NoError(t, err, "failed to create file watcher")

	// Start the watcher with a pattern that will match files with .log extension
	watcher.Start()
	defer watcher.Stop()

	// Wait a bit for the watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// Create multiple log files
	testFiles := []string{
		filepath.Join(tempDir, "app1.log"),
		filepath.Join(tempDir, "app2.log"),
		filepath.Join(tempDir, "app3.log"),
	}

	for _, file := range testFiles {
		err = os.WriteFile(file, []byte("test data"), 0644)
		require.NoError(t, err, "failed to create test file")
	}

	// Wait for the watcher to detect all files
	time.Sleep(200 * time.Millisecond)

	// There should be workers for all the files
	require.Len(t, watcher.workers, len(testFiles), "unexpected workers quantity")

	// Check if each specific worker exists
	for _, file := range testFiles {
		basename := filepath.Base(file)
		_, ok = watcher.workers[basename]
		require.True(t, ok, "worker for %s not found", basename)
	}

	// Remove one file
	err = os.Remove(testFiles[1])
	require.NoError(t, err, "failed to remove test file")

	// Wait for the watcher to detect the removal
	time.Sleep(200 * time.Millisecond)

	// One worker should be removed
	require.Len(t, watcher.workers, len(testFiles)-1, "unexpected workers quantity after removal")

	// The specific worker should be removed
	_, ok = watcher.workers[filepath.Base(testFiles[1])]
	require.False(t, ok, "worker for removed file still exists")

	// Rename file
	err = os.Rename(testFiles[0], testFiles[0]+".1")
	require.NoError(t, err, "failed to rename test file")

	// Wait for the watcher to detect the renaming
	time.Sleep(200 * time.Millisecond)

	// Another one worker should be removed
	require.Len(t, watcher.workers, len(testFiles)-2, "unexpected workers quantity after renaming")

	// The specific worker should be removed
	_, ok = watcher.workers[filepath.Base(testFiles[0])]
	require.False(t, ok, "worker for renamed file still exists")
}
