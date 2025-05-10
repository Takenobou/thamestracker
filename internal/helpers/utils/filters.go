package utils

import (
	"sort"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
)

// FilterOptions defines generic filters for Event slices.
type FilterOptions struct {
	Name     string
	Category string
	After    string
	Before   string
	Unique   bool
	Location string
}

// FilterEvents applies generic filters to a slice of Event.
func FilterEvents(events []models.Event, opts FilterOptions) []models.Event {
	var filtered []models.Event
	name := strings.ToLower(opts.Name)
	category := strings.ToLower(opts.Category)
	location := strings.ToLower(opts.Location)
	var after, before time.Time
	var afterSet, beforeSet bool
	if opts.After != "" {
		t, err := time.Parse(time.RFC3339, opts.After)
		if err == nil {
			after = t
			afterSet = true
		}
	}
	if opts.Before != "" {
		t, err := time.Parse(time.RFC3339, opts.Before)
		if err == nil {
			before = t
			beforeSet = true
		}
	}
	for _, e := range events {
		if name != "" && !strings.Contains(strings.ToLower(e.VesselName), name) {
			continue
		}
		if category != "" && category != "all" && strings.ToLower(e.Category) != category {
			continue
		}
		if location != "" {
			locMatch := false
			switch strings.ToLower(e.Category) {
			case "inport":
				locMatch = strings.Contains(strings.ToLower(e.Location), location)
			case "arrivals", "forecast":
				locMatch = strings.Contains(strings.ToLower(e.To), location)
			case "departures":
				locMatch = strings.Contains(strings.ToLower(e.From), location)
			default:
				locMatch = strings.Contains(strings.ToLower(e.Location), location) ||
					strings.Contains(strings.ToLower(e.To), location) ||
					strings.Contains(strings.ToLower(e.From), location)
			}
			if !locMatch {
				continue
			}
		}
		if afterSet && e.Timestamp.Before(after) {
			continue
		}
		if beforeSet && e.Timestamp.After(before) {
			continue
		}
		filtered = append(filtered, e)
	}
	if opts.Unique {
		if category == "bridge" {
			filtered = filterHybridUniqueBridgeEvents(filtered, 0.10, 8)
		} else {
			filtered = filterUniqueEvents(filtered)
		}
	}
	return filtered
}

// filterUniqueEvents de-duplicates events by VesselName and Category.
func filterUniqueEvents(events []models.Event) []models.Event {
	seen := make(map[string]bool)
	var result []models.Event
	for _, e := range events {
		key := e.Category + ":" + e.VesselName
		if !seen[key] {
			seen[key] = true
			result = append(result, e)
		}
	}
	return result
}

// filterHybridUniqueBridgeEvents filters bridge events by excluding vessels in the top percentile most frequent or that appear more than a specified maximum count.
func filterHybridUniqueBridgeEvents(events []models.Event, percentile float64, maxCount int) []models.Event {
	counts := make(map[string]int)
	for _, e := range events {
		if strings.ToLower(e.Category) == "bridge" {
			counts[e.VesselName]++
		}
	}
	type vesselCount struct {
		Name  string
		Count int
	}
	var vessels []vesselCount
	for name, count := range counts {
		vessels = append(vessels, vesselCount{name, count})
	}
	sort.Slice(vessels, func(i, j int) bool { return vessels[i].Count > vessels[j].Count })
	cutoff := int(float64(len(vessels)) * percentile)
	if cutoff < 1 {
		cutoff = 1
	}
	topVessels := make(map[string]bool)
	for i := 0; i < cutoff && i < len(vessels); i++ {
		topVessels[vessels[i].Name] = true
	}
	var result []models.Event
	for _, e := range events {
		if strings.ToLower(e.Category) == "bridge" {
			if !topVessels[e.VesselName] && counts[e.VesselName] <= maxCount {
				result = append(result, e)
			}
		} else {
			result = append(result, e)
		}
	}
	return result
}
