// filepath: internal/calendar/eventbuilder.go
package calendar

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	ics "github.com/arran4/golang-ical"
)

// MakeUID creates a deterministic UID based on event type, name, and start time.
func MakeUID(eventType, name string, start time.Time) string {
	h := sha1.New()
	io.WriteString(h, eventType)
	io.WriteString(h, name)
	io.WriteString(h, start.UTC().Format(time.RFC3339Nano))
	sum := fmt.Sprintf("%x", h.Sum(nil))
	return sum[:16]
}

// DescribeVessel returns a multi-line description for a vessel event.
func DescribeVessel(v models.Vessel) string {
	var b strings.Builder
	// Route
	if v.Type == "inport" {
		b.WriteString(fmt.Sprintf("Location: %s\n", v.LocationName))
	} else {
		b.WriteString(fmt.Sprintf("Voyage: %s → %s\n", v.LocationFrom, v.LocationTo))
	}
	// Nationality
	if v.Nationality != "" {
		b.WriteString(fmt.Sprintf("Nationality: %s\n", v.Nationality))
	}
	// Arrival/Departure time
	b.WriteString(fmt.Sprintf("%s: %s %s", strings.Title(v.Type), v.Date, v.Time))
	return b.String()
}

// BuildEvent constructs and configures a VEVENT for a generic Event.
func BuildEvent(cal *ics.Calendar, e models.Event) {
	eid := MakeUID(e.Category, e.VesselName, e.Timestamp)
	event := cal.AddEvent(eid)
	now := time.Now()
	event.SetCreatedTime(now)
	event.SetDtStampTime(now)
	event.SetModifiedAt(now)

	summary := e.VesselName
	location := e.Location
	description := ""
	status := "CONFIRMED"
	start := e.Timestamp
	end := start.Add(15 * time.Minute)

	switch strings.ToLower(e.Category) {
	case "bridge":
		summary = fmt.Sprintf("Tower Bridge Lift – %s", e.VesselName)
		location = "Tower Bridge Road, London, SE1 2UP, England"
		description = fmt.Sprintf("Direction: %s", e.Direction)
		end = start.Add(10 * time.Minute)
		event.SetProperty("CATEGORIES", "BRIDGE")
		event.SetProperty("GEO", "51.5055;-0.0754")
	case "inport":
		event.SetAllDayStartAt(start)
		event.SetAllDayEndAt(start.Add(24 * time.Hour))
		summary = fmt.Sprintf("Vessel – %s", e.VesselName)
		location = e.Location
		description = fmt.Sprintf("Location: %s\nVoyage: %s → %s\nVoyage No: %s", e.Location, e.From, e.To, e.VoyageNo)
		event.SetProperty("CATEGORIES", "INPORT")
		// All-day event, so skip SetStartAt/SetEndAt
		goto set_common
	default:
		summary = fmt.Sprintf("Vessel – %s", e.VesselName)
		location = e.Location
		description = fmt.Sprintf("Voyage: %s → %s\nVoyage No: %s", e.From, e.To, e.VoyageNo)
		if e.Category == "forecast" {
			status = "TENTATIVE"
		}
		event.SetProperty("CATEGORIES", strings.ToUpper(e.Category))
	}
	event.SetStartAt(start)
	event.SetEndAt(end)
set_common:
	event.SetSummary(summary)
	event.SetLocation(location)
	event.SetDescription(description)
	event.SetProperty("STATUS", status)
	alarm := event.AddAlarm()
	alarm.SetTrigger("-PT10M")
	alarm.SetAction("DISPLAY")
	alarm.SetDescription("Reminder")
}
