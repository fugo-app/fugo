package file

import "sync"

var offsetMap = make(map[string]int64)
var offsetMutex = &sync.Mutex{}

// TODO: save to storage

func getOffset(path string) int64 {
	offsetMutex.Lock()
	defer offsetMutex.Unlock()

	if offset, ok := offsetMap[path]; ok {
		return offset
	}

	return 0
}

func setOffset(path string, offset int64) {
	offsetMutex.Lock()
	defer offsetMutex.Unlock()

	offsetMap[path] = offset
}
