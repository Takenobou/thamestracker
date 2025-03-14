package models

// BridgeLift represents a Tower Bridge lift event
type BridgeLift struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Vessel    string `json:"vessel"`
	Direction string `json:"direction"`
}
