// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"time"
)

type affinePermutation struct {
	a uint64
	b uint64
}

func (p affinePermutation) value(index PersonIndex) uint64 {
	return (p.a*(uint64(index)-1) + p.b) % emailAndSiteTokenModulus
}

type weightedStringTable struct {
	values  []string
	weights []uint64
}

func ageAt(birthDate, referenceDate time.Time) int {
	age := referenceDate.Year() - birthDate.Year()
	anniversaryDay := birthDate.Day()
	if birthDate.Month() == time.February && anniversaryDay == 29 && !isLeapYear(referenceDate.Year()) {
		anniversaryDay = 28
	}
	anniversary := time.Date(referenceDate.Year(), birthDate.Month(), anniversaryDay, 0, 0, 0, 0, time.UTC)
	if referenceDate.Before(anniversary) {
		age--
	}
	return age
}

func cohortForYear(year int) (string, error) {
	switch {
	case year >= 1950 && year <= 1964:
		return "1950-1964", nil
	case year >= 1965 && year <= 1979:
		return "1965-1979", nil
	case year >= 1980 && year <= 1989:
		return "1980-1989", nil
	case year >= 1990 && year <= 1999:
		return "1990-1999", nil
	case year >= 2000 && year <= 2005:
		return "2000-2005", nil
	default:
		return "", fmt.Errorf("%w: birth year %d has no first-name cohort", ErrCorruptFrozenDataset, year)
	}
}

func deriveAffinePermutation(factory streamFactory, path string) (affinePermutation, error) {
	rng, err := factory.worldStream(path)
	if err != nil {
		return affinePermutation{}, err
	}
	a, err := rng.Uint64n(emailAndSiteTokenModulus)
	if err != nil {
		return affinePermutation{}, err
	}
	for greatestCommonDivisor(a, emailAndSiteTokenModulus) != 1 {
		a++
		if a == emailAndSiteTokenModulus {
			a = 0
		}
	}
	b, err := rng.Uint64n(emailAndSiteTokenModulus)
	if err != nil {
		return affinePermutation{}, err
	}
	return affinePermutation{a: a, b: b}, nil
}

func encodeBase36Fixed5(value uint64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	var encoded [5]byte
	for position := len(encoded) - 1; position >= 0; position-- {
		encoded[position] = alphabet[value%36]
		value /= 36
	}
	return string(encoded[:])
}

func greatestCommonDivisor(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func parseReferenceDate(value string) (time.Time, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, ErrInvalidReferenceDate
	}
	if parsed.Format(time.DateOnly) != value {
		return time.Time{}, ErrInvalidReferenceDate
	}
	minimum := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	maximum := time.Date(2030, time.December, 31, 0, 0, 0, 0, time.UTC)
	if parsed.Before(minimum) || parsed.After(maximum) {
		return time.Time{}, ErrInvalidReferenceDate
	}
	return parsed, nil
}

func phoneForIndex(index PersonIndex) string {
	x := uint64(index) - 1
	y := (phoneAffineA*x + phoneAffineB) % phoneModulus
	return fmt.Sprintf("%s%09d", phonePrefix, y+phoneTokenOffset)
}
