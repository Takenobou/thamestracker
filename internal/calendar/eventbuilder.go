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

// BuildBridgeEvent constructs and configures a VEVENT for a bridge lift.
func BuildBridgeEvent(cal *ics.Calendar, lift models.BridgeLift, start time.Time) {
	eid := MakeUID("bridge", lift.Vessel, start)
	e := cal.AddEvent(eid)
	now := time.Now()
	e.SetCreatedTime(now)
	e.SetDtStampTime(now)
	e.SetModifiedAt(now)
	e.SetStartAt(start)
	e.SetEndAt(start.Add(10 * time.Minute))
	e.SetSummary(fmt.Sprintf("Tower Bridge Lift – %s", lift.Vessel))
	e.SetDescription(fmt.Sprintf("Direction: %s", lift.Direction))
	e.SetLocation("Tower Bridge Road, London, SE1 2UP, England")
	e.SetProperty("CATEGORIES", "BRIDGE")
	e.SetProperty("STATUS", "CONFIRMED")
	e.SetProperty("GEO", "51.5055;-0.0754")
	// Default alarm
	alarm := e.AddAlarm()
	alarm.SetTrigger("-PT10M")
	alarm.SetAction("DISPLAY")
	alarm.SetDescription("Reminder")
}

// BuildVesselEvent constructs and configures a VEVENT for a vessel event.
func BuildVesselEvent(cal *ics.Calendar, v models.Vessel, start, end time.Time) {
	eid := MakeUID(v.Type, v.Name, start)
	e := cal.AddEvent(eid)
	now := time.Now()
	e.SetCreatedTime(now)
	e.SetDtStampTime(now)
	e.SetModifiedAt(now)
	// All-day for inport
	if v.Type == "inport" {
		e.SetAllDayStartAt(start)
		e.SetAllDayEndAt(end)
		e.SetSummary(fmt.Sprintf("Vessel – %s", v.Name))
	} else {
		e.SetStartAt(start)
		e.SetEndAt(end)
		e.SetSummary(fmt.Sprintf("Vessel – %s", v.Name))
	}
	// Common fields
	e.SetProperty("CATEGORIES", strings.ToUpper(v.Type))
	// STATUS
	status := "CONFIRMED"
	if v.Type == "forecast" {
		status = "TENTATIVE"
	}
	e.SetProperty("STATUS", status)
	// Location and description
	e.SetLocation(v.LocationName)
	e.SetDescription(DescribeVessel(v))
	// Default alarm
	alarm := e.AddAlarm()
	alarm.SetTrigger("-PT10M")
	alarm.SetAction("DISPLAY")
	alarm.SetDescription("Reminder")
}
