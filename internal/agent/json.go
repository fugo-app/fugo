package agent

import (
	"encoding/json"
	"fmt"
	"maps"
)

// jsonParser extracts fields from a JSON log line
type jsonParser struct {
}

func newJsonParser() (*jsonParser, error) {
	return &jsonParser{}, nil
}

func (j *jsonParser) Parse(line string, data map[string]string) (map[string]string, error) {
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

	// Copy existing data into the result
	maps.Copy(result, data)

	return result, nil
}
