// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	ageDatasetComponentName        = "age-data-sha256"
	firstNameDatasetComponentName  = "first-name-data-sha256"
	lastNameDatasetComponentName   = "last-name-data-sha256"
	placesDatasetComponentName     = "places-data-sha256"
	postalDatasetComponentName     = "postal-data-sha256"
	streetNameDatasetComponentName = "street-name-data-sha256"

	ageWeightsSHA256  = "5abdc04bb817e5fc300ce71920c1dedf269551d3cc0065148264d26a39c84555"
	firstNamesSHA256  = "ed249d8095d3133d90a662bb3dc4d592941d9aa1de1d476a8000b392761ed8db"
	lastNamesSHA256   = "d44f1859847b5c94066d20c0846fb16859827802bdf878d59e17dc26a046f96b"
	placesSHA256      = "27539c5025412c2f03b199cbd6fdea7b9bda8fb57f6d8665883c70eeaa7b05b6"
	postalCodesSHA256 = "0fed3e2822c13a54c4255d2b83fe0fd63aea27b432d63a682b28f64f4981426c"
	streetNamesSHA256 = "13a757c70266ad96f0c1f9d76ff32597995090009e860eec86a1dded5b5095ec"

	placeWeightBucketSize = 128
)

var loadLayer1ItalyData = sync.OnceValues(parseLayer1ItalyData)

//go:embed data/it-2026a/*.csv
var layer1ItalyFiles embed.FS

type ageRow struct {
	age    int
	weight uint64
}

type firstNameRow struct {
	display   string
	asciiAtom string
	weight    uint64
}

type italyData struct {
	ages                    []ageRow
	ageWeights              []uint64
	firstNames              map[firstNameGroup][]firstNameRow
	firstNameWeights        map[firstNameGroup][]uint64
	lastNames               []nameRow
	lastNameWeights         []uint64
	places                  []placeRow
	placeWeightBuckets      []weightedBucket
	placeWeightBucketTotals []uint64
	postalCodes             map[string][]string
	streetNames             []nameRow
	streetNameWeights       []uint64
	ageDependency           Component
	firstNameDependency     Component
	lastNameDependency      Component
	placesDependency        Component
	postalDependency        Component
	streetNameDependency    Component
}

type firstNameGroup struct {
	gender Gender
	cohort string
}

type nameRow struct {
	display   string
	asciiAtom string
	weight    uint64
}

type placeRow struct {
	istatCode    string
	municipality string
	stateProv    string
	region       string
	population   uint64
}

type weightedBucket struct {
	start   int
	weights []uint64
}

func buildPlaceWeightBuckets(places []placeRow) ([]weightedBucket, []uint64) {
	count := (len(places) + placeWeightBucketSize - 1) / placeWeightBucketSize
	buckets := make([]weightedBucket, 0, count)
	totals := make([]uint64, 0, count)
	for start := 0; start < len(places); start += placeWeightBucketSize {
		end := min(start+placeWeightBucketSize, len(places))
		weights := make([]uint64, end-start)
		var total uint64
		for position := start; position < end; position++ {
			weights[position-start] = places[position].population
			total += places[position].population
		}
		buckets = append(buckets, weightedBucket{start: start, weights: weights})
		totals = append(totals, total)
	}
	return buckets, totals
}

func datasetDependency(name, digest string) (Component, error) {
	raw, err := hex.DecodeString(digest)
	if err != nil {
		return Component{}, fmt.Errorf("%w: invalid embedded checksum for %s", ErrCorruptFrozenDataset, name)
	}
	if len(raw) != sha256.Size {
		return Component{}, fmt.Errorf("%w: invalid embedded checksum for %s", ErrCorruptFrozenDataset, name)
	}
	component, err := BytesComponent(name, raw)
	if err != nil {
		return Component{}, fmt.Errorf("%w: dependency %s: %v", ErrCorruptFrozenDataset, name, err)
	}
	return component, nil
}

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

func parseLayer1CSV(name, expectedDigest string, expectedHeader []string) ([][]string, error) {

	path := "data/it-2026a/" + name
	data, err := layer1ItalyFiles.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrCorruptFrozenDataset, name, err)
	}
	actualDigest := sha256.Sum256(data)
	if hex.EncodeToString(actualDigest[:]) != expectedDigest {
		return nil, fmt.Errorf("%w: %s checksum mismatch", ErrCorruptFrozenDataset, name)
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || !bytes.HasSuffix(data, []byte{'\n'}) || bytes.Contains(data, []byte{'\r'}) {
		return nil, fmt.Errorf("%w: %s has invalid byte format", ErrCorruptFrozenDataset, name)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", ErrCorruptFrozenDataset, name, err)
	}
	if len(records) == 0 || !slices.Equal(records[0], expectedHeader) {
		return nil, fmt.Errorf("%w: %s has invalid header", ErrCorruptFrozenDataset, name)
	}

	return records[1:], nil
}

func parseLayer1ItalyData() (*italyData, error) {

	ageRecords, err := parseLayer1CSV("age_weights.csv", ageWeightsSHA256, []string{"age", "weight"})
	if err != nil {
		return nil, err
	}
	firstNameRecords, err := parseLayer1CSV(
		"first_names.csv", firstNamesSHA256, []string{"gender", "cohort", "name", "ascii_atom", "weight"},
	)
	if err != nil {
		return nil, err
	}
	lastNameRecords, err := parseLayer1CSV("last_names.csv", lastNamesSHA256, []string{"name", "ascii_atom", "weight"})
	if err != nil {
		return nil, err
	}
	placeRecords, err := parseLayer1CSV(
		"places.csv", placesSHA256, []string{"istat_code", "city", "state_prov", "region", "population"},
	)
	if err != nil {
		return nil, err
	}
	postalRecords, err := parseLayer1CSV("postal_codes.csv", postalCodesSHA256, []string{"istat_code", "postal_codes"})
	if err != nil {
		return nil, err
	}
	streetNameRecords, err := parseLayer1CSV("street_names.csv", streetNamesSHA256, []string{"name", "weight"})
	if err != nil {
		return nil, err
	}

	ages, ageWeights, err := parseAgeRows(ageRecords)
	if err != nil {
		return nil, err
	}
	firstNames, firstNameWeights, err := parseFirstNameRows(firstNameRecords)
	if err != nil {
		return nil, err
	}
	lastNames, lastNameWeights, err := parseNameRows("last_names.csv", lastNameRecords, 600, true)
	if err != nil {
		return nil, err
	}
	places, err := parsePlaceRows(placeRecords)
	if err != nil {
		return nil, err
	}
	postalCodes, err := parsePostalRows(postalRecords, places)
	if err != nil {
		return nil, err
	}
	streetNames, streetNameWeights, err := parseNameRows("street_names.csv", streetNameRecords, 400, false)
	if err != nil {
		return nil, err
	}

	ageDependency, err := datasetDependency(ageDatasetComponentName, ageWeightsSHA256)
	if err != nil {
		return nil, err
	}
	firstNameDependency, err := datasetDependency(firstNameDatasetComponentName, firstNamesSHA256)
	if err != nil {
		return nil, err
	}
	lastNameDependency, err := datasetDependency(lastNameDatasetComponentName, lastNamesSHA256)
	if err != nil {
		return nil, err
	}
	placesDependency, err := datasetDependency(placesDatasetComponentName, placesSHA256)
	if err != nil {
		return nil, err
	}
	postalDependency, err := datasetDependency(postalDatasetComponentName, postalCodesSHA256)
	if err != nil {
		return nil, err
	}
	streetNameDependency, err := datasetDependency(streetNameDatasetComponentName, streetNamesSHA256)
	if err != nil {
		return nil, err
	}
	placeWeightBuckets, placeWeightBucketTotals := buildPlaceWeightBuckets(places)

	return &italyData{
		ages: ages, ageWeights: ageWeights,
		firstNames: firstNames, firstNameWeights: firstNameWeights,
		lastNames: lastNames, lastNameWeights: lastNameWeights,
		places: places, placeWeightBuckets: placeWeightBuckets, placeWeightBucketTotals: placeWeightBucketTotals,
		postalCodes: postalCodes, streetNames: streetNames, streetNameWeights: streetNameWeights,
		ageDependency: ageDependency, firstNameDependency: firstNameDependency,
		lastNameDependency: lastNameDependency, placesDependency: placesDependency,
		postalDependency: postalDependency, streetNameDependency: streetNameDependency,
	}, nil
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

func validASCIIAtom(value string) bool {
	if value == "" || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, c := range []byte(value) {
		if (c < 'a' || c > 'z') && c != '-' {
			return false
		}
	}
	return true
}

func validCohort(value string) bool {
	switch value {
	case "1950-1964", "1965-1979", "1980-1989", "1990-1999", "2000-2005":
		return true
	default:
		return false
	}
}

func validDisplayName(value string) bool {
	return value != "" && utf8.ValidString(value) && strings.TrimSpace(value) == value
}

func validFixedDigits(value string, width int) bool {
	if len(value) != width {
		return false
	}
	for _, c := range []byte(value) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validStateProv(value string) bool {
	return len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z'
}

func validTierQuota(weights []uint64, wantOne, wantTwo, wantFour int) bool {
	counts := map[uint64]int{}
	for _, weight := range weights {
		counts[weight]++
	}
	return counts[1] == wantOne && counts[2] == wantTwo && counts[4] == wantFour && len(counts) == 3
}
