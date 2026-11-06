// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	_ "embed"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

// calendarZones lists the IANA names and aliases in Go 1.27.2's zoneinfo.zip,
// excluding the placeholder Factory. Offsets still come from time.LoadLocation.
//
//go:embed calendar_zones.txt
var calendarZones string

// Calendar is a resolved built-in Gregorian calendar with Monday-first weeks.
// Its definition includes an explicit IANA timezone and meaning revision.
type Calendar struct {
	location *time.Location
}

// ResolveCalendar resolves a gregorian-monday/TIMEZONE definition.
// Only immutable meaning revision 1 is available.
// Local and implicit UTC are not calendar definitions.
func ResolveCalendar(definition, meaningRevision string) (*Calendar, error) {
	zone, ok := strings.CutPrefix(definition, "gregorian-monday/")
	if !ok || len(definition) > 256 || zone == "" || zone == "Local" ||
		strings.ContainsAny(zone, "\x00\\.\r\n") || strings.HasPrefix(zone, "/") ||
		strings.HasPrefix(zone, "posix/") || strings.HasPrefix(zone, "right/") ||
		!strings.Contains(calendarZones, "\n"+zone+"\n") {
		return nil, fmt.Errorf("invalid fixed calendar definition")
	}
	if meaningRevision != "1" {
		return nil, fmt.Errorf("unresolved calendar meaning revision")
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, fmt.Errorf("unresolved calendar timezone")
	}
	return &Calendar{location: location}, nil
}

// midnight resolves a civil boundary that identifies exactly one instant.
func (calendar *Calendar) midnight(civil time.Time) (time.Time, error) {
	// IANA UTC offsets fit within a day. Inspect all zone intervals that could
	// contain this midnight instead of letting time.Date choose a DST branch.
	var result time.Time
	found := false
	limit := civil.Add(24 * time.Hour)
	for cursor := civil.Add(-24 * time.Hour); !cursor.After(limit); {
		local := cursor.In(calendar.location)
		_, offset := local.Zone()
		if offset < -24*60*60 || offset > 24*60*60 {
			return time.Time{}, &UnsupportedError{Capability: "calendar UTC offset outside supported range"}
		}
		candidate := civil.Add(-time.Duration(offset) * time.Second)
		stamp := candidate.In(calendar.location)
		if stamp.Year() == civil.Year() && stamp.Month() == civil.Month() && stamp.Day() == civil.Day() &&
			stamp.Hour() == 0 && stamp.Minute() == 0 && stamp.Second() == 0 {
			if found && !result.Equal(candidate) {
				return time.Time{}, &UnsupportedError{Capability: "ambiguous calendar boundary"}
			}
			result = candidate
			found = true
		}
		_, end := local.ZoneBounds()
		if !end.IsZero() && !end.After(cursor) {
			// Go's TZif tail approximates the last zone interval with a
			// 365-day year. On December 31 of a leap year its end can be in
			// the past, although the offset lasts until the next UTC year.
			end = time.Date(cursor.UTC().Year()+1, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		if end.IsZero() || end.After(limit) {
			break
		}
		cursor = end
	}
	if !found {
		return time.Time{}, &UnsupportedError{Capability: "nonexistent calendar boundary"}
	}
	return result.UTC(), nil
}

// CalendarPeriod selects the last Count complete civil days, weeks or months.
// The current period is excluded; Count does not count qualified buckets.
type CalendarPeriod struct {
	Calendar *Calendar
	Unit     string
	Count    uint64
}

func (period CalendarPeriod) bounds(at time.Time) (time.Time, time.Time, error) {

	if period.Calendar == nil || period.Calendar.location == nil || period.Count == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid complete calendar period")
	}
	switch period.Unit {
	case "day", "week", "month":
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid calendar period unit")
	}
	// Any larger count exceeds the supported 1..9999 timestamp domain, even
	// in days. Check before converting to int or multiplying a week count.
	if period.Count > 366*10000 {
		return time.Time{}, time.Time{},
			&UnsupportedError{Capability: "calendar period outside supported timestamp range"}
	}

	local := at.In(period.Calendar.location)
	year, month, day := local.Date()
	end := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	var start time.Time
	switch period.Unit {
	case "day":
		start = end.AddDate(0, 0, -int(period.Count))
	case "week":
		end = end.AddDate(0, 0, -(int(local.Weekday())+6)%7)
		start = end.AddDate(0, 0, -7*int(period.Count))
	case "month":
		end = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		start = end.AddDate(0, -int(period.Count), 0)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("core/internal/unified: unexpected calendar unit %q", period.Unit)
	}
	start, err := period.Calendar.midnight(start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err = period.Calendar.midnight(end)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}
