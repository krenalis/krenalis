//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

// maxTime is the maximum value for a Time value.
const maxTime = 24 * 60 * 60 * 1000

// Decode decodes a JSON-encoded data, validates it according to schema and
// returns the decoded value.
// Panics is schema is not valid.
func Decode(data []byte, schema Schema) (map[string]any, error) {
	if !schema.Valid() {
		return nil, errors.New("schema is not valid")
	}
	dec := json.NewDecoder(bytes.NewReader(norm.NFC.Bytes(data)))
	dec.UseNumber()
	v, err := decodeBySchema(dec, schema, false)
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// DecodeStrict is like Decode but returns an error if a property of the schema
// or a property of an object is missing.
func DecodeStrict(data []byte, schema Schema) (any, error) {
	if !schema.Valid() {
		return nil, errors.New("schema is not valid")
	}
	dec := json.NewDecoder(bytes.NewReader(norm.NFC.Bytes(data)))
	dec.UseNumber()
	v, err := decodeBySchema(dec, schema, true)
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// decodeBySchema decodes a JSON-encoded value, read from dec, validates it
// according to schema and returns the decoded value. If strict is true, it
// returns an error if a property is missing.
func decodeBySchema(dec *json.Decoder, schema Schema, strict bool) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	propertyByName := map[string]Property{}
	for _, p := range schema.properties {
		propertyByName[p.Name] = p
		for _, alias := range p.Aliases {
			propertyByName[alias] = p
		}
	}
	object := map[string]any{}
	for {
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		name, ok := tok.(string)
		if !ok {
			break
		}
		if p, ok := propertyByName[name]; ok {
			object[p.Name], err = decodeByType(dec, nil, p.Type, strict)
			if err != nil {
				return nil, err
			}
			continue
		}
		if name == "" {
			return nil, errors.New("property name is empty")
		}
		if !IsValidPropertyName(name) {
			return nil, errors.New("invalid property name")
		}
		if strict {
			return nil, fmt.Errorf("unknow property name %q", name)
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
				break
			}
		}
	}
	return object, nil
}

// decodeByType decodes a JSON-encoded value, read from dec, validates it
// according to t and returns the decoded value. If strict is true, it returns
// an error if an object property is missing.
// If tok is not nil, it does not read the first token from dec but uses tok.
func decodeByType(dec *json.Decoder, tok json.Token, t Type, strict bool) (any, error) {
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
		layout := t.Layout()
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
		tm, err := time.Parse(t.Layout(), s)
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
		layout := t.Layout()
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
		if tok != json.Delim('[') {
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
			if tok == json.Delim(']') {
				break
			}
			if tok == nil {
				if it.null {
					items = append(items, nil)
					continue
				}
				return nil, errors.New("null item not allowed")
			}
			item, err := decodeByType(dec, tok, it, strict)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if len(items) < int(t.p) {
			return nil, fmt.Errorf("array contains less than %d items", t.p)
		}
		if len(items) > int(t.s) {
			return nil, fmt.Errorf("array contains more than %d items", t.s)
		}
		if t.unique && !unique(items, it) {
			return nil, errors.New("an array item is repeated")
		}
		return items, nil
	case PtObject:
		if tok != json.Delim('{') {
			return nil, errors.New("not a JSON object")
		}
		propertyByName := map[string]ObjectProperty{}
		for _, p := range t.vl.([]ObjectProperty) {
			propertyByName[p.Name] = p
			for _, alias := range p.Aliases {
				propertyByName[alias] = p
			}
		}
		object := map[string]any{}
		for {
			tok, err = dec.Token()
			if err != nil {
				return nil, err
			}
			name, ok := tok.(string)
			if !ok {
				break
			}
			if p, ok := propertyByName[name]; ok {
				object[p.Name], err = decodeByType(dec, nil, p.Type, strict)
				if err != nil {
					return nil, err
				}
				continue
			}
			if name == "" {
				return nil, errors.New("property name is empty")
			}
			if !IsValidPropertyName(name) {
				return nil, errors.New("invalid property name")
			}
			if strict {
				return nil, fmt.Errorf("unknow property name %q", name)
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
					break
				}
			}
		}
		return object, nil
	case PtMap:
		if tok != json.Delim('{') {
			return nil, errors.New("not a JSON object")
		}
		vt := t.ValueType()
		values := map[string]any{}
		for {
			tok, err = dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := tok.(string)
			if !ok {
				break
			}
			key = norm.NFC.String(key)
			if _, ok := values[key]; ok {
				return nil, errors.New("repeated map key")
			}
			values[key], err = decodeByType(dec, nil, vt, strict)
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		panic(fmt.Sprintf("unexpected type %d", t.pt))
	}

}

// unique reports whether items, with the same type t, contain unique values.
func unique(items []any, t Type) bool {
	n := len(items)
	if n < 2 {
		return true
	}
	if t.pt == PtDecimal {
		for i, a := range items[:n-1] {
			a, ok := a.(decimal.Decimal)
			if ok {
				for _, b := range items[i+1:] {
					if b != nil && !a.Equals(b.(decimal.Decimal)) {
						return false
					}
				}
			} else {
				for _, b := range items[i+1:] {
					if b == nil {
						return false
					}
				}
			}
		}
		return true
	}
	for i, a := range items[:n-1] {
		for _, b := range items[i+1:] {
			if a == b {
				return false
			}
		}
	}
	return true
}
