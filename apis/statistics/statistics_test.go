//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package statistics

import (
	"testing"
)

func Test_TimeSlot(t *testing.T) {

	tests := []int32{0, 1, 5, 99, 28714101, maxTimeslot}

	for _, ts := range tests {
		got := TimeSlotFromTime(TimeSlotToTime(ts))
		if ts != got {
			t.Fatalf("expected %d, got %d", ts, got)
		}
	}

}
