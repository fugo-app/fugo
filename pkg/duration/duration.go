package duration

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var re = regexp.MustCompile(`(?i)(\d+)([smhd])`)

var unitMap = map[string]time.Duration{
	"s": time.Second,
	"m": time.Minute,
	"h": time.Hour,
	"d": time.Hour * 24,
}

func ParseDuration(input string) (time.Duration, error) {
	re.FindAllStringSubmatch(input, -1)
	matches := re.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration: %s", input)
	}

	var total time.Duration

	for _, match := range matches {
		if len(match) != 3 {
			return 0, fmt.Errorf("invalid duration format: %s", input)
		}

		value, _ := strconv.Atoi(match[1])
		unit := unitMap[match[2]]

		total += time.Duration(value) * unit
	}

	return total, nil
}
