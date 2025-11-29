// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/tools/decimal"

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
	if !t.Valid() && !t.Generic() {
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
		return err
	}
	*t = t2
	return nil
}

// marshalType marshals t as JSON and writes it to b.
func marshalType(b *bytes.Buffer, t Type) {
	b.WriteString(`{"kind":"`)
	b.WriteString(t.KindName())
	b.WriteString(`"`)
	switch t.kind {
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
			b.WriteString(`,"regexp":`)
			_ = marshalString(b, vl.String())
		case []string:
			b.WriteString(`,"values":[`)
			for i, v := range vl {
				if i > 0 {
					b.WriteByte(',')
				}
				_ = marshalString(b, v)
			}
			b.WriteByte(']')
		}
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
		min, max := decimal.Range(t.Precision(), t.Scale())
		dr := t.vl.(decimalRange)
		if dr.min.Greater(min) {
			b.WriteString(`,"minimum":`)
			dr.min.WriteTo(b)
		}
		if dr.max.Less(max) {
			b.WriteString(`,"maximum":`)
			dr.max.WriteTo(b)
		}
		b.WriteString(`,"precision":`)
		b.WriteString(strconv.Itoa(int(t.p)))
		if t.s > 0 {
			b.WriteString(`,"scale":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
	case ArrayKind:
		if t.p > 0 {
			b.WriteString(`,"minElements":`)
			b.WriteString(strconv.Itoa(int(t.p)))
		}
		if t.s < MaxElements {
			b.WriteString(`,"maxElements":`)
			b.WriteString(strconv.Itoa(int(t.s)))
		}
		if t.unique {
			b.WriteString(`,"uniqueElements":true`)
		}
		b.WriteString(`,"elementType":`)
		marshalType(b, t.vl.(Type))
	case ObjectKind:
		b.WriteString(`,"properties":[`)
		properties := t.vl.(Properties).properties
		for i, p := range properties {
			if i > 0 {
				b.WriteString(",")
			}
			_ = marshalProperty(b, p)
		}
		b.WriteString("]")
	case MapKind:
		b.WriteString(`,"elementType":`)
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
		return errors.New("invalid property syntax")
	}
	p2, err := unmarshalProperty(dec)
	if err != nil {
		return err
	}
	*p = p2
	return nil
}

// marshalProperty marshals p as JSON and writes it to b.
func marshalProperty(b *bytes.Buffer, p Property) error {
	if p.Name == "" {
		return errors.New("missing property name")
	}
	if !p.Type.Valid() && !p.Type.Generic() {
		return errors.New("missing property type")
	}
	b.WriteString(`{"name":`)
	b.WriteByte('"')
	b.WriteString(p.Name)
	b.WriteByte('"')
	if p.Prefilled != "" {
		b.WriteString(`,"prefilled":`)
		_ = marshalString(b, p.Prefilled)
	}
	b.WriteString(`,"type":`)
	marshalType(b, p.Type)
	if p.CreateRequired {
		b.WriteString(`,"createRequired":true`)
	}
	if p.UpdateRequired {
		b.WriteString(`,"updateRequired":true`)
	}
	if p.ReadOptional {
		b.WriteString(`,"readOptional":true`)
	}
	if p.Nullable {
		b.WriteString(`,"nullable":true`)
	}
	b.WriteString(`,"description":`)
	_ = marshalString(b, p.Description)
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

	var hasReal, hasScale, hasMinElements, hasMaxElements, hasUniqueElements bool

	var kind string
	var bitSize int
	var minimum, maximum json.Number
	var real bool
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var values []string
	var elementType Type
	var minElements, maxElements = 0, MaxElements
	var uniqueElements bool
	var properties []Property

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

		if key == "elementType" {
			elementType, err = unmarshalType(dec)
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
		case "kind":
			if kind != "" {
				return Type{}, errors.New("repeated 'kind' key")
			}
			kind = tok.(string)
			if !IsValidPropertyName(kind) {
				return Type{}, errors.New("invalid type kind")
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
		case "minElements":
			if hasMinElements {
				return Type{}, errors.New("repeated 'minElements' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid min elements")
			}
			minElements, _ = strconv.Atoi(string(n))
			if minElements < 0 || minElements > MaxElements {
				return Type{}, errors.New("invalid min elements")
			}
			hasMinElements = true
		case "maxElements":
			if hasMaxElements {
				return Type{}, errors.New("repeated 'maxElements' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid max elements")
			}
			maxElements, _ = strconv.Atoi(string(n))
			if maxElements < 0 || maxElements > MaxElements {
				return Type{}, errors.New("invalid max elements")
			}
			hasMaxElements = true
		case "uniqueElements":
			if hasUniqueElements {
				return Type{}, errors.New("repeated 'uniqueElements' key")
			}
			uniqueElements, ok = tok.(bool)
			if !ok {
				return Type{}, errors.New("invalid unique elements")
			}
			hasUniqueElements = true
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
			if key == "elements" {
				return Type{}, fmt.Errorf(`unknown key %q (maybe "elementType"?)`, key)
			}
			return Type{}, fmt.Errorf("unknown key %q", key)
		}

	}

	var t Type

	if kind == "" {
		return Type{}, errors.New("missing 'kind' key")
	}
	t.kind, _ = KindByName(kind)
	if t.kind == InvalidKind {
		t.generic = true
		t.vl = kind
	}
	if re != nil {
		if t.kind != TextKind {
			return Type{}, errors.New("unexpected regular expression for non-text type")
		}
		t.vl = re
	}
	if values != nil {
		if t.kind != TextKind {
			return Type{}, errors.New("unexpected values for non-text type")
		}
		t.vl = values
	}
	if byteLen > 0 {
		if t.kind != TextKind {
			return Type{}, errors.New("unexpected length in bytes for non-text type")
		}
		t.p = int32(byteLen)
	}
	if charLen > 0 {
		if t.kind != TextKind {
			return Type{}, errors.New("unexpected length in characters for non-text types")
		}
		t.s = int32(charLen)
	}
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
	if precision == 0 {
		if t.kind == DecimalKind {
			return Type{}, errors.New("missing precision")
		}
	} else {
		if t.kind != DecimalKind {
			return Type{}, errors.New("unexpected precision for non-decimal type")
		}
		t.p = int32(precision)
	}
	if hasScale {
		if t.kind != DecimalKind {
			return Type{}, errors.New("unexpected scale for non-decimal type")
		}
		if precision == 0 {
			return Type{}, errors.New("scale also requires precision")
		}
		if precision < scale {
			return Type{}, errors.New("scale cannot be greater tha precision")
		}
		t.s = int32(scale)
	}
	if t.kind == DecimalKind {
		min, max := decimal.Range(precision, scale)
		t.vl = decimalRange{min, max}
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
					minS := strconv.FormatFloat(min, 'f', -1, 64)
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
					minS := strconv.FormatFloat(min, 'f', -1, 32)
					t.vl = floatRange{min: min, max: math.Inf(1), minS: minS}
				}
			}
		case DecimalKind:
			min, err := decimal.Parse(minimum, precision, scale)
			if err != nil {
				return Type{}, errors.New("minimum is out of range")
			}
			dr := t.vl.(decimalRange)
			if min.Less(dr.min) || min.Greater(dr.max) {
				return Type{}, errors.New("minimum is out of range")
			}
			if !min.Equal(dr.min) {
				t.vl = decimalRange{min, dr.max}
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
					maxS := strconv.FormatFloat(max, 'f', -1, 64)
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
					maxS := strconv.FormatFloat(max, 'f', -1, 32)
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
			max, err := decimal.Parse(maximum, precision, scale)
			if err != nil {
				return Type{}, errors.New("maximum is out of range")
			}
			dr := t.vl.(decimalRange)
			if max.Less(dr.min) {
				return Type{}, errors.New("maximum cannot be less than minimum")
			}
			if max.Greater(dr.max) {
				return Type{}, errors.New("maximum is out of range")
			}
			if !max.Equal(dr.max) {
				t.vl = decimalRange{dr.min, max}
			}
		default:
			return Type{}, errors.New("unexpected maximum for non-number type")
		}
	}
	if hasReal {
		if t.kind != FloatKind {
			return Type{}, errors.New("unexpected real for non-float type")
		}
		t.real = real
	}
	if elementType.Valid() || elementType.Generic() {
		if t.kind != ArrayKind && t.kind != MapKind {
			return Type{}, errors.New("unexpected element type for non-array and non-map type")
		}
		t.generic = elementType.generic
		t.vl = elementType
	} else {
		if t.kind == ArrayKind || t.kind == MapKind {
			return Type{}, errors.New("missing element type")
		}
	}
	if hasMinElements {
		if t.kind != ArrayKind {
			return Type{}, errors.New("unexpected minElements for non-array type")
		}
		t.p = int32(minElements)
	}
	if maxElements < MaxElements {
		if t.kind != ArrayKind {
			return Type{}, errors.New("unexpected maxElements for non-array type")
		}
		if maxElements < minElements {
			return Type{}, errors.New("maxElements must be greater or equal to minElements")
		}
	}
	if t.kind == ArrayKind {
		t.s = int32(maxElements)
	}
	if hasUniqueElements {
		if t.kind != ArrayKind {
			return Type{}, errors.New("unexpected uniqueElements for non-array type")
		}
		if k := t.vl.(Type).kind; k == JSONKind || k == ArrayKind || k == MapKind || k == ObjectKind {
			return Type{}, errors.New("unexpected uniqueElements for elements with type json, array, map, or object")
		}
		t.unique = uniqueElements
	}
	if properties == nil {
		if t.kind == ObjectKind {
			return Type{}, errors.New("missing object properties")
		}
	} else {
		if t.kind != ObjectKind {
			return Type{}, errors.New("unexpected properties for non-object type")
		}
		for _, p := range properties {
			if p.Type.generic {
				t.generic = true
				break
			}
		}
		names := make(map[string]int, len(properties))
		for i, p := range properties {
			names[p.Name] = i
		}
		t.vl = Properties{properties: properties, names: names}
	}

	return t, nil
}

// unmarshalProperty reads the JSON tokens from dec, which must have already
// read the token '{', and returns the decoded property.
func unmarshalProperty(dec *json.Decoder) (Property, error) {

	var p Property
	var hasPrefilled, hasCreateRequired, hasUpdateRequired, hasReadOptional, hasNullable, hasDescription bool

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
			if p.Type.Valid() || p.Type.Generic() {
				return Property{}, errors.New("repeated 'type' key")
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
				return Property{}, errors.New("repeated 'name' key")
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
		case "prefilled":
			if hasPrefilled {
				return Property{}, errors.New("repeated 'prefilled' key")
			}
			p.Prefilled, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property prefilled")
			}
			hasPrefilled = true
		case "createRequired":
			if hasCreateRequired {
				return Property{}, errors.New("repeated 'createRequired' key")
			}
			p.CreateRequired, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'createRequired' key of property")
			}
			hasCreateRequired = true
		case "updateRequired":
			if hasUpdateRequired {
				return Property{}, errors.New("repeated 'updateRequired' key")
			}
			p.UpdateRequired, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'updateRequired' key of property")
			}
			hasUpdateRequired = true
		case "readOptional":
			if hasReadOptional {
				return Property{}, errors.New("repeated 'readOptional' key")
			}
			p.ReadOptional, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'readOptional' key of property")
			}
			hasReadOptional = true
		case "nullable":
			if hasNullable {
				return Property{}, errors.New("repeated 'nullable' key")
			}
			p.Nullable, ok = tok.(bool)
			if !ok {
				return Property{}, errors.New("unexpected value for 'nullable' key of property")
			}
			hasNullable = true
		case "description":
			if hasDescription {
				return Property{}, errors.New("repeated 'description' key")
			}
			p.Description, ok = tok.(string)
			if !ok {
				return Property{}, errors.New("unexpected value for property description")
			}
			hasDescription = true
		default:
			return Property{}, fmt.Errorf("unknown property '%s'", key)
		}

	}

	if p.Name == "" {
		return Property{}, errors.New("missing property name")
	}
	if !p.Type.Valid() && !p.Type.Generic() {
		return Property{}, errors.New("missing property type")
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
