package models

// Deprecated: use Event instead.
type Vessel struct {
	Time         string `json:"time"`
	Date         string `json:"date"`
	LocationFrom string `json:"location_from,omitempty"`
	LocationTo   string `json:"location_to,omitempty"`
	LocationName string `json:"location_name,omitempty"`
	Name         string `json:"name"`
	Nationality  string `json:"nationality,omitempty"`
	VoyageNo     string `json:"voyage_number"`
	Type         string `json:"type"` // e.g., "arrival", "departure", "forecast", "inport"
}
