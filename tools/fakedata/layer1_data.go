// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"strings"
	"unicode/utf8"
)

const (
	ageDatasetComponentName        = "age-data-sha256"
	firstNameDatasetComponentName  = "first-name-data-sha256"
	lastNameDatasetComponentName   = "last-name-data-sha256"
	placesDatasetComponentName     = "places-data-sha256"
	postalDatasetComponentName     = "postal-data-sha256"
	streetNameDatasetComponentName = "street-name-data-sha256"

	placeWeightBucketSize = 128
)

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
