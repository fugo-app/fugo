package duration

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var reMatch = regexp.MustCompile(`(?i)^(\d+[smhd]?)+$`)
var reParts = regexp.MustCompile(`(?i)(\d+)([smhd]|$)`)

var unitMap = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': time.Hour * 24,
}

func Match(input string) bool {
	return reMatch.MatchString(input)
}

func Parse(input string) (time.Duration, error) {
	matches := reParts.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration: %s", input)
	}

	var total time.Duration

	for _, match := range matches {
		if len(match) != 3 {
			return 0, fmt.Errorf("invalid duration format: %s", input)
		}

		value, _ := strconv.Atoi(match[1])

		unit := time.Second
		if match[2] != "" {
			key := match[2][0]
			if key >= 'A' && key <= 'Z' {
				key += 'a' - 'A' // Convert to lowercase
			}
			unit = unitMap[key]
		}

		total += time.Duration(value) * unit
	}

	return total, nil
}
