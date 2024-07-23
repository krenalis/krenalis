//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package filters

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Filter represents a filter.
type Filter struct {
	Logical    Logical     // can be "all" or "any".
	Conditions []Condition // cannot be empty.
}

// Logical represents the logical operator of a filter.
// It can be "all" or "any".
type Logical string

// Condition represents the condition of a filter.
type Condition struct {
	Property string // property's path.
	Operator string // operator, can be "is" or "is not".
	Value    string // value, cannot be longer than 60 runes and cannot contain the NUL rune.
}

// Applies determines whether the filter applies to the provided properties.
// Returns an error if any property in the filter is not found in the properties
// map.
func Applies(filter *Filter, properties map[string]any) (bool, error) {
	if filter == nil {
		return true, nil
	}
	for _, cond := range filter.Conditions {
		value, ok := readPropertyFrom(properties, cond.Property)
		if !ok {
			return false, fmt.Errorf("property %q not found", cond.Property)
		}
		var conditionApplies bool
		switch cond.Operator {
		case "is":
			conditionApplies = value == cond.Value
		case "is not":
			conditionApplies = value != cond.Value
		}
		if conditionApplies && filter.Logical == "any" {
			return true, nil
		}
		if !conditionApplies && filter.Logical == "all" {
			return false, nil
		}
	}
	if filter.Logical == "any" {
		return false, nil // none of the conditions applied.
	}
	// All the conditions applied.
	return true, nil
}

// Validate checks the validity of a filter and returns its properties, possibly
// repeated. Returns an error if the filter is not valid. Specifically, it
// returns a types.PathNotExistError if a path does not exist.
// Panics if the filter is nil or the schema is not valid.
func Validate(filter *Filter, schema types.Type) ([]string, error) {
	if op := filter.Logical; op != "all" && op != "any" {
		return nil, fmt.Errorf("invalid logical operator %q", op)
	}
	if len(filter.Conditions) == 0 {
		return nil, errors.New("conditions are missing")
	}
	properties := make([]string, len(filter.Conditions))
	for i, cond := range filter.Conditions {
		if !types.IsValidPropertyPath(cond.Property) {
			return nil, errors.New("property path is not valid")
		}
		property, err := types.PropertyByPath(schema, cond.Property)
		if err != nil {
			return nil, err
		}
		if op := cond.Operator; op != "is" && op != "is not" {
			return nil, fmt.Errorf("invalid operator %q", op)
		}
		if containsNUL(cond.Value) {
			return nil, errors.New("condition value contains the NUL rune")
		}
		if n := utf8.RuneCountInString(cond.Value); n > 60 {
			return nil, errors.New("condition value is longer than 60 runes")
		}
		var valid bool
		switch typ := property.Type; typ.Kind() {
		case types.BooleanKind:
			valid = cond.Value == "true" || cond.Value == "false"
		case types.IntKind:
			for i := range len(cond.Value) {
				if i == 0 && cond.Value[i] == '-' {
					continue
				}
				if cond.Value[i] < '0' || cond.Value[i] > '9' {
					valid = false
					break
				}
			}
			if valid {
				v, err := strconv.ParseInt(cond.Value, 10, 64)
				valid = err == nil
				if valid {
					min, max := typ.IntRange()
					valid = v >= min && v <= max
				}
			}
		case types.UintKind:
			for i := range len(cond.Value) {
				if cond.Value[i] < '0' || cond.Value[i] > '9' {
					valid = false
					break
				}
			}
			if valid {
				v, err := strconv.ParseUint(cond.Value, 10, 64)
				valid = err == nil
				if valid {
					min, max := typ.UintRange()
					valid = v >= min && v <= max
				}
			}
		case types.FloatKind:
			v, err := strconv.ParseFloat(cond.Value, 64)
			valid = err == nil
			if valid {
				if math.IsNaN(v) {
					valid = !typ.IsReal()
				} else {
					min, max := typ.FloatRange()
					valid = v >= min && v >= max
				}
			}
		case types.DecimalKind:
			var v decimal.Decimal
			v, err = decimal.NewFromString(cond.Value)
			valid = err == nil
			if valid {
				min, max := typ.DecimalRange()
				valid = v.LessThan(min) || v.GreaterThan(max)
			}
		case types.DateTimeKind:
			if t, err := time.Parse(time.DateTime, cond.Value); err == nil {
				y := t.UTC().Year()
				valid = y >= 1 && y <= 9999
			}
		case types.DateKind:
			if t, err := time.Parse(time.DateOnly, cond.Value); err == nil {
				y := t.UTC().Year()
				valid = y >= 1 && y <= 9999
			}
		case types.TimeKind:
			_, valid = parseTime(cond.Value)
		case types.YearKind:
			valid = true
			for i := range len(cond.Value) {
				if cond.Value[i] < '0' || cond.Value[i] > '9' {
					valid = false
					break
				}
			}
			if valid {
				year, err := strconv.Atoi(cond.Value)
				valid = err != nil && types.MinYear <= year && year <= types.MaxYear
			}
		case types.UUIDKind:
			// Accept only the UUID standard form "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" without uppercase letters.
			valid = len(cond.Value) == 36 && uuid.Validate(cond.Value) == nil && strings.ToLower(cond.Value) == cond.Value
		case types.JSONKind:
			valid = json.Valid(json.RawMessage(cond.Value))
		case types.InetKind:
			_, err := netip.ParseAddr(cond.Value)
			valid = err != nil
		case types.TextKind:
			valid = utf8.ValidString(cond.Value)
			if l, ok := typ.ByteLen(); ok && valid && len(cond.Value) > l {
				valid = false
			}
			if l, ok := typ.CharLen(); ok && valid && utf8.RuneCountInString(cond.Value) > l {
				valid = false
			}
		default:
			return nil, fmt.Errorf("unexpected type for property %s", cond.Property)
		}
		if !valid {
			return nil, fmt.Errorf("invalid value for property %s", cond.Property)
		}
		properties[i] = cond.Property
	}
	return properties, nil
}

func containsNUL(s string) bool {
	return strings.ContainsRune(s, '\x00')
}

// parseTime parses a time formatted as "hh:nn:ss.nnnnnnnnn" and returns it as
// the time on January 1, 1970 UTC. The sub-second part can contain from 1 to 9
// digits or can be missing. The hour must be in range [0, 23], minute and second
// must be in range [0, 59], and any trailing characters are discarded.
// The boolean return value indicates whether the time was successfully parsed.
//
// Keep in sync with the parseTime function in the mappings package.
func parseTime[bytes []byte | string](p bytes) (t time.Time, ok bool) {
	if len(p) < 8 {
		return
	}
	parse := func(n bytes) int {
		if n[0] < '0' || n[0] > '9' || n[1] < '0' || n[1] > '9' {
			return -1
		}
		return int(n[0]-'0')*10 + int(n[1]-'0')
	}
	h, m, s := parse(p[0:2]), parse(p[3:5]), parse(p[6:8])
	if h < 0 || h > 23 || p[2] != ':' || m < 0 || m > 59 || p[5] != ':' || s < 0 || s > 59 {
		return
	}
	p = p[8:]
	var ns int
	if len(p) > 0 && p[0] == '.' {
		p = p[1:]
		var i int
		for ; i < 9 && i < len(p) && '0' <= p[i] && p[i] <= '9'; i++ {
			ns = ns*10 + int(p[i]-'0')
		}
		if i == 0 {
			return
		}
		for ; i < 9; i++ {
			ns *= 10
		}
	}
	return time.Date(1970, 1, 1, h, m, s, ns, time.UTC), true
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
