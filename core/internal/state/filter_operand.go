// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/relvacode/iso8601"
)

// DecodeFilterOperand validates and converts an operand for typ and operator.
// It does not enforce cardinality, string limits, or allowed string values.
// Integer thresholds use 64-bit ranges; decimal thresholds use the global
// precision and scale limits, independently of the property's narrower range.
func DecodeFilterOperand(value json.Value, typ types.Type, operator WhereOperator) (any, error) {

	if !typ.Valid() || operator < OpIs || operator > OpDoesNotExist {
		return nil, errors.New("core/internal/state: invalid filter operand type or operator")
	}
	if len(value) == 0 || !json.Valid(value) {
		return nil, errors.New("filter operand must contain valid JSON")
	}
	if operator >= OpIsTrue {
		return nil, errors.New("unary filter operator cannot have operands")
	}
	value = json.TrimSpace(value)
	if typ.IsArray() {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case types.StringKind, types.UUIDKind, types.IPKind, types.DateTimeKind, types.DateKind, types.TimeKind:
		if !value.IsString() {
			return nil, fmt.Errorf("filter operand for %s must be a JSON String", typ.Kind())
		}
		text := value.String()
		switch typ.Kind() {
		case types.StringKind:
			return text, nil
		case types.UUIDKind:
			v, ok := types.NormalizeUUID(text)
			if !ok {
				return nil, errors.New("filter operand is not a valid UUID")
			}
			return v, nil
		case types.IPKind:
			v, ok := types.NormalizeIP(text)
			if !ok {
				return nil, errors.New("filter operand is not a valid IP")
			}
			return v, nil
		case types.DateTimeKind:
			v, err := iso8601.ParseString(text)
			if err != nil {
				return nil, errors.New("filter operand is not a valid datetime")
			}
			if year := v.UTC().Year(); year < types.MinYear || year > types.MaxYear {
				return nil, errors.New("filter datetime operand is out of range")
			}
			return v, nil
		case types.DateKind:
			v, err := time.Parse(time.DateOnly, text)
			if err != nil {
				return nil, errors.New("filter operand is not a valid date")
			}
			if year := v.Year(); year < types.MinYear || year > types.MaxYear {
				return nil, errors.New("filter date operand is out of range")
			}
			return v, nil
		case types.TimeKind:
			v, ok := util.ParseTime(text)
			if !ok {
				return nil, errors.New("filter operand is not a valid time")
			}
			return v, nil
		default:
			return nil, fmt.Errorf("core/internal/state: unexpected filter operand type %s", typ)
		}
	case types.IntKind, types.YearKind:
		if !value.IsNumber() {
			return nil, errors.New("integer filter operand must be a JSON Number")
		}
		if typ.IsUnsigned() {
			v, err := strconv.ParseUint(string(value), 10, 64)
			if err != nil {
				return nil, errors.New("filter operand is not a valid unsigned integer")
			}
			return uint(v), nil
		}
		v, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			return nil, errors.New("filter operand is not a valid integer")
		}
		if typ.Kind() == types.YearKind && (v < types.MinYear || v > types.MaxYear) {
			return nil, errors.New("filter year operand is out of range")
		}
		return int(v), nil
	case types.FloatKind:
		if !value.IsNumber() {
			return nil, errors.New("float filter operand must be a JSON Number")
		}
		return value.Float(typ.BitSize())
	case types.DecimalKind:
		if !value.IsNumber() {
			return nil, errors.New("decimal filter operand must be a JSON Number")
		}
		v, err := value.Decimal(0, 0)
		if err != nil {
			return nil, err
		}
		if !types.IsDecimal(v) {
			return nil, errors.New("decimal operand exceeds precision 76 or scale 37")
		}
		return v, nil
	case types.JSONKind:
		switch value.Kind() {
		case json.String:
			return json.Value(bytes.Clone(value)), nil
		case json.Number:
			if operator == OpStartsWith || operator == OpEndsWith {
				return nil, errors.New("JSON prefix and suffix operands must be strings")
			}
			v, err := value.Decimal(0, 0)
			if err != nil {
				return nil, err
			}
			if !types.IsDecimal(v) {
				return nil, errors.New("JSON number operand exceeds precision 76 or scale 37")
			}
			encoded, err := v.MarshalJSON()
			if err != nil {
				return nil, err
			}
			return json.Value(encoded), nil
		case json.True, json.False, json.Null:
			if operator == OpIsOneOf || operator == OpIsNotOneOf {
				return json.Value(bytes.Clone(value)), nil
			}
		default:
			// Object and array operands are not part of the public filter model.
		}
		return nil, errors.New("JSON filter operand has an incompatible kind")
	default:
		return nil, fmt.Errorf("core/internal/state: unsupported filter operand type %s", typ)
	}

}
