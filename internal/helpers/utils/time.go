// Package utils supplies small helper utilities.
package utils

import "time"

// ParseRawTimestamp converts a PLA-style raw timestamp (layout
// "2006-01-02 15:04:05.000") into separate date ("02/01/2006") and time
func ParseRawTimestamp(rawTS string) (string, string) {
	const srcLayout = "2006-01-02 15:04:05.000"
	const dateLayout = "02/01/2006"
	const timeLayout = "15:04"

	london, _ := time.LoadLocation("Europe/London")

	if rawTS != "" {
		if t, err := time.ParseInLocation(srcLayout, rawTS, london); err == nil {
			return t.Format(dateLayout), t.Format(timeLayout)
		}
	}
	now := time.Now().In(london)
	return now.Format(dateLayout), now.Format(timeLayout)
}
