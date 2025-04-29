package utils

import (
	"time"
)

// ParseRawTimestamp parses a raw timestamp ("2006-01-02 15:04:05.000"),
// and returns formatted date ("02/01/2006") and time ("15:04").
// If rawTS is empty or parse fails, uses current time.
func ParseRawTimestamp(rawTS string) (string, string) {
	if rawTS == "" {
		now := time.Now()
		return now.Format("02/01/2006"), now.Format("15:04")
	}
	t, err := time.Parse("2006-01-02 15:04:05.000", rawTS)
	if err != nil {
		now := time.Now()
		return now.Format("02/01/2006"), now.Format("15:04")
	}
	return t.Format("02/01/2006"), t.Format("15:04")
}
