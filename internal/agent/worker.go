package agent

type FileWorker struct {
	path string
	data map[string]string
}

func NewFileWorker(path string, data map[string]string) *FileWorker {
	return &FileWorker{path, data}
}

func (fw *FileWorker) Handle() {
	// TODO: implement file handling logic
}
