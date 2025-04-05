package agent

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

type FileWorker struct {
	path string
	data map[string]string

	stop chan struct{}
}

func NewFileWorker(path string, data map[string]string) (*FileWorker, error) {
	return &FileWorker{
		path: path,
		data: data,
		stop: make(chan struct{}),
	}, nil
}

func (fw *FileWorker) Start() {
	go fw.watch()
}

func (fw *FileWorker) Stop() {
	close(fw.stop)
}

func (fw *FileWorker) Handle() {
	// TODO: push to queue
}

func (fw *FileWorker) watch() {
	// TODO: debounce
	<-fw.stop
}

func (fw *FileWorker) tail() {
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

	offset := GetOffset(fw.path)

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
	SetOffset(fw.path, offset)
}
