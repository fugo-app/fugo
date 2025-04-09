package file

import (
	"encoding/json"
	"fmt"
)

// jsonParser extracts fields from a JSON log line
type jsonParser struct {
}

func newJsonParser() (*jsonParser, error) {
	return &jsonParser{}, nil
}

func (j *jsonParser) Parse(line string) (map[string]string, error) {
	var raw map[string]interface{}

	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON log line: %w", err)
	}

	if len(raw) == 0 {
		return nil, nil
	}

	// Convert types to strings
	result := make(map[string]string)
	for key, val := range raw {
		result[key] = fmt.Sprintf("%v", val)
	}

	return result, nil
}
