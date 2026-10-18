// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import "fmt"

// usaState binds Census FIPS/GNIS and USPS identity to the existing L2 ordinal.
type usaState struct {
	FIPS            string `json:"fips"`
	USPS            string `json:"usps"`
	Name            string `json:"name"`
	GNIS            string `json:"gnis"`
	Division        string `json:"division"`
	DivisionOrdinal uint32 `json:"divisionOrdinal"`
}

func resolveUSAState(usps, fips string) (usaState, error) {
	for _, state := range usaStates() {
		if state.USPS == usps && state.FIPS == fips {
			return state, nil
		}
	}
	return usaState{}, fmt.Errorf("unknown or incompatible USA state identifiers")
}

// usaStates follows Spec-USA E.1 and Census state.txt/us_regdiv.pdf; originals
// and SHA-256 values are recorded in the USA Layer 1 CANDIDATE provenance.
func usaStates() [51]usaState {
	return [51]usaState{
		{"01", "AL", "Alabama", "01779775", "east-south-central", 5},
		{"02", "AK", "Alaska", "01785533", "pacific", 8},
		{"04", "AZ", "Arizona", "01779777", "mountain", 7},
		{"05", "AR", "Arkansas", "00068085", "west-south-central", 6},
		{"06", "CA", "California", "01779778", "pacific", 8},
		{"08", "CO", "Colorado", "01779779", "mountain", 7},
		{"09", "CT", "Connecticut", "01779780", "new-england", 0},
		{"10", "DE", "Delaware", "01779781", "south-atlantic", 4},
		{"11", "DC", "District of Columbia", "01702382", "south-atlantic", 4},
		{"12", "FL", "Florida", "00294478", "south-atlantic", 4},
		{"13", "GA", "Georgia", "01705317", "south-atlantic", 4},
		{"15", "HI", "Hawaii", "01779782", "pacific", 8},
		{"16", "ID", "Idaho", "01779783", "mountain", 7},
		{"17", "IL", "Illinois", "01779784", "east-north-central", 2},
		{"18", "IN", "Indiana", "00448508", "east-north-central", 2},
		{"19", "IA", "Iowa", "01779785", "west-north-central", 3},
		{"20", "KS", "Kansas", "00481813", "west-north-central", 3},
		{"21", "KY", "Kentucky", "01779786", "east-south-central", 5},
		{"22", "LA", "Louisiana", "01629543", "west-south-central", 6},
		{"23", "ME", "Maine", "01779787", "new-england", 0},
		{"24", "MD", "Maryland", "01714934", "south-atlantic", 4},
		{"25", "MA", "Massachusetts", "00606926", "new-england", 0},
		{"26", "MI", "Michigan", "01779789", "east-north-central", 2},
		{"27", "MN", "Minnesota", "00662849", "west-north-central", 3},
		{"28", "MS", "Mississippi", "01779790", "east-south-central", 5},
		{"29", "MO", "Missouri", "01779791", "west-north-central", 3},
		{"30", "MT", "Montana", "00767982", "mountain", 7},
		{"31", "NE", "Nebraska", "01779792", "west-north-central", 3},
		{"32", "NV", "Nevada", "01779793", "mountain", 7},
		{"33", "NH", "New Hampshire", "01779794", "new-england", 0},
		{"34", "NJ", "New Jersey", "01779795", "middle-atlantic", 1},
		{"35", "NM", "New Mexico", "00897535", "mountain", 7},
		{"36", "NY", "New York", "01779796", "middle-atlantic", 1},
		{"37", "NC", "North Carolina", "01027616", "south-atlantic", 4},
		{"38", "ND", "North Dakota", "01779797", "west-north-central", 3},
		{"39", "OH", "Ohio", "01085497", "east-north-central", 2},
		{"40", "OK", "Oklahoma", "01102857", "west-south-central", 6},
		{"41", "OR", "Oregon", "01155107", "pacific", 8},
		{"42", "PA", "Pennsylvania", "01779798", "middle-atlantic", 1},
		{"44", "RI", "Rhode Island", "01219835", "new-england", 0},
		{"45", "SC", "South Carolina", "01779799", "south-atlantic", 4},
		{"46", "SD", "South Dakota", "01785534", "west-north-central", 3},
		{"47", "TN", "Tennessee", "01325873", "east-south-central", 5},
		{"48", "TX", "Texas", "01779801", "west-south-central", 6},
		{"49", "UT", "Utah", "01455989", "mountain", 7},
		{"50", "VT", "Vermont", "01779802", "new-england", 0},
		{"51", "VA", "Virginia", "01779803", "south-atlantic", 4},
		{"53", "WA", "Washington", "01779804", "pacific", 8},
		{"54", "WV", "West Virginia", "01779805", "south-atlantic", 4},
		{"55", "WI", "Wisconsin", "01779806", "east-north-central", 2},
		{"56", "WY", "Wyoming", "01779807", "mountain", 7},
	}
}
