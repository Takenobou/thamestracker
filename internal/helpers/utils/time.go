// Package utils supplies small helper utilities.
package utils

import "time"

var LondonLocation *time.Location

const (
	SrcLayout  = "2006-01-02 15:04:05.000"
	DateLayout = "02/01/2006"
	TimeLayout = "15:04"
)

func init() {
	var err error
	LondonLocation, err = time.LoadLocation("Europe/London")
	if err != nil {
		LondonLocation = time.UTC // fallback
	}
}

// ParseRawTimestamp converts a PLA-style raw timestamp (layout
// "2006-01-02 15:04:05.000") into separate date ("02/01/2006") and time
func ParseRawTimestamp(rawTS string) (string, string) {
	if rawTS != "" {
		if t, err := time.ParseInLocation(SrcLayout, rawTS, LondonLocation); err == nil {
			return t.Format(DateLayout), t.Format(TimeLayout)
		}
	}
	now := time.Now().In(LondonLocation)
	return now.Format(DateLayout), now.Format(TimeLayout)
}
