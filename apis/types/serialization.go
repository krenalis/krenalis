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
	"io"
	"regexp"
	"strconv"
	"strings"
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
	case PtDecimal:
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
			b.WriteString(`,"values":[`)
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
		b.WriteString(`,"items":`)
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

	var pt PhysicalType
	var lt LogicalType
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var values []string
	var items Type
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

		if key == "items" {
			items, err = unmarshalType(dec, resolve)
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
		case "regexp":
			if re != nil {
				return Type{}, errors.New("repeated 'regexp' key")
			}
			if values != nil {
				return Type{}, errors.New("regular expression cannot be provided if values are provided")
			}
			if expr, ok := tok.(string); ok {
				re, err = regexp.Compile(expr)
			}
			if re == nil {
				return Type{}, errors.New("invalid regular expression")
			}
		case "values":
			if values != nil {
				return Type{}, errors.New("repeated 'values' key")
			}
			if re != nil {
				return Type{}, errors.New("values cannot be provided if regular expression is provided")
			}
			if d, ok := tok.(json.Delim); !ok || d != '[' {
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
					return Type{}, errors.New("invalid value in values")
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
			if scale > 0 {
				return Type{}, errors.New("repeated 'scale' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid scale")
			}
			scale, _ = strconv.Atoi(string(n))
			if scale <= 0 || scale > MaxDecimalScale {
				return Type{}, errors.New("invalid scale")
			}
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
			if minItems > 0 {
				return Type{}, errors.New("repeated 'minItems' key")
			}
			n, ok := tok.(json.Number)
			if !ok {
				return Type{}, errors.New("invalid minimum items")
			}
			minItems, _ = strconv.Atoi(string(n))
			if minItems <= 0 || minItems > MaxArrayLen {
				return Type{}, errors.New("invalid minimum items")
			}
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
			if uniqueItems == true {
				return Type{}, errors.New("repeated 'uniqueItems' key")
			}
			uniqueItems, ok = tok.(bool)
			if !ok || !uniqueItems {
				return Type{}, errors.New("invalid unique items")
			}
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
			return Type{}, errors.New("unknown key")
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
	if re != nil {
		if pt != PtText {
			return Type{}, errors.New("unexpected regular expression for no Text type")
		}
		t.vl = re
	}
	if values != nil {
		if pt != PtText {
			return Type{}, errors.New("unexpected values for no Text type")
		}
		t.vl = values
	}
	if byteLen > 0 {
		if pt != PtText {
			return Type{}, errors.New("unexpected length in bytes for no Text type")
		}
		t.p = int32(byteLen)
	}
	if charLen > 0 {
		if pt != PtText {
			return Type{}, errors.New("unexpected length in characters for no Text type")
		}
		t.s = int32(charLen)
	}
	if precision > 0 {
		if pt != PtDecimal {
			return Type{}, errors.New("unexpected precision for no Decimal type")
		}
		t.p = int32(precision)
	}
	if scale > 0 {
		if pt != PtDecimal {
			return Type{}, errors.New("unexpected scale for no Decimal type")
		}
		if precision < scale {
			if precision == 0 {
				return Type{}, errors.New("with scale, precision is required")
			}
			return Type{}, errors.New("precision cannot not be less than scale")
		}
		t.s = int32(scale)
	}
	if items.Valid() {
		if pt != PtArray {
			return Type{}, errors.New("unexpected items for no Array type")
		}
		t.vl = items
	} else {
		if pt == PtArray {
			return Type{}, errors.New("missing array items type")
		}
	}
	if minItems > 0 {
		if pt != PtArray {
			return Type{}, errors.New("unexpected minItems for no Array type")
		}
		t.p = int32(minItems)
	}
	if maxItems < MaxArrayLen {
		if pt != PtArray {
			return Type{}, errors.New("unexpected maxItems for no Array type")
		}
		if maxItems < minItems {
			return Type{}, errors.New("maxItems must be greater or equal to minItems")
		}
	}
	if pt == PtArray {
		t.s = int32(maxItems)
	}
	if uniqueItems {
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
			return Type{}, errors.New("unexpected properties for no Object type")
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
