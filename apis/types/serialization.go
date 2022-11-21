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

	"github.com/shopspring/decimal"
)

// Resolver resolves a custom type name to its type. If a custom type with the
// given name does not exist, it returns a ErrCustomTypeNotExist error.
type Resolver func(name string) (Type, error)

// A ErrCustomTypeNotExist error is returned by a Resolver function when the
// custom type to resolve does not exist.
var ErrCustomTypeNotExist = errors.New("custom type does not exist")

// MarshalType marshals t into JSON.
func MarshalType(t Type) ([]byte, error) {
	var b bytes.Buffer
	marshalType(&b, t)
	return b.Bytes(), nil
}

// UnmarshalType parses the JSON-encoded data and returns the decoded type.
func UnmarshalType(data []byte, resolve Resolver) (Type, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	t, err := unmarshalType(dec, resolve)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return t, err
}

// MarshalJSON marshals t into JSON.
func (t Type) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	marshalType(&b, t)
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the type
// pointed by t.
func (t *Type) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	t2, err := unmarshalType(dec, nil)
	if err != nil {
		return err
	}
	*t = t2
	return nil
}

// MarshalJSON marshals property p into JSON.
func (p Property) MarshalJSON() ([]byte, error) {
	if !p.Type.Valid() {
		return nil, errors.New("property type is not valid")
	}
	var b bytes.Buffer
	marshalProperty(&b, p)
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the
// property pointed by p.
func (p *Property) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	// Read delimiter '{'.
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return errors.New("invalid property syntax")
	}
	// Read the property.
	p2, err := unmarshalProperty(dec, nil)
	if err != nil {
		return err
	}
	*p = p2
	return nil
}

// marshalType marshals t as JSON and writes it to b.
func marshalType(b *bytes.Buffer, t Type) {
	if t.custom != "" {
		marshalString(b, t.custom)
		return
	}
	b.WriteString(`{"name":"`)
	b.WriteString(t.pt.String())
	b.WriteString(`"`)
	if t.lt > 0 {
		b.WriteString(`,"logical":"`)
		b.WriteString(t.lt.String())
		b.WriteString(`"`)
	}
	switch t.pt {
	case PtInt, PtInt8, PtInt16, PtInt24, PtInt64:
		if i, ok := t.vl.(intRange); ok {
			if i.min > minInt[t.pt-PtInt] {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatInt(i.min, 10))
			}
			if i.max < maxInt[t.pt-PtInt] {
				b.WriteString(`,"maximum":`)
				b.WriteString(strconv.FormatInt(i.max, 10))
			}
		}
	case PtUInt, PtUInt8, PtUInt16, PtUInt24, PtUInt64:
		if i, ok := t.vl.(uintRange); ok {
			if i.min > 0 {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatUint(i.min, 10))
			}
			if i.max < maxUInt[t.pt-PtUInt] {
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
		if t.s < MaxArrayLen {
			b.WriteString(`,"maxItems":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
		if t.unique {
			b.WriteString(`,"uniqueItems":true`)
		}
		b.WriteString(`,"itemsType":`)
		marshalType(b, t.vl.(Type))
	case PtObject:
		b.WriteString(`,"properties":[`)
		properties := t.vl.([]Property)
		for i, p := range properties {
			if i > 0 {
				b.WriteString(",")
			}
			marshalProperty(b, p)
		}
		b.WriteString("]")
	}
	b.WriteString(`}`)
}

// unmarshalType reads the JSON tokens from dec and returns the decoded type.
// For custom types, it calls the resolve function to resolve the type custom
// name to its type.
func unmarshalType(dec *json.Decoder, resolve Resolver) (Type, error) {

	// Read a type custom or delimiter '{'.
	tok, err := dec.Token()
	if err != nil {
		return Type{}, err
	}

	// Resolve custom type.
	if name, ok := tok.(string); ok {
		if name == "" {
			return Type{}, errors.New("empty custom type name")
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

	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return Type{}, errors.New("invalid type syntax")
	}

	var hasScale, hasMinItems, hasUniqueItems bool

	var pt PhysicalType
	var lt LogicalType
	var minimum, maximum json.Number
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var enum []string
	var itemsType Type
	var minItems, maxItems = 0, MaxArrayLen
	var uniqueItems bool
	var properties []Property

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

		if key == "itemsType" {
			itemsType, err = unmarshalType(dec, resolve)
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
			if d, ok := tok.(json.Delim); !ok || d != '[' {
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
				return Type{}, errors.New("invalid minimum items")
			}
			minItems, _ = strconv.Atoi(string(n))
			if minItems < 0 || minItems > MaxArrayLen {
				return Type{}, errors.New("invalid minimum items")
			}
			hasMinItems = true
		case "maxItems":
			if maxItems < MaxArrayLen {
				return Type{}, errors.New("repeated 'maxItems' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid maximum items")
			}
			maxItems, _ = strconv.Atoi(string(n))
			if maxItems < 0 || maxItems >= MaxArrayLen {
				return Type{}, errors.New("invalid maximum items")
			}
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
			if d, ok := tok.(json.Delim); !ok || d != '[' {
				return Type{}, errors.New("invalid properties")
			}
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
				property, err := unmarshalProperty(dec, resolve)
				if err != nil {
					return Type{}, err
				}
				for _, p := range properties {
					if property.Name == p.Name {
						return Type{}, errors.New("property name is repeated")
					}
				}
				properties = append(properties, property)
			}
			if properties == nil {
				return Type{}, errors.New("invalid empty properties")
			}
		default:
			if key == "items" {
				return Type{}, fmt.Errorf(`unknown key %q (maybe "itemsType"?)`, key)
			}
			return Type{}, fmt.Errorf("unknown key %q", key)
		}

	}

	var t Type

	if !pt.Valid() {
		return Type{}, errors.New("missing physical type")
	}
	t.pt = pt
	if lt.Valid() {
		t.lt = lt
	}
	if minimum != "" {
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
				t.vl = intRange{min, Max}
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
				t.vl = uintRange{min, Max}
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
	if maximum != "" {
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
				if i, ok := t.vl.(intRange); ok {
					if max < i.min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					i.max = max
					t.vl = i
				} else {
					t.vl = intRange{Min, max}
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
				if f, ok := t.vl.(uintRange); ok {
					if max < f.min {
						return Type{}, errors.New("maximum cannot be less than minimum")
					}
					f.max = max
					t.vl = f
				} else {
					t.vl = uintRange{0, max}
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
	if itemsType.Valid() {
		if pt != PtArray {
			return Type{}, errors.New("unexpected items type for non-Array type")
		}
		t.vl = itemsType
	} else {
		if pt == PtArray {
			return Type{}, errors.New("missing items type")
		}
	}
	if hasMinItems {
		if pt != PtArray {
			return Type{}, errors.New("unexpected minItems for non-Array type")
		}
		t.p = int32(minItems)
	}
	if maxItems < MaxArrayLen {
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

	return t, nil
}

// marshalProperty marshals p as JSON and writes it to b.
func marshalProperty(b *bytes.Buffer, p Property) {
	b.WriteString(`{"name":`)
	marshalString(b, p.Name)
	if p.Label != "" {
		b.WriteString(`,"label":"`)
		marshalString(b, p.Label)
	}
	if p.Description != "" {
		b.WriteString(`,"description":"`)
		marshalString(b, p.Description)
	}
	b.WriteString(`,"type":`)
	marshalType(b, p.Type)
	b.WriteByte('}')
}

// unmarshalProperty reads the JSON tokens from dec, which must have already
// read the token '{', and returns the decoded property. For custom types, it
// calls the resolve function to resolve the type custom name to its type.
func unmarshalProperty(dec *json.Decoder, resolve Resolver) (Property, error) {

	var p Property

	// Read property keys and values.
	for {

		// Read a key or delimiter '}'.
		tok, err := dec.Token()
		if err != nil {
			return Property{}, err
		}
		if _, ok := tok.(json.Delim); ok {
			break
		}
		key := tok.(string)

		if key == "type" {
			if p.Type.Valid() {
				return Property{}, errors.New("repeated 'type' key")
			}
			p.Type, err = unmarshalType(dec, resolve)
			if err != nil {
				return Property{}, err
			}
			continue
		}

		// Read the value.
		tok, err = dec.Token()
		if err != nil {
			return Property{}, err
		}

		var ok bool

		switch key {
		case "name":
			if p.Name != "" {
				return Property{}, errors.New("repeated 'name' key")
			}
			p.Name, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property name")
			}
			if p.Name == "" {
				return Property{}, errors.New("unexpected empty property name")
			}
		case "label":
			if p.Label != "" {
				return Property{}, errors.New("repeated 'label' key")
			}
			p.Label, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property label")
			}
			if p.Label == "" {
				return Property{}, errors.New("unexpected empty property label")
			}
		case "description":
			if p.Description != "" {
				return Property{}, errors.New("repeated 'description' key")
			}
			p.Description, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property description")
			}
			if p.Description == "" {
				return Property{}, errors.New("unexpected empty property description")
			}
		default:
			return Property{}, errors.New("unknown property key")
		}

	}

	if p.Name == "" {
		return Property{}, errors.New("missing property name")
	}
	if !p.Type.Valid() {
		return Property{}, errors.New("missing property type")
	}

	return p, nil
}

// marshalString marshals s as a JSON string and writes it to b.
func marshalString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for len(s) > 0 {
		i := strings.IndexAny(s, "&'<>\"\r")
		if i == -1 {
			b.WriteString(s)
			break
		}
		if i > 0 {
			b.WriteString(s[:i])
		}
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '\'':
			esc = "&#39;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			esc = "&#34;"
		case '\r':
			esc = "&#13;"
		}
		b.WriteString(esc)
		s = s[i+1:]
	}
	b.WriteByte('"')
}
