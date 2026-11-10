// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

// Keep this file in sync between the warehouse and the connector.

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/json"
)

// plainJSONNumber expands a valid scientific JSON number only when its exact
// value fits NUMBER's maximum precision 38 and scale 37. A nil result leaves
// the original token unchanged; no floating-point conversion is performed.
func plainJSONNumber(text string) []byte {

	e := strings.IndexAny(text, "eE")
	if e < 0 {
		return nil
	}
	mantissa := text[:e]
	negative := mantissa[0] == '-'
	if negative {
		mantissa = mantissa[1:]
	}
	first, last, digits, leading, trailing := -1, -1, 0, 0, 0
	for i := range len(mantissa) {
		if mantissa[i] == '.' {
			continue
		}
		digits++
		if mantissa[i] == '0' {
			trailing++
			if first < 0 {
				leading++
			}
		} else {
			if first < 0 {
				first = i
			}
			last, trailing = i, 0
		}
	}
	if first < 0 {
		if negative {
			return []byte("-0")
		}
		return []byte("0")
	}
	significant := digits - leading - trailing
	if significant > 38 {
		return nil
	}
	fraction := 0
	if point := strings.IndexByte(mantissa, '.'); point >= 0 {
		fraction = len(mantissa) - point - 1
	}
	exponent, err := strconv.ParseInt(text[e+1:], 10, 64)
	if err != nil {
		return nil
	}
	// Bound the exponent before adding the mantissa's offset, including cases
	// where a long mantissa cancels a large exponent.
	offset := int64(trailing - fraction)
	if exponent < -37-offset || exponent > 38-offset {
		return nil
	}
	shift := int(exponent + offset)
	scale := max(-shift, 0)
	integer := significant + shift
	if scale > 37 || max(integer, 0)+scale > 38 {
		return nil
	}

	result := make([]byte, 0, 40)
	if negative {
		result = append(result, '-')
	}
	if integer <= 0 {
		result = append(result, '0', '.')
		for range -integer {
			result = append(result, '0')
		}
	}
	position := 0
	for i := first; i <= last; i++ {
		if mantissa[i] == '.' {
			continue
		}
		if position == integer && integer > 0 {
			result = append(result, '.')
		}
		result = append(result, mantissa[i])
		position++
	}
	for position < integer {
		result = append(result, '0')
		position++
	}

	return result
}

// plainJSONNumbers preserves the document's bytes except for scientific
// number tokens that can be stored exactly as INTEGER/DECIMAL in Snowflake.
// Values outside that domain retain the existing loading behavior.
func plainJSONNumbers(value json.Value) (json.Value, error) {

	// A numeric exponent always follows a digit. Strings can cause harmless
	// false positives, but documents without such a pair need no decoding.
	hasExponent := false
	for i := 1; i < len(value); i++ {
		if (value[i] == 'e' || value[i] == 'E') && value[i-1] >= '0' && value[i-1] <= '9' {
			hasExponent = true
			break
		}
	}
	if !hasExponent {
		return value, nil
	}
	decoder := json.NewDecoder(bytes.NewBuffer(value))
	var result json.Value
	copied := 0
	for {
		token, err := decoder.ReadToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("warehouses/snowflake: invalid JSON value for CSV serialization")
		}
		if token.Kind() != json.Number {
			continue
		}
		text := token.String()
		plain := plainJSONNumber(text)
		if plain == nil {
			continue
		}
		end := int(decoder.ByteOffset())
		result = append(result, value[copied:end-len(text)]...)
		result = append(result, plain...)
		copied = end
	}
	if result == nil {
		return value, nil
	}
	result = append(result, value[copied:]...)

	return result, nil
}
