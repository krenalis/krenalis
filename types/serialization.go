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
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/text/unicode/norm"
)

var errNullToken = errors.New("invalid type syntax")

var null = []byte("null")

// Parse parses the JSON-encoded data and returns the decoded type.
// If data represents JSON null, Parse returns an error.
func Parse(data string) (Type, error) {
	dec := json.NewDecoder(strings.NewReader(norm.NFC.String(data)))
	dec.UseNumber()
	t, err := unmarshalType(dec)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return Type{}, err
	}
	if tok, err := dec.Token(); err != io.EOF {
		return Type{}, fmt.Errorf("invalid token %s after top-level value", tok)
	}
	return t, nil
}

// MarshalJSON marshals t into JSON.
// If t is not valid, it is marshalled as 'null'.
func (t Type) MarshalJSON() ([]byte, error) {
	if !t.Valid() {
		return null, nil
	}
	var b bytes.Buffer
	marshalType(&b, t)
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the type
// pointed by t.
func (t *Type) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(norm.NFC.Bytes(data)))
	dec.UseNumber()
	t2, err := unmarshalType(dec)
	if err != nil && err != errNullToken {
		return fmt.Errorf("json: %s", err)
	}
	*t = t2
	return nil
}

// marshalType marshals t as JSON and writes it to b.
func marshalType(b *bytes.Buffer, t Type) {
	b.WriteString(`{"name":"`)
	b.WriteString(t.kind.String())
	b.WriteString(`"`)
	switch t.kind {
	case IntKind:
		b.WriteString(`,"bitSize":`)
		b.WriteString(strconv.Itoa(t.BitSize()))
		if t.size < 4 {
			// 8, 16, 24, and 32 bits.
			if min := int64(t.p); min > minInt[t.size] {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatInt(min, 10))
			}
			if max := int64(t.s); max < maxInt[t.size] {
				b.WriteString(`,"maximum":`)
				b.WriteString(strconv.FormatInt(max, 10))
			}
		} else {
			// 64 bits.
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
		}
	case UintKind:
		b.WriteString(`,"bitSize":`)
		b.WriteString(strconv.Itoa(t.BitSize()))
		if t.size < 4 {
			// 8, 16, 24, and 32 bits.
			if min := uint64(t.p); min > 0 {
				b.WriteString(`,"minimum":`)
				b.WriteString(strconv.FormatUint(min, 10))
			}
			if max := uint64(t.s); max < maxUint[t.size] {
				b.WriteString(`,"maximum":`)
				b.WriteString(strconv.FormatUint(max, 10))
			}
		} else {
			// 64 bits.
			if i, ok := t.vl.(uintRange); ok {
				if i.min > 0 {
					b.WriteString(`,"minimum":`)
					b.WriteString(strconv.FormatUint(i.min, 10))
				}
				if i.max < MaxUint64 {
					b.WriteString(`,"maximum":`)
					b.WriteString(strconv.FormatUint(i.max, 10))
				}
			}
		}
	case FloatKind:
		b.WriteString(`,"bitSize":`)
		b.WriteString(strconv.Itoa(t.BitSize()))
		if t.real {
			b.WriteString(`,"real":true`)
		}
		if f, ok := t.vl.(floatRange); ok {
			if !math.IsInf(f.min, -1) {
				b.WriteString(`,"minimum":`)
				b.WriteString(f.minS)
			}
			if !math.IsInf(f.max, 1) {
				b.WriteString(`,"maximum":`)
				b.WriteString(f.maxS)
			}
		}
	case DecimalKind:
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
	case TextKind:
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
			b.WriteString(`,"values":[`)
			for i, v := range vl {
				if i > 0 {
					b.WriteByte(',')
				}
				marshalString(b, v)
			}
			b.WriteByte(']')
		}
	case ArrayKind:
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
		marshalType(b, t.vl.(Type))
	case ObjectKind:
		b.WriteString(`,"properties":[`)
		properties := t.vl.([]Property)
		for i, p := range properties {
			if i > 0 {
				b.WriteString(",")
			}
			_ = marshalProperty(b, p)
		}
		b.WriteString("]")
	case MapKind:
		b.WriteString(`,"valueType":`)
		marshalType(b, t.vl.(Type))
	}
	b.WriteString(`}`)
}

// MarshalJSON marshals p into JSON.
func (p Property) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	err := marshalProperty(&b, p)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the
// property pointed by p.
func (p *Property) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(norm.NFC.Bytes(data)))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok || d != '{' {
		return errors.New("json: invalid property syntax")
	}
	p2, err := unmarshalProperty(dec)
	if err != nil {
		return fmt.Errorf("json: %s", err)
	}
	*p = p2
	return nil
}

// marshalProperty marshals p as JSON and writes it to b.
func marshalProperty(b *bytes.Buffer, p Property) error {
	if p.Name == "" {
		return errors.New("missing property name")
	}
	if !p.Type.Valid() {
		return errors.New("missing property type")
	}
	b.WriteString(`{"name":`)
	b.WriteByte('"')
	b.WriteString(p.Name)
	b.WriteByte('"')
	b.WriteString(`,"label":`)
	_ = marshalString(b, p.Label)
	b.WriteString(`,"description":`)
	_ = marshalString(b, p.Description)
	switch ph := p.Placeholder.(type) {
	case nil:
		b.WriteString(`,"placeholder":null`)
	case string:
		b.WriteString(`,"placeholder":`)
		_ = marshalString(b, ph)
	case map[string]string:
		b.WriteString(`,"placeholder":{`)
		keys := maps.Keys(ph)
		slices.Sort(keys)
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			_ = marshalString(b, k)
			b.WriteByte(':')
			_ = marshalString(b, ph[k])
		}
		b.WriteByte('}')
	}
	b.WriteString(`,"type":`)
	marshalType(b, p.Type)
	if p.Required {
		b.WriteString(`,"required":true`)
	}
	if p.Nullable {
		b.WriteString(`,"nullable":true`)
	} else {
		b.WriteString(`,"nullable":false`)
	}
	b.WriteByte('}')
	return nil
}

// unmarshalType reads the JSON tokens from dec and returns the decoded type.
// If the first token is 'null' it returns the errNullToken error.
func unmarshalType(dec *json.Decoder) (Type, error) {

	// Read the delimiter '{' or 'null'.
	tok, err := dec.Token()
	if err != nil {
		return Type{}, err
	}
	if tok == nil {
		return Type{}, errNullToken
	}
	if tok != json.Delim('{') {
		return Type{}, errors.New("invalid type syntax")
	}

	var hasReal, hasScale, hasMinItems, hasMaxItems, hasUniqueItems bool

	var kind Kind
	var bitSize int
	var minimum, maximum json.Number
	var real bool
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var values []string
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
			itemType, err = unmarshalType(dec)
			if err != nil {
				return Type{}, err
			}
			continue
		case "valueType":
			valueType, err = unmarshalType(dec)
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
			if kind.Valid() {
				return Type{}, errors.New("repeated 'name' key")
			}
			kind, ok = KindByName(tok.(string))
			if !ok {
				return Type{}, errors.New("invalid kind type")
			}
		case "bitSize":
			if bitSize != 0 {
				return Type{}, errors.New("repeated 'bitSize' key")
			}
			s, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid bit size")
			}
			size, _ := s.Int64()
			switch size {
			case 8, 16, 24, 32, 64:
				bitSize = int(size)
			default:
				return Type{}, errors.New("invalid bit size")
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
		case "real":
			if hasReal {
				return Type{}, errors.New("repeated 'real' key")
			}
			real, ok = tok.(bool)
			if !ok {
				return Type{}, errors.New("invalid real")
			}
			hasReal = true
		case "regexp":
			if re != nil {
				return Type{}, errors.New("repeated 'regexp' key")
			}
			if values != nil {
				return Type{}, errors.New("regular expression cannot be provided if values are provided")
			}
			if expr, ok := tok.(string); ok {
				re, _ = regexp.Compile(expr)
			}
			if re == nil {
				return Type{}, errors.New("invalid regular expression")
			}
		case "values":
			if values != nil {
				return Type{}, errors.New(`repeated value`)
			}
			if re != nil {
				return Type{}, errors.New("values cannot be provided if regular expression is provided")
			}
			if tok != json.Delim('[') {
				return Type{}, errors.New("invalid values")
			}
		Values:
			for {
				tok, err = dec.Token()
				if err != nil {
					return Type{}, err
				}
				switch v := tok.(type) {
				case string:
					values = append(values, v)
				case json.Delim:
					break Values
				default:
					return Type{}, errors.New("invalid value")
				}
			}
			if len(values) == 0 {
				return Type{}, errors.New("invalid empty values")
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
				property, err := unmarshalProperty(dec)
				if err != nil {
					return Type{}, err
				}
				if _, ok := exists[property.Name]; ok {
					return Type{}, errors.New("repeated property name")
				}
				exists[property.Name] = struct{}{}
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

	if !kind.Valid() {
		return Type{}, errors.New("missing 'name' key")
	}
	t.kind = kind
	if bitSize == 0 {
		if t.kind == IntKind || t.kind == UintKind || t.kind == FloatKind {
			return Type{}, errors.New("missing 'bitSize' key")
		}
	} else {
		switch t.kind {
		case IntKind, UintKind:
		case FloatKind:
			if bitSize != 32 && bitSize != 64 {
				return Type{}, errors.New("invalid bit size")
			}
		default:
			return Type{}, errors.New("invalid bit size")
		}
		switch bitSize {
		case 8:
			t.size = 0
		case 16:
			t.size = 1
		case 24:
			t.size = 2
		case 32:
			t.size = 3
		case 64:
			t.size = 4
		}
	}
	if minimum == "" {
		if t.kind == IntKind && t.size < 4 { // 8, 16, 24, and 32 bits
			t.p = int32(minInt[t.size])
		}
	} else {
		switch t.kind {
		case IntKind:
			min, err := minimum.Int64()
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			Min, Max := minInt[t.size], maxInt[t.size]
			if min < Min || min > Max {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min > Min {
				if t.size < 4 {
					t.p = int32(min) // 8, 16, 24, and 32 bits
				} else {
					t.vl = intRange{min, Max} // 64 bits
				}
			}
		case UintKind:
			min, err := strconv.ParseUint(string(minimum), 10, 64)
			if err != nil {
				return Type{}, errors.New("invalid value for minimum")
			}
			Max := maxUint[t.size]
			if min > Max {
				return Type{}, errors.New("invalid value for minimum")
			}
			if min > 0 {
				if t.size < 4 {
					t.p = int32(min) // 8, 16, 24, and 32 bits
				} else {
					t.vl = uintRange{min, Max} // 64 bits
				}
			}
		case FloatKind:
			if t.size == 4 {
				// 64 bits.
				min, err := minimum.Float64()
				if err != nil || math.IsNaN(min) {
					return Type{}, errors.New("invalid value for minimum")
				}
				if !math.IsInf(min, -1) {
					minS := decimal.NewFromFloat(min).String()
					t.vl = floatRange{min: min, max: math.Inf(1), minS: minS}
				}
			} else {
				// 32 bits.
				min, err := strconv.ParseFloat(string(minimum), 32)
				if err != nil || math.IsNaN(min) {
					return Type{}, errors.New("invalid value for minimum")
				}
				if min < -math.MaxFloat32 && !math.IsInf(min, -1) || min > math.MaxFloat32 && !math.IsInf(min, 1) {
					return Type{}, errors.New("invalid value for minimum")
				}
				if !math.IsInf(min, -1) {
					minS := decimal.NewFromFloat32(float32(min)).String()
					t.vl = floatRange{min: min, max: math.Inf(1), minS: minS}
				}
			}
		case DecimalKind:
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
		if t.size < 4 {
			// 8, 16, 24, and 32 bits.
			switch t.kind {
			case IntKind:
				t.s = int32(maxInt[t.size])
			case UintKind:
				t.s = int32(maxUint[t.size])
			}
		}
	} else {
		switch t.kind {
		case IntKind:
			max, err := maximum.Int64()
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			Min, Max := minInt[t.size], maxInt[t.size]
			if max < Min || max > Max {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < Max {
				if t.size == 4 {
					// 64 bits.
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
		case UintKind:
			max, err := strconv.ParseUint(string(maximum), 10, 64)
			if err != nil {
				return Type{}, errors.New("invalid value for maximum")
			}
			Max := maxUint[t.size]
			if max > Max {
				return Type{}, errors.New("invalid value for maximum")
			}
			if max < Max {
				if t.size == 4 {
					// 64 bits.
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
		case FloatKind:
			if t.size == 4 {
				// 64 bits.
				max, err := maximum.Float64()
				if err != nil || math.IsNaN(max) {
					return Type{}, errors.New("invalid value for maximum")
				}
				if !math.IsInf(max, 1) {
					maxS := decimal.NewFromFloat(max).String()
					if f, ok := t.vl.(floatRange); ok {
						if max < f.min {
							return Type{}, errors.New("maximum cannot be less than minimum")
						}
						f.max = max
						f.maxS = maxS
						t.vl = f
					} else {
						t.vl = floatRange{min: math.Inf(-1), max: max, maxS: maxS}
					}
				}
			} else {
				// 32 bits.
				max, err := strconv.ParseFloat(string(maximum), 32)
				if err != nil || math.IsNaN(max) {
					return Type{}, errors.New("invalid value for maximum")
				}
				max = float64(float32(max))
				if !math.IsInf(max, 1) {
					maxS := decimal.NewFromFloat32(float32(max)).String()
					if f, ok := t.vl.(floatRange); ok {
						if max < f.min {
							return Type{}, errors.New("maximum cannot be less than minimum")
						}
						f.max = max
						f.maxS = maxS
						t.vl = f
					} else {
						t.vl = floatRange{min: math.Inf(-1), max: max, maxS: maxS}
					}
				}
			}
		case DecimalKind:
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
	if hasReal {
		if kind != FloatKind {
			return Type{}, errors.New("unexpected real for non-Float type")
		}
		t.real = real
	}
	if re != nil {
		if kind != TextKind {
			return Type{}, errors.New("unexpected regular expression for non-Text type")
		}
		t.vl = re
	}
	if values != nil {
		if kind != TextKind {
			return Type{}, errors.New("unexpected values for non-Text type")
		}
		t.vl = values
	}
	if byteLen > 0 {
		if kind != TextKind {
			return Type{}, errors.New("unexpected length in bytes for non-Text type")
		}
		t.p = int32(byteLen)
	}
	if charLen > 0 {
		if kind != TextKind {
			return Type{}, errors.New("unexpected length in characters for non-Text types")
		}
		t.s = int32(charLen)
	}
	if precision > 0 {
		if kind != DecimalKind {
			return Type{}, errors.New("unexpected precision for non-Decimal type")
		}
		t.p = int32(precision)
	}
	if hasScale {
		if kind != DecimalKind {
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
		if kind != ArrayKind {
			return Type{}, errors.New("unexpected item type for non-Array type")
		}
		t.vl = itemType
	} else {
		if kind == ArrayKind {
			return Type{}, errors.New("missing item type")
		}
	}
	if hasMinItems {
		if kind != ArrayKind {
			return Type{}, errors.New("unexpected minItems for non-Array type")
		}
		t.p = int32(minItems)
	}
	if maxItems < MaxItems {
		if kind != ArrayKind {
			return Type{}, errors.New("unexpected maxItems for non-Array type")
		}
		if maxItems < minItems {
			return Type{}, errors.New("maxItems must be greater or equal to minItems")
		}
	}
	if kind == ArrayKind {
		t.s = int32(maxItems)
	}
	if hasUniqueItems {
		if kind != ArrayKind {
			return Type{}, errors.New("unexpected uniqueItems for non-Array type")
		}
		if k := t.vl.(Type).kind; k == ArrayKind || k == ObjectKind {
			return Type{}, errors.New("unexpected uniqueItems for items with type Array or Object")
		}
		t.unique = uniqueItems
	}
	if properties == nil {
		if kind == ObjectKind {
			return Type{}, errors.New("missing object properties")
		}
	} else {
		if kind != ObjectKind {
			return Type{}, errors.New("unexpected properties for non-Object type")
		}
		t.vl = properties
	}
	if valueType.Valid() {
		if kind != MapKind {
			return Type{}, errors.New("unexpected value type for non-Map type")
		}
		t.vl = valueType
	} else {
		if kind == MapKind {
			return Type{}, errors.New("missing value type")
		}
	}

	return t, nil
}

// unmarshalProperty reads the JSON tokens from dec, which must have already
// read the token '{', and returns the decoded property.
func unmarshalProperty(dec *json.Decoder) (Property, error) {

	var p Property
	var hasLabel, hasDescription, hasPlaceholder, hasRequired, hasNullable bool

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
				return Property{}, errors.New("json: repeated 'type' key")
			}
			p.Type, err = unmarshalType(dec)
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
				return Property{}, errors.New("json: repeated 'name' key")
			}
			p.Name, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property name")
			}
			if p.Name == "" {
				return Property{}, errors.New("property name is empty")
			}
			if !IsValidPropertyName(p.Name) {
				return Property{}, errors.New("invalid property name")
			}
		case "label":
			if hasLabel {
				return Property{}, errors.New("repeated 'label' key")
			}
			p.Label, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property label")
			}
			hasLabel = true
		case "description":
			if hasDescription {
				return Property{}, errors.New("repeated 'description' key")
			}
			p.Description, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property description")
			}
			hasDescription = true
		case "placeholder":
			if hasPlaceholder {
				return Property{}, errors.New("repeated 'placeholder' key")
			}
			switch tok.(type) {
			case nil:
			case string:
				p.Placeholder = tok
			case json.Delim:
				if tok != json.Delim('{') {
					return Property{}, errors.New("unexpected value for property placeholder")
				}
				placeholder := map[string]string{}
				for {
					tok, err = dec.Token()
					if err != nil {
						return Property{}, err
					}
					if tok == json.Delim('}') {
						break
					}
					k := tok.(string)
					tok, err = dec.Token()
					if err != nil {
						return Property{}, err
					}
					v, ok := tok.(string)
					if !ok {
						return Property{}, errors.New("unexpected value for property placeholder")
					}
					placeholder[k] = v
				}
				p.Placeholder = placeholder
			default:
				return Property{}, errors.New("unexpected value for property placeholder")
			}
		case "required":
			if hasRequired {
				return Property{}, errors.New("repeated 'required' key")
			}
			p.Required, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'required' key of property")
			}
			hasRequired = true
		case "nullable":
			if hasNullable {
				return Property{}, errors.New("repeated 'nullable' key")
			}
			p.Nullable, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'nullable' key of property")
			}
			hasNullable = true
		default:
			return Property{}, fmt.Errorf("unknown property '%s'", key)
		}

	}

	if p.Name == "" {
		return Property{}, errors.New("missing property name")
	}
	if !p.Type.Valid() {
		return Property{}, errors.New("missing property type")
	}
	if hasPlaceholder {
		if _, ok := p.Placeholder.(map[string]string); ok && p.Type.Kind() != MapKind {
			return Property{}, errors.New("invalid placeholder value")
		}
	}

	return p, nil
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
