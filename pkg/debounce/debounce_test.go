package debounce

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDebounce_withoutImmediate(t *testing.T) {
	var counter atomic.Int32
	increment := func() {
		counter.Add(1)
	}

	d := NewDebounce(increment, 50*time.Millisecond, false)
	d.Start()
	defer d.Stop()

	// Initial state
	require.Equal(t, int32(0), counter.Load(), "Counter should be 0 initially")

	// Emit multiple times within delay window
	d.Emit()
	d.Emit()
	d.Emit()

	// Nothing should happen immediately
	require.Equal(t, int32(0), counter.Load(), "Counter should still be 0 before delay passes")

	// Wait for debounce
	time.Sleep(60 * time.Millisecond)

	// Counter should be incremented once
	require.Equal(t, int32(1), counter.Load(), "Counter should be 1 after delay passes")
}

func TestDebounce_withImmediate(t *testing.T) {
	var counter atomic.Int32
	increment := func() {
		counter.Add(1)
	}

	d := NewDebounce(increment, 50*time.Millisecond, true)
	d.Start()
	defer d.Stop()

	// Wait a bit for the immediate execution to happen
	time.Sleep(10 * time.Millisecond)

	// Should have executed immediately upon start
	require.Equal(t, int32(1), counter.Load(), "Counter should be 1")

	// Emit multiple times within delay window
	d.Emit()
	d.Emit()
	d.Emit()

	// Wait for debounce
	time.Sleep(60 * time.Millisecond)

	// Counter should be incremented again
	require.Equal(t, int32(2), counter.Load(), "Counter should be 2 after delay passes")
}

func TestDebounceStop(t *testing.T) {
	var counter atomic.Int32
	increment := func() {
		counter.Add(1)
	}

	d := NewDebounce(increment, 50*time.Millisecond, false)
	d.Start()

	// Emit and immediately stop
	d.Emit()
	d.Stop()

	// Wait longer than the delay
	time.Sleep(60 * time.Millisecond)

	// Counter should not have incremented due to the stop
	require.Equal(t, int32(0), counter.Load(), "Counter should still be 0 after stop")

	// Stop should be idempotent
	d.Stop() // Should not panic
}
