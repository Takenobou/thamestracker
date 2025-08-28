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

// BuildEvent constructs and configures a VEVENT for a generic Event.
func BuildEvent(cal *ics.Calendar, e models.Event) {
	eid := MakeUID(e.Category, e.VesselName, e.Timestamp)
	event := cal.AddEvent(eid)
	now := time.Now()
	event.SetCreatedTime(now)
	event.SetDtStampTime(now)
	event.SetModifiedAt(now)

	description := ""
	status := "CONFIRMED"
	start := e.Timestamp
	end := start.Add(15 * time.Minute)

	summary := ""
	location := ""

	switch strings.ToLower(e.Category) {
	case "bridge":
		summary = fmt.Sprintf("Tower Bridge Lift - %s", e.VesselName)
		location = "222 Tower Bridge Road, London, SE1 2UP"
		description = fmt.Sprintf("Direction: %s", e.Direction)
		end = start.Add(10 * time.Minute)
		event.SetProperty("CATEGORIES", "BRIDGE")
		event.SetProperty("GEO", "51.505507;-0.075402")
		event.SetProperty("X-APPLE-STRUCTURED-LOCATION;VALUE=URI;X-APPLE-RADIUS=70;X-TITLE=Tower Bridge", "geo:51.505507,-0.075402")
	case "inport":
		event.SetAllDayStartAt(start)
		event.SetAllDayEndAt(start.Add(24 * time.Hour))
		summary = fmt.Sprintf("Vessel - %s", e.VesselName)
		location = e.Location
		desc := fmt.Sprintf("Location: %s", e.Location)
		if e.From != "" || e.To != "" {
			desc += fmt.Sprintf("\nVoyage: %s → %s", e.From, e.To)
		}
		if e.VoyageNo != "" {
			desc += fmt.Sprintf("\nVoyage No: %s", e.VoyageNo)
		}
		description = desc
		event.SetProperty("CATEGORIES", "INPORT")
		goto set_common
	default:
		summary = fmt.Sprintf("Vessel - %s", e.VesselName)
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
