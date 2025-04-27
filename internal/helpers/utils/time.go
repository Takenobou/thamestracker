package utils

import (
	"time"
)

// ParseRawTimestamp parses a raw timestamp ("2006-01-02 15:04:05.000") in UTC, converts to Europe/London,
// and returns formatted date ("02/01/2006") and time ("15:04").
// If rawTS is empty or parse fails, uses current time.
func ParseRawTimestamp(rawTS string) (string, string) {
	var t time.Time
	var err error
	if rawTS == "" {
		t = time.Now()
	} else {
		t, err = time.ParseInLocation("2006-01-02 15:04:05.000", rawTS, time.UTC)
		if err != nil {
			t = time.Now()
		}
	}
	loc, _ := time.LoadLocation("Europe/London")
	local := t.In(loc)
	return local.Format("02/01/2006"), local.Format("15:04")
}
