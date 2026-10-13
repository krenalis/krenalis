// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Frozen source hashes used only by the offline converter.
const (
	ageWeightsSHA256  = "5abdc04bb817e5fc300ce71920c1dedf269551d3cc0065148264d26a39c84555"
	firstNamesSHA256  = "ed249d8095d3133d90a662bb3dc4d592941d9aa1de1d476a8000b392761ed8db"
	lastNamesSHA256   = "d44f1859847b5c94066d20c0846fb16859827802bdf878d59e17dc26a046f96b"
	placesSHA256      = "27539c5025412c2f03b199cbd6fdea7b9bda8fb57f6d8665883c70eeaa7b05b6"
	postalCodesSHA256 = "0fed3e2822c13a54c4255d2b83fe0fd63aea27b432d63a682b28f64f4981426c"
	streetNamesSHA256 = "13a757c70266ad96f0c1f9d76ff32597995090009e860eec86a1dded5b5095ec"
)

func parseAgeRows(records [][]string) ([]ageRow, []uint64, error) {

	if len(records) != 55 {
		return nil, nil, fmt.Errorf("%w: age_weights.csv has %d rows, expected 55", ErrCorruptFrozenDataset, len(records))
	}

	ages := make([]ageRow, len(records))
	weights := make([]uint64, len(records))
	for position, record := range records {
		age, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, nil, fmt.Errorf("%w: age_weights.csv row %d has invalid age", ErrCorruptFrozenDataset, position+2)
		}
		if age != position+21 {
			return nil, nil, fmt.Errorf("%w: age_weights.csv row %d has invalid age", ErrCorruptFrozenDataset, position+2)
		}
		weight, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: age_weights.csv row %d has invalid weight", ErrCorruptFrozenDataset, position+2)
		}
		if weight == 0 {
			return nil, nil, fmt.Errorf("%w: age_weights.csv row %d has invalid weight", ErrCorruptFrozenDataset, position+2)
		}
		ages[position] = ageRow{age: age, weight: weight}
		weights[position] = weight
	}

	return ages, weights, nil
}

func parseFirstNameRows(records [][]string) (map[firstNameGroup][]firstNameRow, map[firstNameGroup][]uint64, error) {

	if len(records) != 800 {
		return nil, nil, fmt.Errorf("%w: first_names.csv has %d rows, expected 800", ErrCorruptFrozenDataset, len(records))
	}

	groups := map[firstNameGroup][]firstNameRow{}
	weights := map[firstNameGroup][]uint64{}
	for position, record := range records {

		gender := Gender(record[0])
		if gender != GenderFemale && gender != GenderMale {
			return nil, nil, fmt.Errorf("%w: first_names.csv row %d has invalid gender", ErrCorruptFrozenDataset, position+2)
		}
		if !validCohort(record[1]) {
			return nil, nil, fmt.Errorf("%w: first_names.csv row %d has invalid cohort", ErrCorruptFrozenDataset, position+2)
		}
		if !validDisplayName(record[2]) || !validASCIIAtom(record[3]) {
			return nil, nil, fmt.Errorf("%w: first_names.csv row %d has invalid name", ErrCorruptFrozenDataset, position+2)
		}
		weight, err := parseTierWeight(record[4])
		if err != nil {
			return nil, nil, fmt.Errorf("%w: first_names.csv row %d has invalid weight", ErrCorruptFrozenDataset, position+2)
		}

		group := firstNameGroup{gender: gender, cohort: record[1]}
		groups[group] = append(groups[group], firstNameRow{display: record[2], asciiAtom: record[3], weight: weight})
		weights[group] = append(weights[group], weight)

	}

	cohorts := [...]string{"1950-1964", "1965-1979", "1980-1989", "1990-1999", "2000-2005"}
	for _, gender := range [...]Gender{GenderFemale, GenderMale} {
		for _, cohort := range cohorts {
			group := firstNameGroup{gender: gender, cohort: cohort}
			if len(groups[group]) != 80 || !validTierQuota(weights[group], 20, 40, 20) {
				return nil, nil, fmt.Errorf("%w: first_names.csv group %s/%s has invalid quota", ErrCorruptFrozenDataset, gender, cohort)
			}
		}
	}
	if len(groups) != 10 {
		return nil, nil, fmt.Errorf("%w: first_names.csv has %d groups, expected 10", ErrCorruptFrozenDataset, len(groups))
	}

	return groups, weights, nil
}

func parseNameRows(name string, records [][]string, expectedRows int, hasASCIIAtom bool) ([]nameRow, []uint64, error) {

	if len(records) != expectedRows {
		return nil, nil, fmt.Errorf("%w: %s has %d rows, expected %d", ErrCorruptFrozenDataset, name, len(records), expectedRows)
	}

	rows := make([]nameRow, len(records))
	weights := make([]uint64, len(records))
	for position, record := range records {
		display := record[0]
		asciiAtom := ""
		weightField := record[1]
		if hasASCIIAtom {
			asciiAtom = record[1]
			weightField = record[2]
		}
		if !validDisplayName(display) || hasASCIIAtom && !validASCIIAtom(asciiAtom) {
			return nil, nil, fmt.Errorf("%w: %s row %d has invalid name", ErrCorruptFrozenDataset, name, position+2)
		}
		weight, err := parseTierWeight(weightField)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s row %d has invalid weight", ErrCorruptFrozenDataset, name, position+2)
		}
		rows[position] = nameRow{display: display, asciiAtom: asciiAtom, weight: weight}
		weights[position] = weight
	}

	wantOne, wantTwo, wantFour := 160, 160, 80
	if hasASCIIAtom {
		wantOne, wantTwo, wantFour = 240, 240, 120
	}
	if !validTierQuota(weights, wantOne, wantTwo, wantFour) {
		return nil, nil, fmt.Errorf("%w: %s has invalid weight quotas", ErrCorruptFrozenDataset, name)
	}

	return rows, weights, nil
}

func parsePlaceRows(records [][]string) ([]placeRow, error) {

	if len(records) != 7896 {
		return nil, fmt.Errorf("%w: places.csv has %d rows, expected 7896", ErrCorruptFrozenDataset, len(records))
	}

	places := make([]placeRow, len(records))
	known := map[string]struct{}{}
	previous := ""
	for position, record := range records {
		code := record[0]
		if !validFixedDigits(code, 6) || code <= previous {
			return nil, fmt.Errorf("%w: places.csv row %d has invalid Istat code", ErrCorruptFrozenDataset, position+2)
		}
		if _, exists := known[code]; exists {
			return nil, fmt.Errorf("%w: places.csv row %d duplicates an Istat code", ErrCorruptFrozenDataset, position+2)
		}
		if !validDisplayName(record[1]) || !validStateProv(record[2]) || !validDisplayName(record[3]) {
			return nil, fmt.Errorf("%w: places.csv row %d has invalid geography", ErrCorruptFrozenDataset, position+2)
		}
		population, err := strconv.ParseUint(record[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: places.csv row %d has invalid population", ErrCorruptFrozenDataset, position+2)
		}
		if population == 0 {
			return nil, fmt.Errorf("%w: places.csv row %d has invalid population", ErrCorruptFrozenDataset, position+2)
		}
		places[position] = placeRow{
			istatCode: code, municipality: record[1], stateProv: record[2], region: record[3], population: population,
		}
		known[code] = struct{}{}
		previous = code
	}

	return places, nil
}

func parsePostalRows(records [][]string, places []placeRow) (map[string][]string, error) {

	if len(records) != len(places) {
		return nil, fmt.Errorf("%w: postal_codes.csv has %d rows, expected %d", ErrCorruptFrozenDataset, len(records), len(places))
	}

	knownPlaces := make(map[string]struct{}, len(places))
	for _, place := range places {
		knownPlaces[place.istatCode] = struct{}{}
	}
	postalCodes := make(map[string][]string, len(records))
	for position, record := range records {
		code := record[0]
		if _, exists := knownPlaces[code]; !exists {
			return nil, fmt.Errorf("%w: postal_codes.csv row %d refers to an unknown municipality", ErrCorruptFrozenDataset, position+2)
		}
		if _, exists := postalCodes[code]; exists {
			return nil, fmt.Errorf("%w: postal_codes.csv row %d duplicates a municipality", ErrCorruptFrozenDataset, position+2)
		}
		values := strings.Split(record[1], ";")
		if len(values) == 0 || values[0] == "" || !slices.IsSorted(values) {
			return nil, fmt.Errorf("%w: postal_codes.csv row %d has invalid CAP set", ErrCorruptFrozenDataset, position+2)
		}
		for valuePosition, value := range values {
			if !validFixedDigits(value, 5) || valuePosition > 0 && value == values[valuePosition-1] {
				return nil, fmt.Errorf("%w: postal_codes.csv row %d has invalid CAP", ErrCorruptFrozenDataset, position+2)
			}
		}
		postalCodes[code] = values
	}
	for _, place := range places {
		if _, exists := postalCodes[place.istatCode]; !exists {
			return nil, fmt.Errorf("%w: postal_codes.csv has no row for %s", ErrCorruptFrozenDataset, place.istatCode)
		}
	}

	return postalCodes, nil
}

func parseTierWeight(value string) (uint64, error) {
	weight, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid tier weight")
	}
	if weight != 1 && weight != 2 && weight != 4 {
		return 0, fmt.Errorf("invalid tier weight")
	}
	return weight, nil
}
