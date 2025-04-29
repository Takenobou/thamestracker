package cache

import (
	"fmt"
)

// KeyBridgeLifts returns the cache key for bridge lifts.
func KeyBridgeLifts() string {
	return "bridge_lifts"
}

// KeyVessels returns the cache key for vessels of the given type, including version.
func KeyVessels(vesselType string) string {
	vt := vesselType
	if vt == "all" {
		return "v3_all_vessels"
	}
	return fmt.Sprintf("v3_vessels_%s", vt)
}

// KeyVesselsByLoc returns the cache key for filtered vessels by type and location.
func KeyVesselsByLoc(vesselType, location string) string {
	return fmt.Sprintf("v3_vessels_%s_location_%s", vesselType, location)
}
