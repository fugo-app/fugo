package file

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

type fileWorker struct {
	path string
	data map[string]string

	debounce chan struct{}
	stop     chan struct{}
}

func newFileWorker(path string, data map[string]string) (*fileWorker, error) {
	return &fileWorker{
		path:     path,
		data:     data,
		debounce: make(chan struct{}, 1),
		stop:     make(chan struct{}),
	}, nil
}

func (fw *fileWorker) Start() {
	go fw.watch()
}

func (fw *fileWorker) Stop() {
	close(fw.stop)
}

// Handle pushes the task to the debouncer
func (fw *fileWorker) Handle() {
	select {
	case fw.debounce <- struct{}{}:
	default:
	}
}

func (fw *fileWorker) watch() {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	timerActive := false

	for {
		select {
		case <-fw.stop:
			if timerActive {
				timer.Stop()
			}
			return

		case <-fw.debounce:
			if !timerActive {
				timer.Reset(250 * time.Millisecond)
				timerActive = true
			}

		case <-timer.C:
			fw.tail()
			timerActive = false
		}
	}
}

func (fw *fileWorker) tail() {
	file, err := os.Open(fw.path)
	if err != nil {
		return
	}
	defer file.Close()

	// Get file info to check size
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}

	offset := getOffset(fw.path)

	// Check if file has been truncated (logrotate case)
	if offset > fileInfo.Size() {
		fmt.Printf("File %s has been truncated, resetting offset\n", fw.path)
		offset = 0
	}

	_, err = file.Seek(offset, 0)
	if err != nil {
		return
	}

	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			break
		}

		if !bytes.HasSuffix(line, []byte("\n")) {
			break
		}

		offset += int64(len(line))

		line = line[:len(line)-1]
		if bytes.HasSuffix(line, []byte("\r")) {
			line = line[:len(line)-1]
		}

		if len(line) > 0 {
			text := string(line)

			// TODO: push to parser
			fmt.Println(text)
		}

		if err == io.EOF {
			break
		}
	}

	// Update the offset for next run
	setOffset(fw.path, offset)
}
