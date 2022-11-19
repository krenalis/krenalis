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
)

// MarshalJSON marshals t into JSON.
func (t Type) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
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
			b.WriteString(strconv.Itoa(t.p))
		}
		if t.s > 0 {
			b.WriteString(`,"scale":`)
			b.WriteString(strconv.Itoa(t.s))
		}
	case PtText:
		if t.p > 0 {
			b.WriteString(`,"byteLen":`)
			b.WriteString(strconv.Itoa(t.p))
		}
		if t.s > 0 {
			b.WriteString(`,"charLen":`)
			b.WriteString(strconv.Itoa(t.s))
		}
		switch vl := t.vl.(type) {
		case *regexp.Regexp:
			b.WriteString(`,"regexp":"`)
			b.WriteString(vl.String())
			b.WriteString(`"`)
		case []string:
			b.WriteString(`,"values":`)
			values, err := json.Marshal(vl)
			if err != nil {
				return nil, err
			}
			b.Write(values)
		}
	case PtArray:
		b.WriteString(`,"items":`)
		p, err := t.vl.(Type).MarshalJSON()
		if err != nil {
			return nil, err
		}
		b.Write(p)
	case PtObject:
		b.WriteString(`,"properties":[`)
		properties := t.vl.([]Property)
		for i, field := range properties {
			if i > 0 {
				b.WriteString(",")
			}
			p, err := json.Marshal(field)
			if err != nil {
				return nil, err
			}
			b.Write(p)
		}
		b.WriteString("]")
	}
	b.WriteString(`}`)
	return b.Bytes(), nil
}

// UnmarshalJSON parses the JSON-encoded data and returns the decoded type.
func UnmarshalJSON(data []byte) (Type, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	t, err := unmarshalType(dec)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return t, err
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the type
// pointed by t.
func (t *Type) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	t2, err := unmarshalType(dec)
	if err != nil {
		return err
	}
	*t = t2
	return nil
}

// unmarshalType reads the JSON tokens from dec and returns the decoded type.
func unmarshalType(dec *json.Decoder) (Type, error) {

	var t Type

	// Read delimiter '{'.
	tok, err := dec.Token()
	if err != nil {
		return t, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return t, errors.New("invalid type syntax")
	}

	// Unmarshal any other type.
	var precision, scale, byteLen, charLen int
	var re *regexp.Regexp
	var values []string
	var items Type
	var properties []Property

	// Read type keys and values.
	for {

		// Read the key or the delimiter '}'.
		tok, err = dec.Token()
		if err != nil {
			return t, err
		}
		if _, ok = tok.(json.Delim); ok {
			break
		}
		key := tok.(string)

		if key == "items" {
			items, err = unmarshalType(dec)
			if err != nil {
				return t, err
			}
			continue
		}

		// Read the value.
		tok, err = dec.Token()
		if err != nil {
			return t, err
		}

		switch key {
		case "name":
			if name, ok := tok.(string); ok {
				for i, n := range physicalName {
					if n == name {
						t.pt = PhysicalType(i + 1)
						break
					}
				}
			}
			if t.pt == 0 {
				return t, errors.New("invalid physical type")
			}
		case "logical":
			if name, ok := tok.(string); ok {
				for i, n := range logicalName {
					if n == name {
						t.lt = LogicalType(i + 1)
						break
					}
				}
			}
			if t.lt == 0 {
				return t, errors.New("invalid logical type")
			}
		case "regexp":
			if expr, ok := tok.(string); ok {
				re, err = regexp.Compile(expr)
			}
			if re == nil {
				return t, errors.New("invalid regular expression")
			}
			if values != nil {
				return t, errors.New("regular expression cannot be provided if values is provided")
			}
		case "values":
			if d, ok := tok.(json.Delim); !ok || d != '[' {
				return t, errors.New("invalid values")
			}
			if values != nil {
				return t, errors.New("repeated 'values' key")
			}
		Values:
			for {
				tok, err = dec.Token()
				if err != nil {
					return t, err
				}
				switch v := tok.(type) {
				case string:
					values = append(values, v)
				case json.Delim:
					break Values
				default:
					return t, errors.New("invalid value in values")
				}
			}
			if len(values) == 0 {
				return t, errors.New("invalid empty values")
			}
			if re != nil {
				return t, errors.New("values cannot be provided if regular expression is provided")
			}
		case "precision":
			n, ok := tok.(json.Number)
			if !ok {
				return t, errors.New("invalid precision")
			}
			precision, _ = strconv.Atoi(string(n))
			if precision <= 0 || precision > MaxDecimalPrecision {
				return t, errors.New("invalid precision")
			}
		case "scale":
			n, ok := tok.(json.Number)
			if !ok {
				return t, errors.New("invalid scale")
			}
			scale, _ = strconv.Atoi(string(n))
			if scale <= 0 || scale > MaxDecimalScale {
				return t, errors.New("invalid scale")
			}
		case "byteLen":
			n, ok := tok.(json.Number)
			if !ok {
				return t, errors.New("invalid length in bytes")
			}
			byteLen, _ = strconv.Atoi(string(n))
			if byteLen <= 0 {
				return t, errors.New("invalid length in bytes")
			}
		case "charLen":
			n, ok := tok.(json.Number)
			if !ok {
				return t, errors.New("invalid length in characters")
			}
			charLen, _ = strconv.Atoi(string(n))
			if charLen <= 0 {
				return t, errors.New("invalid length in characters")
			}
		case "properties":
			if d, ok := tok.(json.Delim); !ok || d != '[' {
				return t, errors.New("invalid properties")
			}
		Properties:
			for {
				var property Property

				// Read delimiter '{' or ']'.
				tok, err = dec.Token()
				if err != nil {
					return t, err
				}
				delim, ok = tok.(json.Delim)
				if !ok {
					return t, errors.New("invalid struct syntax")
				}
				if delim == ']' {
					break Properties
				}
				if delim != '{' {
					return t, errors.New("invalid struct syntax")
				}

				// Read property keys and values.
				for {

					// Read a key or delimiter '}'.
					tok, err = dec.Token()
					if err != nil {
						return t, err
					}
					if _, ok := tok.(json.Delim); ok {
						break
					}
					key := tok.(string)

					if key == "type" {
						property.Type, err = unmarshalType(dec)
						if err != nil {
							return t, err
						}
						continue
					}

					// Read the value.
					tok, err = dec.Token()
					if err != nil {
						return t, err
					}

					switch key {
					case "name":
						property.Name, ok = tok.(string)
						if !ok {
							return t, errors.New("unexpected value for property name")
						}
						if property.Name == "" {
							return t, errors.New("unexpected empty property name")
						}
					case "label":
						property.Label, ok = tok.(string)
						if !ok {
							return t, errors.New("unexpected value for property label")
						}
						if property.Label == "" {
							return t, errors.New("unexpected empty property label")
						}
					case "description":
						property.Description, ok = tok.(string)
						if !ok {
							return t, errors.New("unexpected value for property description")
						}
						if property.Description == "" {
							return t, errors.New("unexpected empty property description")
						}
					default:
						return t, errors.New("unknown property key")
					}

				}

				if property.Name == "" {
					return t, errors.New("missing property name")
				}
				if property.Type.pt == 0 {
					return t, errors.New("missing property type")
				}

				properties = append(properties, property)
			}
		default:
			return t, errors.New("unknown key")
		}

	}

	if t.pt == 0 {
		return Type{}, errors.New("missing physical type")
	}
	if re != nil {
		if t.pt != PtText {
			return Type{}, errors.New("unexpected regular expression for no Text type")
		}
		t.vl = re
	}
	if values != nil {
		if t.pt != PtText {
			return Type{}, errors.New("unexpected values for no Text type")
		}
		t.vl = values
	}
	if byteLen > 0 {
		if t.pt != PtText {
			return Type{}, errors.New("unexpected length in bytes for no Text type")
		}
		t.p = byteLen
	}
	if charLen > 0 {
		if t.pt != PtText {
			return Type{}, errors.New("unexpected length in characters for no Text type")
		}
		t.s = charLen
	}
	if precision > 0 {
		if t.pt != PtDecimal {
			return Type{}, errors.New("unexpected precision for no Decimal type")
		}
		t.p = precision
	}
	if scale > 0 {
		if t.pt != PtDecimal {
			return Type{}, errors.New("unexpected scale for no Decimal type")
		}
		if precision < scale {
			if precision == 0 {
				return Type{}, errors.New("with scale, precision is required")
			}
			return Type{}, errors.New("precision cannot not be less than scale")
		}
		t.s = scale
	}
	if items.pt == 0 {
		if t.pt == PtArray {
			return Type{}, errors.New("missing array items type")
		}
	} else {
		if t.pt != PtArray {
			return Type{}, errors.New("unexpected items for no Array type")
		}
		t.vl = items
	}
	if properties == nil {
		if t.pt == PtObject {
			return Type{}, errors.New("missing object properties")
		}
	} else {
		if t.pt != PtObject {
			return Type{}, errors.New("unexpected properties for no Object type")
		}
		t.vl = properties
	}

	return t, nil
}
