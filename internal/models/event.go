package models

import "time"

// Event represents a unified event for bridge lifts and vessel movements.
type Event struct {
	Timestamp  time.Time `json:"timestamp"` // RFC3339
	VesselName string    `json:"vessel_name"`
	Category   string    `json:"category"` // e.g. "bridge", "inport", "arrival", etc.
	VoyageNo   string    `json:"voyage_number,omitempty"`
	Direction  string    `json:"direction,omitempty"`
	From       string    `json:"from,omitempty"`
	To         string    `json:"to,omitempty"`
	Location   string    `json:"location,omitempty"`
}
