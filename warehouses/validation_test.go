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

// TestValidateIdentityResolutionCounts verifies that valid counts are accepted
// and invalid, inconsistent, or unrepresentable counts are rejected.
func TestValidateIdentityResolutionCounts(t *testing.T) {

	quarterMax := math.MaxInt / 4
	tests := []struct {
		name    string
		counts  *warehouses.IdentityResolutionCounts
		wantErr bool
	}{
		{name: "nil", wantErr: true},
		{name: "empty", counts: &warehouses.IdentityResolutionCounts{}},
		{
			name: "valid",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Anonymous: 1, Recognized: 1},
				Identities:  warehouses.Counts{Anonymous: 2, Recognized: 1},
				Composition: warehouses.IdentityResolutionComposition{One: 1, Two: 1},
			},
		},
		{
			name: "valid open-ended composition",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 1},
				Identities:  warehouses.Counts{Recognized: 25},
				Composition: warehouses.IdentityResolutionComposition{MoreThanTwenty: 1},
			},
		},
		{
			name: "valid maximum profile count",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: math.MaxInt32},
				Identities:  warehouses.Counts{Recognized: math.MaxInt32},
				Composition: warehouses.IdentityResolutionComposition{One: math.MaxInt32},
			},
		},
		{
			name:    "negative profile count",
			counts:  &warehouses.IdentityResolutionCounts{Profiles: warehouses.Counts{Anonymous: -1}},
			wantErr: true,
		},
		{
			name:    "negative identity count",
			counts:  &warehouses.IdentityResolutionCounts{Identities: warehouses.Counts{Recognized: -1}},
			wantErr: true,
		},
		{
			name: "composition without profiles",
			counts: &warehouses.IdentityResolutionCounts{
				Identities:  warehouses.Counts{Recognized: 1},
				Composition: warehouses.IdentityResolutionComposition{One: 1},
			},
			wantErr: true,
		},
		{
			name: "negative composition count",
			counts: &warehouses.IdentityResolutionCounts{
				Composition: warehouses.IdentityResolutionComposition{One: -1},
			},
			wantErr: true,
		},
		{
			name: "profile count differs from composition total",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 2},
				Identities:  warehouses.Counts{Recognized: 4},
				Composition: warehouses.IdentityResolutionComposition{FourToTen: 1},
			},
			wantErr: true,
		},
		{
			name: "fewer anonymous identities than anonymous profiles",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Anonymous: 1},
				Identities:  warehouses.Counts{Recognized: 1},
				Composition: warehouses.IdentityResolutionComposition{One: 1},
			},
			wantErr: true,
		},
		{
			name: "fewer recognized identities than recognized profiles",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 1},
				Identities:  warehouses.Counts{Anonymous: 1},
				Composition: warehouses.IdentityResolutionComposition{One: 1},
			},
			wantErr: true,
		},
		{
			name: "fewer identities than composition minimum",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 2},
				Identities:  warehouses.Counts{Recognized: 2},
				Composition: warehouses.IdentityResolutionComposition{One: 1, Two: 1},
			},
			wantErr: true,
		},
		{
			name: "more identities than composition maximum",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 1},
				Identities:  warehouses.Counts{Recognized: 2},
				Composition: warehouses.IdentityResolutionComposition{One: 1},
			},
			wantErr: true,
		},
		{
			name: "profile total overflow",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles: warehouses.Counts{Anonymous: math.MaxInt, Recognized: 1},
			},
			wantErr: true,
		},
		{
			name: "profile total exceeds supported maximum",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Anonymous: math.MaxInt32, Recognized: 1},
				Identities:  warehouses.Counts{Anonymous: math.MaxInt32, Recognized: 1},
				Composition: warehouses.IdentityResolutionComposition{One: math.MaxInt32, Two: 1},
			},
			wantErr: true,
		},
		{
			name: "identity total overflow",
			counts: &warehouses.IdentityResolutionCounts{
				Identities: warehouses.Counts{Anonymous: math.MaxInt, Recognized: 1},
			},
			wantErr: true,
		},
		{
			name: "composition total overflow",
			counts: &warehouses.IdentityResolutionCounts{
				Identities: warehouses.Counts{Anonymous: math.MaxInt},
				Composition: warehouses.IdentityResolutionComposition{
					One: math.MaxInt, Two: 1,
				},
			},
			wantErr: true,
		},
		{
			name: "minimum multiplication overflow",
			counts: &warehouses.IdentityResolutionCounts{
				Identities:  warehouses.Counts{Anonymous: math.MaxInt},
				Composition: warehouses.IdentityResolutionComposition{Two: math.MaxInt/2 + 1},
			},
			wantErr: true,
		},
		{
			name: "minimum accumulation overflow",
			counts: &warehouses.IdentityResolutionCounts{
				Identities:  warehouses.Counts{Anonymous: math.MaxInt},
				Composition: warehouses.IdentityResolutionComposition{Two: quarterMax, Three: quarterMax},
			},
			wantErr: true,
		},
		{
			name: "valid unrepresentable upper bound",
			counts: &warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Anonymous: 1},
				Identities:  warehouses.Counts{Anonymous: math.MaxInt},
				Composition: warehouses.IdentityResolutionComposition{MoreThanTwenty: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := warehouses.ValidateIdentityResolutionCounts(test.counts)
			if test.wantErr && err == nil {
				t.Fatal("expected ValidateIdentityResolutionCounts to return an error, got nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected ValidateIdentityResolutionCounts to return no error, got %v", err)
			}
		})
	}

}
