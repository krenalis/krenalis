//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package mappings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

// decode decodes a JSON-encoded data, read from r, validates it according to t
// and returns the decoded value.
// Returns error if t is not valid.
func decode(r io.Reader, t types.Type) (map[string]any, error) {
	if !t.Valid() {
		return nil, errors.New("type is not valid")
	}
	if t.PhysicalType() != types.PtObject {
		return nil, errors.New("type is not an object")
	}
	dec := json.NewDecoder(norm.NFC.Reader(r))
	dec.UseNumber()
	v, err := decodeByType(dec, nil, t)
	if err != nil {
		return nil, err
	}
	return v.(map[string]any), nil
}

// decodeByType decodes a JSON-encoded value, read from dec, validates it
// according to t and returns the decoded value.
// If tok is not nil, it does not read the first token from dec but uses tok.
func decodeByType(dec *json.Decoder, tok json.Token, t types.Type) (any, error) {
	var err error
	if tok == nil {
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			return nil, errors.New("null not allowed")
		}
	}
	switch pt := t.PhysicalType(); pt {
	case types.PtBoolean:
		b, ok := tok.(bool)
		if !ok {
			return nil, errors.New("not a Boolean value")
		}
		return b, nil
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", pt, tok)
		}
		i, err := n.Int64()
		if err != nil {
			return nil, fmt.Errorf("%s is not an %s", n, pt)
		}
		if min, max := t.IntRange(); i < min || i > max {
			return nil, fmt.Errorf("value must be in [%d,%d]", min, max)
		}
		return int(i), nil
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", pt, tok)
		}
		i, err := strconv.ParseUint(string(n), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s is not an %s", n, pt)
		}
		if min, max := t.UIntRange(); i < min || i > max {
			return nil, fmt.Errorf("value must be in [%d,%d]", min, max)
		}
		return uint(i), nil
	case types.PtFloat, types.PtFloat32:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", pt, tok)
		}
		bitSize := 64
		if pt == types.PtFloat32 {
			bitSize = 32
		}
		f, err := strconv.ParseFloat(string(n), bitSize)
		if err != nil {
			return nil, fmt.Errorf("%s is not a %s", n, pt)
		}
		if min, max := t.FloatRange(); f < min || f > max {
			return nil, fmt.Errorf("value must be in [%f,%f]", min, max)
		}
		return f, nil
	case types.PtDecimal:
		n, ok := tok.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected %s, got a %T value", pt, tok)
		}
		d, err := decimal.NewFromString(string(n))
		if err != nil {
			return nil, fmt.Errorf("%s is not a %s", n, pt)
		}
		if min, max := t.DecimalRange(); d.LessThan(min) || d.GreaterThan(max) {
			return nil, fmt.Errorf("value must be in [%s,%s]", min, max)
		}
		return d, nil
	case types.PtDateTime:
		var tm time.Time
		layout := t.Layout()
		switch layout {
		case types.Nanoseconds, types.Microseconds, types.Milliseconds, types.Seconds:
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
			case types.Nanoseconds:
				tm = time.Unix(0, d.IntPart())
			case types.Microseconds:
				tm = time.UnixMicro(d.IntPart())
			case types.Milliseconds:
				tm = time.UnixMilli(d.IntPart())
			case types.Seconds:
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
		if year := tm.Year(); year < types.MinYear || year > types.MaxYear {
			return nil, errors.New("not a DateTime value")
		}
		return tm, nil
	case types.PtDate:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a Date value")
		}
		tm, err := time.Parse(t.Layout(), s)
		if err != nil {
			return nil, errors.New("not a Date value")
		}
		tm = tm.UTC()
		if year := tm.Year(); year < types.MinYear || year > types.MaxYear {
			return nil, errors.New("not a Date value")
		}
		return tm, nil
	case types.PtTime:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a Time value")
		}
		_, err := time.Parse("15:04:05.999999999", s)
		if err != nil {
			return nil, errors.New("not a Time value")
		}
		return s, nil
	case types.PtYear:
		s, ok := tok.(json.Number)
		if !ok {
			return nil, errors.New("not an Year value")
		}
		y, err := strconv.Atoi(string(s))
		if err != nil {
			return nil, errors.New("not an Year value")
		}
		if y < types.MinYear || y > types.MaxYear {
			return nil, fmt.Errorf("year must be in [%d,%d]", types.MinYear, types.MaxYear)
		}
		return y, nil
	case types.PtUUID:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a UUID value")
		}
		_, err := uuid.Parse(s)
		if err != nil {
			return nil, errors.New("not a UUID value")
		}
		return s, nil
	case types.PtJSON:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not a JSON value")
		}
		if !json.Valid([]byte(s)) {
			return nil, errors.New("not a valid JSON value")
		}
		return s, nil
	case types.PtInet:
		s, ok := tok.(string)
		if !ok {
			return nil, errors.New("not an Inet value")
		}
		if net.ParseIP(s) == nil {
			return nil, errors.New("not a valid Inet value")
		}
		return s, nil
	case types.PtText:
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
	case types.PtArray:
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
				return nil, errors.New("null item not allowed")
			}
			item, err := decodeByType(dec, tok, it)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if len(items) < t.MinItems() {
			return nil, fmt.Errorf("array contains less than %d items", t.MinItems())
		}
		if len(items) > t.MaxItems() {
			return nil, fmt.Errorf("array contains more than %d items", t.MaxItems())
		}
		if t.Unique() && !unique(items, it) {
			return nil, errors.New("an array item is repeated")
		}
		return items, nil
	case types.PtObject:
		if tok != json.Delim('{') {
			return nil, errors.New("not a JSON object")
		}
		propertyByName := map[string]types.Property{}
		requiredProperties := map[string]struct{}{}
		for _, p := range t.Properties() {
			propertyByName[p.Name] = p
			if p.Required {
				requiredProperties[p.Name] = struct{}{}
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
				tok, err = dec.Token()
				if err != nil {
					return nil, err
				}
				if tok == nil {
					if !p.Nullable {
						return nil, fmt.Errorf("property %s cannot be null", p.Name)
					}
					object[p.Name] = nil
					delete(requiredProperties, p.Name)
					continue
				}
				object[p.Name], err = decodeByType(dec, tok, p.Type)
				if err != nil {
					return nil, err
				}
				delete(requiredProperties, p.Name)
				continue
			}
			if name == "" {
				return nil, errors.New("property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return nil, errors.New("invalid property name")
			}
			return nil, fmt.Errorf("unknown property name %q", name)
		}
		if len(requiredProperties) > 0 {
			var name string
			for p := range requiredProperties {
				if name == "" || p < name {
					name = p
				}
			}
			return nil, fmt.Errorf("required property %s not found", name)
		}
		return object, nil
	case types.PtMap:
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
			values[key], err = decodeByType(dec, nil, vt)
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		panic(fmt.Sprintf("unexpected type %d", pt))
	}

}

// unique reports whether items, with the same type t, contain unique values.
func unique(items []any, t types.Type) bool {
	n := len(items)
	if n < 2 {
		return true
	}
	if t.PhysicalType() == types.PtDecimal {
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
