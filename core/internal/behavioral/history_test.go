// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"testing"
	"time"
)

// TestHistoryBounds resolves only T and rejects unresolved or mixed periods.
func TestHistoryBounds(t *testing.T) {

	history, err := ResolveHistory("registered-queryable-events", "1")
	if err != nil {
		t.Fatalf("expected recorded history binding, got %v", err)
	}
	for _, at := range []time.Time{
		time.Date(1, 1, 1, 0, 0, 0, 1000, time.UTC),
		time.Date(2026, 2, 1, 12, 0, 0, 123456000, time.FixedZone("offset", 3600)),
		time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC),
	} {
		scope := Scope{Event: "Purchase", Grain: "registered-message", History: history}
		start, end, err := scope.bounds(at)
		if err != nil {
			t.Fatalf("expected unbounded history before %s, got %v", at, err)
		}
		if !start.IsZero() || !end.Equal(at) || end.Location() != time.UTC {
			t.Fatalf("expected no lower bound and UTC end %s, got %s and %s", at, start, end)
		}
	}

	for _, scope := range []Scope{
		{History: &History{}},
		{History: history, Start: time.Now()},
		{History: history, End: time.Now()},
		{History: history, Elapsed: &ElapsedPeriod{Older: time.Hour}},
		{History: history, Calendar: &CalendarPeriod{}},
	} {
		_, _, err := scope.bounds(time.Now().Truncate(time.Microsecond))
		if err == nil {
			t.Fatal("expected unresolved or conflicting history rejection, got success")
		}
	}

}
