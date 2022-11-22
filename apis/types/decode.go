//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// maxTime is the maximum value for a Time value.
const maxTime = 24 * 60 * 60 * 1000

// Decode decodes a JSON-encoded data read from r, validates it according
// to the given Object type and returns the decoded value.
func Decode(r io.Reader, t Type) (map[string]any, error) {
	if t.pt != PtObject {
		return nil, errors.New("t is not an Object type")
	}
	dec := json.NewDecoder(r)
	dec.UseNumber()
	v, err := decode(dec, nil, t, false)
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// DecodeStrict is like Decode but returns an error if a property is missing.
func DecodeStrict(r io.Reader, t Type) (any, error) {
	if t.pt != PtObject {
		return nil, errors.New("t is not an Object type")
	}
	dec := json.NewDecoder(r)
	dec.UseNumber()
	v, err := decode(dec, nil, t, true)
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// decode decodes a JSON-encoded value, read from dec, validates it according
// to the given type and returns the decoded value. If strict is true, it
// returns an error if a property is missing.
// If tok is not nil, it does not read the first token from dec but uses tok.
func decode(dec *json.Decoder, tok json.Token, t Type, strict bool) (any, error) {
	var err error
	if tok == nil {
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			if t.null {
				return nil, nil
			}
			return nil, errors.New("null not allowed")
		}
	}
	switch t.pt {
	case PtBoolean:
		b, ok := tok.(bool)
		if !ok {
			return nil, errors.New("not a Boolean value")
		}
		return b, nil
	case PtInt, PtInt8, PtInt16, PtInt24, PtInt64:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", t.pt, tok)
		}
		i, err := n.Int64()
		if err != nil {
			return nil, fmt.Errorf("%s is not an %s", n, t.pt)
		}
		if min, max := t.IntRange(); i < min || i > max {
			return nil, fmt.Errorf("value must be in [%d,%d]", min, max)
		}
		return int(i), nil
	case PtUInt, PtUInt8, PtUInt16, PtUInt24, PtUInt64:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", t.pt, tok)
		}
		i, err := strconv.ParseUint(string(n), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s is not an %s", n, t.pt)
		}
		if min, max := t.UIntRange(); i < min || i > max {
			return nil, fmt.Errorf("value must be in [%d,%d]", min, max)
		}
		return uint(i), nil
	case PtFloat, PtFloat32:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", t.pt, tok)
		}
		bitSize := 64
		if t.pt == PtFloat32 {
			bitSize = 32
		}
		f, err := strconv.ParseFloat(string(n), bitSize)
		if err != nil {
			return nil, fmt.Errorf("%s is not a %s", n, t.pt)
		}
		if min, max := t.FloatRange(); f < min || f > max {
			return nil, fmt.Errorf("value must be in [%f,%f]", min, max)
		}
		return f, nil
	case PtDecimal:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", t.pt, tok)
		}
		d, err := decimal.NewFromString(string(n))
		if err != nil {
			return nil, fmt.Errorf("%s is not a %s", n, t.pt)
		}
		if min, max := t.DecimalRange(); d.LessThan(min) || d.GreaterThan(d) {
			return nil, fmt.Errorf("value must be in [%s,%s]", min, max)
		}
		return d, nil
	case PtDateTime:
		var tm time.Time
		layout := t.TimeLayout()
		switch layout {
		case Nanoseconds, Microseconds, Milliseconds, Seconds:
			var s string
			switch v := tok.(type) {
			case string:
				s = v
			case json.Number:
				s = string(v)
			}
			d, err := decimal.NewFromString(s)
			if err != nil {
				return nil, errors.New("not a DateTime value")
			}
			switch layout {
			case Nanoseconds:
				tm = time.Unix(0, d.IntPart())
			case Microseconds:
				tm = time.UnixMicro(d.IntPart())
			case Milliseconds:
				tm = time.UnixMilli(d.IntPart())
			case Seconds:
				tm = time.Unix(d.IntPart(), 0)
			}
		default:
			s, ok := tok.(string)
			if !ok {
				return nil, errors.New("not a DateTime value")
			}
			tm, err = time.Parse(layout, s)
			if err != nil {
				return nil, errors.New("not a DateTime value")
			}
		}
		tm = tm.UTC()
		if year := tm.Year(); year < MinYear || year > MaxYear {
			return nil, errors.New("not a DateTime value")
		}
		return tm, nil
	case PtDate:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a Date value")
		}
		tm, err := time.Parse(t.TimeLayout(), s)
		if err != nil {
			return nil, errors.New("not a Date value")
		}
		tm = tm.UTC()
		if year := tm.Year(); year < MinYear || year > MaxYear {
			return nil, errors.New("not a Date value")
		}
		return tm, nil
	case PtTime:
		var tm int
		layout := t.TimeLayout()
		switch layout {
		case Nanoseconds, Microseconds, Milliseconds, Seconds:
			var s string
			switch v := tok.(type) {
			case string:
				s = v
			case json.Number:
				s = string(v)
			}
			d, err := decimal.NewFromString(s)
			if err != nil || d.IsNegative() {
				return nil, errors.New("not a Time value")
			}
			switch layout {
			case Nanoseconds:
				d = d.Div(decimal.NewFromInt(1_000_000))
			case Microseconds:
				d = d.Div(decimal.NewFromInt(1_000))
			case Seconds:
				d = d.Mul(decimal.NewFromInt(1_000))
			}
			tm = int(d.IntPart())
		default:
			s, ok := tok.(string)
			if !ok {
				return nil, errors.New("not a Time value")
			}
			tp, err := time.Parse(layout, s)
			if err != nil {
				return nil, errors.New("not a Time value")
			}
			tm = int(tp.Sub(time.Date(tp.Year(), tp.Month(), tp.Day(), 0, 0, 0, 0, time.UTC)).Milliseconds())
		}
		if tm > maxTime {
			return nil, errors.New("Time values must be less than 24h")
		}
		return tm, nil
	case PtYear:
		s, ok := tok.(json.Number)
		if !ok {
			return nil, errors.New("not an Year value")
		}
		y, err := strconv.Atoi(string(s))
		if err != nil {
			return nil, errors.New("not an Year value")
		}
		if y < MinYear || y > MaxYear {
			return nil, fmt.Errorf("year must be in [%d,%d]", MinYear, MaxYear)
		}
		return y, nil
	case PtUUID:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a UUID value")
		}
		_, err := uuid.Parse(s)
		if err != nil {
			return nil, errors.New("not a UUID value")
		}
		return s, nil
	case PtJSON:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a JSON value")
		}
		if !json.Valid([]byte(s)) {
			return nil, errors.New("not a valid JSON value")
		}
		return s, nil
	case PtText:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a Text value")
		}
		if l, ok := t.ByteLen(); ok && len(s) > l {
			return nil, fmt.Errorf("is more than %d bytes long", l)
		}
		if l, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > l {
			return nil, fmt.Errorf("is more than %d characters long", l)
		}
		return s, nil
	case PtArray:
		if d, ok := tok.(json.Delim); !ok || d != '[' {
			return nil, errors.New("not an array value")
		}
		it := t.ItemType()
		items := []any{}
		tok = nil
		for {
			tok, err = dec.Token()
			if err != nil {
				return nil, err
			}
			if d, ok := tok.(json.Delim); ok && d == ']' {
				return items, nil
			}
			if tok == nil {
				if it.null {
					items = append(items, nil)
					continue
				}
				return nil, errors.New("null item not allowed")
			}
			item, err := decode(dec, tok, it, strict)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	case PtObject:
		if d, ok := tok.(json.Delim); !ok || d != '{' {
			return nil, errors.New("not an object value")
		}
		propertyByName := map[string]Property{}
		for _, p := range t.vl.([]Property) {
			propertyByName[p.Name] = p
		}
		object := map[string]any{}
	Property:
		for {
			tok, err = dec.Token()
			if err != nil {
				return nil, err
			}
			name, ok := tok.(string)
			if !ok {
				return object, nil
			}
			if name == "" {
				return nil, errors.New("unexpected empty property name")
			}
			p, ok := propertyByName[name]
			if !ok {
				if strict {
					return nil, fmt.Errorf("missing property %q", name)
				}
				// Skip the property.
				depth := 0
				for {
					tok, err = dec.Token()
					if err != nil {
						return nil, err
					}
					if d, ok := tok.(json.Delim); ok {
						switch d {
						case '{', '[':
							depth++
						case '}', ']':
							depth--
						}
					}
					if depth == 0 {
						continue Property
					}
				}
			}
			value, err := decode(dec, nil, p.Type, strict)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
	}

	return nil, nil
}
