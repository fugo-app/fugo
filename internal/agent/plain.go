package agent

import (
	"fmt"
	"maps"
	"regexp"
)

// plainParser extracts fields from a plain-text log line using the configured regex pattern
type plainParser struct {
	re *regexp.Regexp
}

func newPlainParser(pattern string) (*plainParser, error) {
	if re, err := regexp.Compile(pattern); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	} else {
		return &plainParser{re}, nil
	}
}

func (p *plainParser) Parse(line string, data map[string]string) (map[string]string, error) {
	match := p.re.FindStringSubmatch(line)
	if match == nil {
		return nil, nil
	}

	// Extract named capture groups
	result := make(map[string]string)
	for i, name := range p.re.SubexpNames() {
		if i == 0 || name == "" {
			continue // Skip the full match and unnamed groups
		}

		result[name] = match[i]
	}

	// If no named groups were matched, return the log line as a message
	if len(result) == 0 {
		return nil, nil
	}

	// Copy existing data into the result
	maps.Copy(result, data)

	return result, nil
}
