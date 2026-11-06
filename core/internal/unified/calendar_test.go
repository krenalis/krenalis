// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
)

func TestCalendarPeriodBounds(t *testing.T) {
	for _, test := range []struct {
		name, zone, unit string
		count            uint64
		at, start, end   string
	}{
		{"last month includes DST", "Europe/Rome", "month", 1, "2026-11-06T12:00:00Z", "2026-09-30T22:00:00Z", "2026-10-31T23:00:00Z"},
		{"spring day", "Europe/Rome", "day", 1, "2026-03-30T12:00:00Z", "2026-03-28T23:00:00Z", "2026-03-29T22:00:00Z"},
		{"autumn day", "Europe/Rome", "day", 1, "2026-10-26T12:00:00Z", "2026-10-24T22:00:00Z", "2026-10-25T23:00:00Z"},
		{"spring week", "Europe/Rome", "week", 1, "2026-03-30T12:00:00Z", "2026-03-22T23:00:00Z", "2026-03-29T22:00:00Z"},
		{"autumn week", "Europe/Rome", "week", 1, "2026-10-26T12:00:00Z", "2026-10-18T22:00:00Z", "2026-10-25T23:00:00Z"},
		{"Monday boundary", "UTC", "week", 1, "2026-11-02T00:00:00Z", "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z"},
		{"local Monday on UTC Sunday", "Asia/Tokyo", "week", 1, "2026-11-01T16:00:00Z", "2026-10-25T15:00:00Z", "2026-11-01T15:00:00Z"},
		{"local Sunday on UTC Monday", "America/New_York", "week", 1, "2026-11-02T02:00:00Z", "2026-10-19T04:00:00Z", "2026-10-26T04:00:00Z"},
		{"year boundary", "UTC", "month", 1, "2027-01-01T00:00:00Z", "2026-12-01T00:00:00Z", "2027-01-01T00:00:00Z"},
		{"leap month", "UTC", "month", 2, "2024-03-01T12:00:00Z", "2024-01-01T00:00:00Z", "2024-03-01T00:00:00Z"},
		{"leap day", "UTC", "day", 1, "2024-03-01T12:00:00Z", "2024-02-29T00:00:00Z", "2024-03-01T00:00:00Z"},
		{"east of UTC", "Asia/Tokyo", "day", 1, "2026-11-06T00:00:00Z", "2026-11-04T15:00:00Z", "2026-11-05T15:00:00Z"},
		{"west of UTC", "America/New_York", "month", 1, "2026-11-06T12:00:00Z", "2026-10-01T04:00:00Z", "2026-11-01T04:00:00Z"},
		{"last supported year", "UTC", "month", 1, "9999-12-31T23:59:59Z", "9999-11-01T00:00:00Z", "9999-12-01T00:00:00Z"},
		{"local year beyond UTC domain", "Pacific/Kiritimati", "day", 1, "9999-12-31T23:00:00Z", "9999-12-30T10:00:00Z", "9999-12-31T10:00:00Z"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calendar, err := ResolveCalendar("gregorian-monday/"+test.zone, "1")
			if err != nil {
				t.Fatalf("expected fixed calendar, got %v", err)
			}
			var instants [3]time.Time
			for i, text := range []string{test.at, test.start, test.end} {
				instants[i], err = time.Parse(time.RFC3339, text)
				if err != nil {
					t.Fatalf("expected fixture instant, got %v", err)
				}
			}
			scope := Scope{Calendar: &CalendarPeriod{Calendar: calendar, Unit: test.unit, Count: test.count}}
			start, end, err := scope.bounds(instants[0])
			if err != nil {
				t.Fatalf("expected complete calendar bounds, got %v", err)
			}
			if !start.Equal(instants[1]) || !end.Equal(instants[2]) ||
				start.Location() != time.UTC || end.Location() != time.UTC {
				t.Fatalf("expected UTC [%s,%s), got [%s,%s)", instants[1], instants[2], start, end)
			}
		})
	}
}

func TestCalendarPeriodUnavailable(t *testing.T) {
	for _, test := range []struct {
		name, zone, unit, at string
		count                uint64
	}{
		{"ambiguous midnight", "America/Havana", "day", "2026-11-02T12:00:00Z", 1},
		{"nonexistent midnight", "America/Sao_Paulo", "day", "2018-11-05T12:00:00Z", 1},
		{"skipped date", "Pacific/Apia", "day", "2011-12-31T12:00:00Z", 2},
		{"before timestamp domain", "UTC", "month", "0001-02-01T00:00:00Z", 2},
		{"large count", "UTC", "week", "2026-11-06T12:00:00Z", 1 << 63},
	} {
		t.Run(test.name, func(t *testing.T) {
			calendar, err := ResolveCalendar("gregorian-monday/"+test.zone, "1")
			if err != nil {
				t.Fatalf("expected fixed calendar, got %v", err)
			}
			at, err := time.Parse(time.RFC3339, test.at)
			if err != nil {
				t.Fatalf("expected fixture instant, got %v", err)
			}
			scope := Scope{Calendar: &CalendarPeriod{Calendar: calendar, Unit: test.unit, Count: test.count}}
			_, _, err = scope.bounds(at)
			if err != nil {
				if _, ok := errors.AsType[*UnsupportedError](err); !ok {
					t.Fatalf("expected unavailable calendar capability, got %v", err)
				}
				return
			}
			t.Fatal("expected unavailable calendar capability, got success")
		})
	}
}

func TestResolveCalendar(t *testing.T) {
	for _, test := range []struct{ definition, revision string }{
		{"", "1"}, {"Europe/Rome", "1"}, {"gregorian-monday/", "1"},
		{"gregorian-monday/Local", "1"}, {"gregorian-monday/../Europe/Rome", "1"},
		{"gregorian-monday/posix/Europe/Rome", "1"}, {"gregorian-monday/unknown", "1"},
		{"gregorian-monday/UTC", ""}, {"gregorian-monday/UTC", "2"},
	} {
		calendar, err := ResolveCalendar(test.definition, test.revision)
		if err != nil {
			if calendar != nil {
				t.Fatalf("expected no resolved calendar on error, got %+v", calendar)
			}
			continue
		}
		t.Fatal("expected rejected calendar reference, got success")
	}
}
