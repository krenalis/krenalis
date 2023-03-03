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
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

// Resolver resolves a custom type name to its type. If a custom type with the
// given name does not exist, it returns a ErrCustomTypeNotExist error.
type Resolver func(name string) (Type, error)

// A ErrCustomTypeNotExist error is returned by a Resolver function when the
// custom type to resolve does not exist.
var ErrCustomTypeNotExist = errors.New("custom type does not exist")

var null = []byte("null")

// Marshal marshals t into JSON.
// It returns an error is t is not valid.
func Marshal(t Type) ([]byte, error) {
	if !t.Valid() {
		return nil, errors.New("type is not valid")
	}
	var b bytes.Buffer
	marshalType(&b, t, true)
	return b.Bytes(), nil
}

// Parse parses the JSON-encoded data and returns the decoded type. For custom
// types, it calls the resolve function, if not nil, to resolve the type custom
// name to its type.
func Parse(data string, resolve Resolver) (Type, error) {
	dec := json.NewDecoder(strings.NewReader(norm.NFC.String(data)))
	dec.UseNumber()
	t, err := unmarshalType(dec, resolve)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return t, err
}

// MarshalJSON marshals t into JSON.
// It returns an error is t is not valid.
func (t Type) MarshalJSON() ([]byte, error) {
	if !t.Valid() {
		return nil, errors.New("type is not valid")
	}
	var b bytes.Buffer
	marshalType(&b, t, true)
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the type
// pointed by t.
func (t *Type) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(norm.NFC.Bytes(data)))
	dec.UseNumber()
	t2, err := unmarshalType(dec, nil)
	if err != nil {
		return err
	}
	*t = t2
	return nil
}

// marshalType marshals t as JSON and writes it to b.
func marshalType(b *bytes.Buffer, t Type, custom bool) {
	if t.custom != "" {
		if !custom {
			marshalString(b, t.custom)
			return
		}
		b.WriteString(`{"custom":"`)
		b.WriteString(t.custom)
		b.WriteString(`",`)
	} else {
		b.WriteByte('{')
	}
	b.WriteString(`"name":"`)
	b.WriteString(t.pt.String())
	b.WriteString(`"`)
	if t.lt > 0 {
		b.WriteString(`,"logical":"`)
		b.WriteString(t.lt.String())
		b.WriteString(`"`)
	}
	switch t.pt {
	case PtInt, PtInt8, PtInt16, PtInt24:
		if min := int64(t.p); min > minInt[t.pt-PtInt] {
			b.WriteString(`,"minimum":`)
			b.WriteString(strconv.FormatInt(min, 10))
		}
		if max := int64(t.s); max < maxInt[t.pt-PtInt] {
			b.WriteString(`,"maximum":`)
			b.WriteString(strconv.FormatInt(max, 10))
		}
	case PtInt64:
		if i, ok := t.vl.(intRange); ok {
			if i.min > MinInt64 {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatInt(i.min, 10))
			}
			if i.max < MaxInt64 {
				b.WriteString(`,"maximum":`)
				b.WriteString(strconv.FormatInt(i.max, 10))
			}
		}
	case PtUInt, PtUInt8, PtUInt16, PtUInt24:
		if min := uint64(t.p); min > 0 {
			b.WriteString(`,"minimum":`)
			b.WriteString(strconv.FormatUint(min, 10))
		}
		if max := uint64(t.s); max < maxUInt[t.pt-PtUInt] {
			b.WriteString(`,"maximum":`)
			b.WriteString(strconv.FormatUint(max, 10))
		}
	case PtUInt64:
		if i, ok := t.vl.(uintRange); ok {
			if i.min > 0 {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatUint(i.min, 10))
			}
			if i.max < MaxUInt64 {
				b.WriteString(`,"maximum":`)
				b.WriteString(strconv.FormatUint(i.max, 10))
			}
		}
	case PtFloat, PtFloat32:
		Max := MaxFloat
		if t.pt == PtFloat32 {
			Max = MaxFloat32
		}
		if f, ok := t.vl.(floatRange); ok {
			if f.min > -Max {
				b.WriteString(`,"minimum":`)
				b.WriteString(f.minS)
			}
			if f.max < Max {
				b.WriteString(`,"maximum":`)
				b.WriteString(f.maxS)
			}
		}
	case PtDecimal:
		if d, ok := t.vl.(decimalRange); ok {
			if d.min.GreaterThan(MinDecimal) {
				b.WriteString(`,"minimum":`)
				b.WriteString(d.min.String())
			}
			if d.max.LessThan(MaxDecimal) {
				b.WriteString(`,"maximum":`)
				b.WriteString(d.max.String())
			}
		}
		if t.p > 0 {
			b.WriteString(`,"precision":`)
			b.WriteString(strconv.Itoa(int(t.p)))
		}
		if t.s > 0 {
			b.WriteString(`,"scale":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
	case PtDateTime, PtDate:
		b.WriteString(`,"layout":`)
		marshalString(b, t.vl.(string))
	case PtText:
		if t.p > 0 {
			b.WriteString(`,"byteLen":`)
			b.WriteString(strconv.Itoa(int(t.p)))
		}
		if t.s > 0 {
			b.WriteString(`,"charLen":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
		switch vl := t.vl.(type) {
		case *regexp.Regexp:
			b.WriteString(`,"regexp":"`)
			b.WriteString(vl.String())
			b.WriteString(`"`)
		case []string:
			b.WriteString(`,"enum":[`)
			for i, v := range vl {
				if i > 0 {
					b.WriteByte(',')
				}
				marshalString(b, v)
			}
			b.WriteByte(']')
		}
	case PtArray:
		if t.p > 0 {
			b.WriteString(`,"minItems":`)
			b.WriteString(strconv.Itoa(int(t.p)))
		}
		if t.s < MaxItems {
			b.WriteString(`,"maxItems":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
		if t.unique {
			b.WriteString(`,"uniqueItems":true`)
		}
		b.WriteString(`,"itemType":`)
		marshalType(b, t.vl.(Type), false)
	case PtObject:
		b.WriteString(`,"properties":[`)
		properties := t.vl.([]Property)
		for i, p := range properties {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{"name":`)
			b.WriteByte('"')
			b.WriteString(p.Name)
			b.WriteByte('"')
			if p.Aliases != nil {
				b.WriteString(`,"aliases":[`)
				for i, alias := range p.Aliases {
					if i > 0 {
						b.WriteByte(',')
					}
					b.WriteByte('"')
					b.WriteString(alias)
					b.WriteByte('"')
				}
				b.WriteByte(']')
			}
			if p.Label != "" {
				b.WriteString(`,"label":`)
				_ = marshalString(b, p.Label)
			}
			if p.Description != "" {
				b.WriteString(`,"description":`)
				_ = marshalString(b, p.Description)
			}
			if p.Required {
				b.WriteString(`,"required":true`)
			}
			b.WriteString(`,"type":`)
			marshalType(b, p.Type, false)
			if p.Nullable {
				b.WriteString(`,"nullable":true`)
			}
			b.WriteByte('}')
		}
		b.WriteString("]")
	case PtMap:
		b.WriteString(`,"valueType":`)
		marshalType(b, t.vl.(Type), false)
	}
	b.WriteString(`}`)
}

// unmarshalType reads the JSON tokens from dec and returns the decoded type.
// For custom types, it calls the resolve function, if not nil, to resolve the
// type custom name to its type.
func unmarshalType(dec *json.Decoder, resolve Resolver) (Type, error) {

	// Read a type custom or delimiter '{'.
	tok, err := dec.Token()
	if err != nil {
		return Type{}, err
	}

	// Resolve custom type.
	if name, ok := tok.(string); ok {
		if name == "" {
			return Type{}, errors.New("custom type name is empty")
		}
		if !IsValidCustomTypeName(name) {
			return Type{}, errors.New("custom type name is not valid")
		}
		if resolve == nil {
			return Type{}, errors.New("unknown custom type")
		}
		t, err := resolve(name)
		if err != nil {
			if err == ErrCustomTypeNotExist {
				return Type{}, errors.New("unknown custom type")
			}
			return Type{}, err
		}
		if !t.Valid() {
			return Type{}, errors.New("resolve has returned an invalid type")
		}
		if t.custom == "" {
			return Type{}, errors.New("resolve has returned a non-custom type")
		}
		if t.custom != name {
			return Type{}, errors.New("resolve has not returned the named custom type")
		}
		return t, nil
	}

	if tok != json.Delim('{') {
		return Type{}, errors.New("invalid type syntax")
	}

	var hasScale, hasLayout, hasMinItems, hasMaxItems, hasUniqueItems bool

	var pt PhysicalType
	var lt LogicalType
	var custom string
	var minimum, maximum json.Number
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var enum []string
	var layout string
	var itemType Type
	var minItems, maxItems = 0, MaxItems
	var uniqueItems bool
	var properties []Property
	var valueType Type

	var ok bool

	// Read type keys and values.
	for {

		// Read the key or the delimiter '}'.
		tok, err = dec.Token()
		if err != nil {
			return Type{}, err
		}
		if _, ok = tok.(json.Delim); ok {
			break
		}
		key := tok.(string)

		switch key {
		case "itemType":
			itemType, err = unmarshalType(dec, resolve)
			if err != nil {
				return Type{}, err
			}
			continue
		case "valueType":
			valueType, err = unmarshalType(dec, resolve)
			if err != nil {
				return Type{}, err
			}
			continue
		}

		// Read the value.
		tok, err = dec.Token()
		if err != nil {
			return Type{}, err
		}

		switch key {
		case "name":
			if pt.Valid() {
				return Type{}, errors.New("repeated 'name' key")
			}
			pt, ok = PhysicalTypeByName(tok.(string))
			if !ok {
				return Type{}, errors.New("invalid physical type")
			}
		case "logical":
			if lt.Valid() {
				return Type{}, errors.New("repeated 'logical' key")
			}
			lt, ok = LogicalTypeByName(tok.(string))
			if !ok {
				return Type{}, errors.New("invalid logical type")
			}
		case "custom":
			if custom != "" {
				return Type{}, errors.New("repeated 'custom' key")
			}
			custom, _ = tok.(string)
			if !IsValidCustomTypeName(custom) {
				return Type{}, errors.New("invalid custom")
			}
		case "minimum":
			if minimum != "" {
				return Type{}, errors.New("repeated 'minimum' key")
			}
			minimum, ok = tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid minimum")
			}
		case "maximum":
			if maximum != "" {
				return Type{}, errors.New("repeated 'maximum' key")
			}
			maximum, ok = tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid maximum")
			}
		case "regexp":
			if re != nil {
				return Type{}, errors.New("repeated 'regexp' key")
			}
			if enum != nil {
				return Type{}, errors.New("regular expression cannot be provided if enum is provided")
			}
			if expr, ok := tok.(string); ok {
				re, err = regexp.Compile(expr)
			}
			if re == nil {
				return Type{}, errors.New("invalid regular expression")
			}
		case "enum":
			if enum != nil {
				return Type{}, errors.New(`repeated "enum" key`)
			}
			if re != nil {
				return Type{}, errors.New("enum cannot be provided if regular expression is provided")
			}
			if tok != json.Delim('[') {
				return Type{}, errors.New("invalid enum")
			}
		Enum:
			for {
				tok, err = dec.Token()
				if err != nil {
					return Type{}, err
				}
				switch v := tok.(type) {
				case string:
					enum = append(enum, v)
				case json.Delim:
					break Enum
				default:
					return Type{}, errors.New("invalid value in enum")
				}
			}
			if len(enum) == 0 {
				return Type{}, errors.New("invalid empty enum")
			}
		case "precision":
			if precision > 0 {
				return Type{}, errors.New("repeated 'precision' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid precision")
			}
			precision, _ = strconv.Atoi(string(n))
			if precision <= 0 || precision > MaxDecimalPrecision {
				return Type{}, errors.New("invalid precision")
			}
		case "scale":
			if hasScale {
				return Type{}, errors.New("repeated 'scale' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid scale")
			}
			scale, _ = strconv.Atoi(string(n))
			if scale < 0 || scale > MaxDecimalScale {
				return Type{}, errors.New("invalid scale")
			}
			hasScale = true
		case "layout":
			if hasLayout {
				return Type{}, errors.New("repeated 'layout' key")
			}
			layout, ok = tok.(string)
			if !ok {
				return Type{}, errors.New("invalid layout")
			}
			if layout == "" {
				return Type{}, errors.New("layout cannot be empty")
			}
			hasLayout = true
		case "byteLen":
			if byteLen > 0 {
				return Type{}, errors.New("repeated 'byteLen' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid length in bytes")
			}
			byteLen, _ = strconv.Atoi(string(n))
			if byteLen <= 0 || byteLen > MaxTextLen {
				return Type{}, errors.New("invalid length in bytes")
			}
		case "charLen":
			if charLen > 0 {
				return Type{}, errors.New("repeated 'charLen' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid length in characters")
			}
			charLen, _ = strconv.Atoi(string(n))
			if charLen <= 0 || charLen > MaxTextLen {
				return Type{}, errors.New("invalid length in characters")
			}
		case "minItems":
			if hasMinItems {
				return Type{}, errors.New("repeated 'minItems' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid min items")
			}
			minItems, _ = strconv.Atoi(string(n))
			if minItems < 0 || minItems > MaxItems {
				return Type{}, errors.New("invalid min items")
			}
			hasMinItems = true
		case "maxItems":
			if hasMaxItems {
				return Type{}, errors.New("repeated 'maxItems' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid max items")
			}
			maxItems, _ = strconv.Atoi(string(n))
			if maxItems < 0 || maxItems > MaxItems {
				return Type{}, errors.New("invalid max items")
			}
			hasMaxItems = true
		case "uniqueItems":
			if hasUniqueItems {
				return Type{}, errors.New("repeated 'uniqueItems' key")
			}
			uniqueItems, ok = tok.(bool)
			if !ok {
				return Type{}, errors.New("invalid unique items")
			}
			hasUniqueItems = true
		case "properties":
			if properties != nil {
				return Type{}, errors.New("repeated 'properties' key")
			}
			if tok != json.Delim('[') {
				return Type{}, errors.New("invalid properties")
			}
			exists := map[string]struct{}{}
			for {
				// Read delimiter '{' or ']'.
				tok, err := dec.Token()
				if err != nil {
					return Type{}, err
				}
				d, ok := tok.(json.Delim)
				if !ok || d != '{' && d != ']' {
					return Type{}, errors.New("invalid property syntax")
				}
				if d == ']' {
					break
				}
				property, _, err := unmarshalProperty(dec, resolve, false)
				if err != nil {
					return Type{}, err
				}
				if _, ok := exists[property.Name]; ok {
					return Type{}, errors.New("repeated property name")
				}
				exists[property.Name] = struct{}{}
				for _, alias := range property.Aliases {
					if _, ok := exists[alias]; ok {
						return Type{}, errors.New("property alias already named")
					}
					exists[alias] = struct{}{}
				}
				properties = append(properties, property)
			}
			if properties == nil {
				return Type{}, errors.New("invalid empty properties")
			}
		default:
			if key == "items" {
				return Type{}, fmt.Errorf(`unknown key %q (maybe "itemType"?)`, key)
			}
			return Type{}, fmt.Errorf("unknown key %q", key)
		}

	}

	var t Type

	if !pt.Valid() {
		return Type{}, errors.New("missing 'name' key")
	}
	t.pt = pt
	if lt.Valid() {
		t.lt = lt
	}
	t.custom = custom
	if minimum == "" {
		if PtInt <= t.pt && t.pt <= PtInt24 {
			t.p = int32(minInt[t.pt-PtInt])
		}
	} else {
		switch t.pt {
		case PtInt, PtInt8, PtInt16, PtInt24, PtInt64:
			min, err := minimum.Int64()
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			Min, Max := minInt[t.pt-PtInt], maxInt[t.pt-PtInt]
			if min < Min || min > Max {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min > Min {
				if t.pt == PtInt64 {
					t.vl = intRange{min, Max}
				} else {
					t.p = int32(min)
				}
			}
		case PtUInt, PtUInt8, PtUInt16, PtUInt24, PtUInt64:
			min, err := strconv.ParseUint(string(minimum), 10, 64)
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			Max := maxUInt[t.pt-PtInt]
			if min < 0 || min > Max {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min > 0 {
				if t.pt == PtInt64 {
					t.vl = uintRange{min, Max}
				} else {
					t.p = int32(min)
				}
			}
		case PtFloat:
			min, err := minimum.Float64()
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min < -MaxFloat || min > MaxFloat {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min > -MaxFloat {
				minS := decimal.NewFromFloat(min).String()
				t.vl = floatRange{min: min, max: MaxFloat, minS: minS}
			}
		case PtFloat32:
			min, err := strconv.ParseFloat(string(minimum), 32)
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min < -MaxFloat32 || min > MaxFloat32 {
				return Type{}, errors.New("invalid value for minimum")
			}
			minS := decimal.NewFromFloat32(float32(min)).String()
			t.vl = floatRange{min: min, max: MaxFloat32, minS: minS}
		case PtDecimal:
			min, err := decimal.NewFromString(string(minimum))
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min.LessThan(MinDecimal) || min.GreaterThan(MaxDecimal) {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min.GreaterThan(MinDecimal) {
				t.vl = decimalRange{min, MaxDecimal}
			}
		default:
			return Type{}, errors.New("unexpected minimum for non-number type")
		}
	}
	if maximum == "" {
		if PtInt <= t.pt && t.pt <= PtInt24 {
			t.s = int32(maxInt[t.pt-PtInt])
		} else if PtUInt <= t.pt && t.pt <= PtUInt24 {
			t.s = int32(maxUInt[t.pt-PtUInt])
		}
	} else {
		switch t.pt {
		case PtInt, PtInt8, PtInt16, PtInt24, PtInt64:
			max, err := maximum.Int64()
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			Min, Max := minInt[t.pt-PtInt], maxInt[t.pt-PtInt]
			if max < Min || max > Max {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < Max {
				if t.pt == PtInt64 {
					if i, ok := t.vl.(intRange); ok {
						if max < i.min {
							return Type{}, errors.New("maximum cannot be less than minimum")
						}
						i.max = max
						t.vl = i
					} else {
						t.vl = intRange{Min, max}
					}
				} else {
					if min := int64(t.p); max < min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					t.s = int32(max)
				}
			}
		case PtUInt, PtUInt8, PtUInt16, PtUInt24, PtUInt64:
			max, err := strconv.ParseUint(string(maximum), 10, 64)
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			Max := maxUInt[t.pt-PtInt]
			if max < 0 || max > Max {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < Max {
				if t.pt == PtInt64 {
					if f, ok := t.vl.(uintRange); ok {
						if max < f.min {
							return Type{}, errors.New("maximum cannot be less than minimum")
						}
						f.max = max
						t.vl = f
					} else {
						t.vl = uintRange{0, max}
					}
				} else {
					if min := uint64(t.p); max < min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					t.s = int32(max)
				}
			}
		case PtFloat:
			max, err := maximum.Float64()
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < -MaxFloat || max > MaxFloat {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < MaxFloat {
				maxS := decimal.NewFromFloat(max).String()
				if f, ok := t.vl.(floatRange); ok {
					if max < f.min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					f.max = max
					f.maxS = maxS
					t.vl = f
				} else {
					t.vl = floatRange{min: -MaxFloat, max: max, maxS: maxS}
				}
			}
		case PtFloat32:
			max, err := strconv.ParseFloat(string(maximum), 32)
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			max = float64(float32(max))
			if max < -MaxFloat32 || max > MaxFloat32 {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < MaxFloat32 {
				maxS := decimal.NewFromFloat32(float32(max)).String()
				if f, ok := t.vl.(floatRange); ok {
					if max < f.min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					f.max = max
					f.maxS = maxS
					t.vl = f
				} else {
					t.vl = floatRange{min: -MaxFloat32, max: max, maxS: maxS}
				}
			}
		case PtDecimal:
			max, err := decimal.NewFromString(string(maximum))
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max.LessThan(MinDecimal) || max.GreaterThan(MaxDecimal) {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max.LessThan(MaxDecimal) {
				if d, ok := t.vl.(decimalRange); ok {
					if max.LessThan(d.min) {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					d.max = max
					t.vl = d
				} else {
					t.vl = decimalRange{min: MinDecimal, max: max}
				}
			}
		default:
			return Type{}, errors.New("unexpected maximum for non-number type")
		}
	}
	if re != nil {
		if pt != PtText {
			return Type{}, errors.New("unexpected regular expression for non-Text type")
		}
		t.vl = re
	}
	if enum != nil {
		if pt != PtText {
			return Type{}, errors.New("unexpected enum for non-Text type")
		}
		t.vl = enum
	}
	if pt == PtDateTime || pt == PtDate {
		if !hasLayout {
			return Type{}, errors.New("missing 'layout' key")
		}
		t.vl = layout
	} else if hasLayout {
		return Type{}, errors.New("unexpected layout for non-time type")
	}
	if byteLen > 0 {
		if pt != PtText {
			return Type{}, errors.New("unexpected length in bytes for non-Text type")
		}
		t.p = int32(byteLen)
	}
	if charLen > 0 {
		if pt != PtText {
			return Type{}, errors.New("unexpected length in characters for non-Text type")
		}
		t.s = int32(charLen)
	}
	if precision > 0 {
		if pt != PtDecimal {
			return Type{}, errors.New("unexpected precision for non-Decimal type")
		}
		t.p = int32(precision)
	}
	if hasScale {
		if pt != PtDecimal {
			return Type{}, errors.New("unexpected scale for non-Decimal type")
		}
		if precision == 0 {
			return Type{}, errors.New("scale also requires precision")
		}
		if precision < scale {
			return Type{}, errors.New("scale cannot be greater tha precision")
		}
		t.s = int32(scale)
	}
	if itemType.Valid() {
		if pt != PtArray {
			return Type{}, errors.New("unexpected item type for non-Array type")
		}
		t.vl = itemType
	} else {
		if pt == PtArray {
			return Type{}, errors.New("missing item type")
		}
	}
	if hasMinItems {
		if pt != PtArray {
			return Type{}, errors.New("unexpected minItems for non-Array type")
		}
		t.p = int32(minItems)
	}
	if maxItems < MaxItems {
		if pt != PtArray {
			return Type{}, errors.New("unexpected maxItems for non-Array type")
		}
		if maxItems < minItems {
			return Type{}, errors.New("maxItems must be greater or equal to minItems")
		}
	}
	if pt == PtArray {
		t.s = int32(maxItems)
	}
	if hasUniqueItems {
		if pt != PtArray {
			return Type{}, errors.New("unexpected uniqueItems for no Array type")
		}
		if pt := t.vl.(Type).pt; pt == PtArray || pt == PtObject {
			return Type{}, errors.New("unexpected uniqueItems for items with type Array or Object")
		}
		t.unique = uniqueItems
	}
	if properties == nil {
		if pt == PtObject {
			return Type{}, errors.New("missing object properties")
		}
	} else {
		if pt != PtObject {
			return Type{}, errors.New("unexpected properties for non-Object type")
		}
		t.vl = properties
	}
	if valueType.Valid() {
		if pt != PtMap {
			return Type{}, errors.New("unexpected value type for non-Map type")
		}
		t.vl = valueType
	} else {
		if pt == PtMap {
			return Type{}, errors.New("missing value type")
		}
	}

	return t, nil
}

// unmarshalProperty reads the JSON tokens from dec, which must have already
// read the token '{', and returns the decoded property.
// For custom types, it calls the resolve function, if not nil, to resolve the
// type custom name to its type. If inSchema is true, it unmarshals a schema
// property and then also returns its role.
func unmarshalProperty(dec *json.Decoder, resolve Resolver, inSchema bool) (Property, Role, error) {

	var p Property
	var role Role

	// Read property keys and values.
	for {

		// Read a key or delimiter '}'.
		tok, err := dec.Token()
		if err != nil {
			return Property{}, 0, err
		}
		if _, ok := tok.(json.Delim); ok {
			break
		}
		key := tok.(string)

		if key == "type" {
			if p.Type.Valid() {
				return Property{}, 0, errors.New("repeated 'type' key")
			}
			p.Type, err = unmarshalType(dec, resolve)
			if err != nil {
				return Property{}, 0, err
			}
			continue
		}

		// Read the value.
		tok, err = dec.Token()
		if err != nil {
			return Property{}, 0, err
		}

		var ok bool
		var hasLabel, hasDescription, hasNullable, hasRequired, hasRole bool

		switch key {
		case "name":
			if p.Name != "" {
				return Property{}, 0, errors.New("repeated 'name' key")
			}
			p.Name, ok = tok.(string)
			if !ok {
				return Property{}, 0, errors.New("unexpected value for property name")
			}
			if p.Name == "" {
				return Property{}, 0, errors.New("property name is empty")
			}
			if !IsValidPropertyName(p.Name) {
				return Property{}, 0, errors.New("invalid property name")
			}
		case "aliases":
			if p.Aliases != nil {
				return Property{}, 0, errors.New("repeated 'aliases' key")
			}
			p.Aliases = []string{}
			if tok != json.Delim('[') {
				return Property{}, 0, errors.New("invalid aliases")
			}
		Aliases:
			for {
				tok, err = dec.Token()
				if err != nil {
					return Property{}, 0, err
				}
				switch alias := tok.(type) {
				case string:
					if alias == "" {
						return Property{}, 0, errors.New("property alias is empty")
					}
					if !IsValidPropertyName(alias) {
						return Property{}, 0, errors.New("invalid property alias")
					}
					p.Aliases = append(p.Aliases, alias)
				case json.Delim:
					break Aliases
				default:
					return Property{}, 0, errors.New("invalid value in aliases")
				}
			}
			if len(p.Aliases) == 0 {
				return Property{}, 0, errors.New("invalid empty aliases")
			}
		case "label":
			if hasLabel {
				return Property{}, 0, errors.New("repeated 'label' key")
			}
			p.Label, ok = tok.(string)
			if !ok {
				return Property{}, 0, errors.New("unexpected value for property label")
			}
			hasLabel = true
		case "description":
			if hasDescription {
				return Property{}, 0, errors.New("repeated 'description' key")
			}
			p.Description, ok = tok.(string)
			if !ok {
				return Property{}, 0, errors.New("unexpected value for property description")
			}
			hasDescription = true
		case "required":
			if hasRequired {
				return Property{}, 0, errors.New("repeated 'required' key")
			}
			p.Required, ok = tok.(bool)
			if !ok {
				return Property{}, 0, errors.New("unexpected value for 'required' key of property")
			}
			hasRequired = true
		case "role":
			if !inSchema {
				return Property{}, 0, errors.New("unknown property 'role'")
			}
			if hasRole {
				return Property{}, 0, errors.New("repeated 'role' key")
			}
			switch r, _ := tok.(string); r {
			case "both":
			case "source":
				role = SourceRole
			case "destination":
				role = DestinationRole
			default:
				return Property{}, 0, errors.New("unexpected value for property role")
			}
			hasRole = true
		case "nullable":
			if hasNullable {
				return Property{}, 0, errors.New("repeated 'nullable' key")
			}
			p.Nullable, ok = tok.(bool)
			if !ok {
				return Property{}, 0, errors.New("unexpected value for 'nullable' key of property")
			}
			hasNullable = true
		default:
			return Property{}, 0, fmt.Errorf("unknown property '%s'", key)
		}

	}

	if p.Name == "" {
		return Property{}, 0, errors.New("missing property name")
	}
	if !p.Type.Valid() {
		return Property{}, 0, errors.New("missing property type")
	}

	return p, role, nil
}

const hex = "0123456789abcdef"

// marshalString marshals s as a JSON string and writes it to b.
//
// This code is derived from the (*encodeState).string method of the json
// standard package that is copyright The Go Authors.
func marshalString(b *bytes.Buffer, s string) error {
	b.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c > 31 && c != '"' && c != '\\' {
				i++
				continue
			}
			if start < i {
				b.WriteString(s[start:i])
			}
			b.WriteByte('\\')
			switch c {
			case '"', '\\':
				b.WriteByte(c)
			case '\n':
				b.WriteByte('n')
			case '\r':
				b.WriteByte('r')
			case '\t':
				b.WriteByte('t')
			default:
				b.WriteString(`u00`)
				b.WriteByte(hex[c>>4])
				b.WriteByte(hex[c&0xF])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return errors.New("invalid UTF-8 encoding")
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		if r == '\u2028' || r == '\u2029' {
			if start < i {
				b.WriteString(s[start:i])
			}
			b.WriteString(`\u202`)
			b.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		b.WriteString(s[start:])
	}
	b.WriteByte('"')
	return nil
}
