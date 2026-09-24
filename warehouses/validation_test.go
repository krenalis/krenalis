// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses_test

import (
	"math"
	"testing"

	"github.com/krenalis/krenalis/warehouses"
)

// TestValidateIdentityCounts verifies that formally valid warehouse identity
// counts are accepted and invalid or unrepresentable counts are rejected.
func TestValidateIdentityCounts(t *testing.T) {

	tests := []struct {
		name    string
		counts  *warehouses.IdentityCounts
		wantErr bool
	}{
		{name: "nil", wantErr: true},
		{name: "empty", counts: &warehouses.IdentityCounts{}},
		{name: "valid sparse counts", counts: &warehouses.IdentityCounts{
			Anonymous:      map[string]int{"connection01": 2},
			Recognized:     map[string]int{"connection01": 3, "connection02": 1},
			WithoutProfile: map[string]int{"connection01": 4},
		}},
		{name: "valid maximum count", counts: &warehouses.IdentityCounts{
			Anonymous:      map[string]int{"connection01": math.MaxInt},
			WithoutProfile: map[string]int{"connection01": math.MaxInt},
		}},
		{name: "negative anonymous", counts: &warehouses.IdentityCounts{
			Anonymous: map[string]int{"connection01": -1},
		}, wantErr: true},
		{name: "negative recognized", counts: &warehouses.IdentityCounts{
			Recognized: map[string]int{"connection01": -1},
		}, wantErr: true},
		{name: "negative without profile", counts: &warehouses.IdentityCounts{
			WithoutProfile: map[string]int{"connection01": -1},
		}, wantErr: true},
		{name: "without profile exceeds connection total", counts: &warehouses.IdentityCounts{
			Anonymous: map[string]int{"connection01": 1}, WithoutProfile: map[string]int{"connection01": 2},
		}, wantErr: true},
		{name: "anonymous total overflow", counts: &warehouses.IdentityCounts{
			Anonymous: map[string]int{"connection01": math.MaxInt, "connection02": 1},
		}, wantErr: true},
		{name: "recognized total overflow", counts: &warehouses.IdentityCounts{
			Recognized: map[string]int{"connection01": math.MaxInt, "connection02": 1},
		}, wantErr: true},
		{name: "connection total overflow", counts: &warehouses.IdentityCounts{
			Anonymous: map[string]int{"connection01": math.MaxInt}, Recognized: map[string]int{"connection01": 1},
		}, wantErr: true},
		{name: "workspace total overflow", counts: &warehouses.IdentityCounts{
			Anonymous: map[string]int{"connection01": math.MaxInt}, Recognized: map[string]int{"connection02": 1},
		}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := warehouses.ValidateIdentityCounts(test.counts)
			if test.wantErr && err == nil {
				t.Fatal("expected ValidateIdentityCounts to return an error, got nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected ValidateIdentityCounts to return no error, got %v", err)
			}
		})
	}

}
