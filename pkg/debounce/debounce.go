package debounce

import (
	"sync"
	"time"
)

type Debounce struct {
	fn        func()
	debounce  chan struct{}
	stop      chan struct{}
	delay     time.Duration
	immediate bool
	once      sync.Once
}

func NewDebounce(fn func(), delay time.Duration, immediate bool) *Debounce {
	return &Debounce{
		fn:        fn,
		debounce:  make(chan struct{}, 1),
		stop:      make(chan struct{}),
		delay:     delay,
		immediate: immediate,
	}
}

func (d *Debounce) Start() {
	go d.watch()
}

func (d *Debounce) Stop() {
	if d == nil {
		return
	}

	d.once.Do(func() {
		close(d.stop)
	})
}

func (d *Debounce) Emit() {
	select {
	case d.debounce <- struct{}{}:
	default:
	}
}

func (d *Debounce) watch() {
	if d.immediate {
		d.fn()
	}

	// Wait for delay after the first signal before processing
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	timerActive := false

	for {
		select {
		case <-d.stop:
			if timerActive {
				timer.Stop()
			}
			return

		case <-d.debounce:
			if !timerActive {
				timer.Reset(d.delay)
				timerActive = true
			}

		case <-timer.C:
			d.fn()
			timerActive = false
		}
	}
}
