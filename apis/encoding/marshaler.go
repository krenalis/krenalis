//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package encoding provides functions for unmarshalling data received from the
// API, validating it based on a specified schema, and marshalling data returned
// from the API according to a defined schema.
package encoding

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/types"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/shopspring/decimal"
)

// Marshal encodes the given value, based on its schema, into a JSON object,
// and returns it. The schema must be an Object.
//
// Unlike Unmarshal, this function does not validate the value. Its behavior is
// undefined if the value does not validate against the schema.
func Marshal(schema types.Type, value map[string]any) ([]byte, error) {
	if k := schema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("apis/encoding: schema is the invalid schema")
		}
		return nil, errors.New("apis/encoding: schema is not an object")
	}
	return marshal(nil, schema, value)
}

// MarshalSlice is like Marshal but encodes a slice of values as a JSON array.
func MarshalSlice(schema types.Type, values []map[string]any) ([]byte, error) {
	if k := schema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("apis/encoding: schema is the invalid schema")
		}
		return nil, errors.New("apis/encoding: schema is not an object")
	}
	return marshal(nil, types.Array(schema), values)
}

// marshal marshals v as a JSON value and appends it to b.
func marshal(b []byte, t types.Type, v any) ([]byte, error) {
	if v == nil {
		return append(b, "null"...), nil
	}
	if t.Kind() == types.JSONKind {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		if err != nil {
			return nil, fmt.Errorf("apis/encoding: cannot marshal to JSON: %s", err)
		}
		s := buf.String()
		s = s[:len(s)-1]
		b, err = jsontext.AppendQuote(b, s)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	switch v := v.(type) {
	case bool:
		if v {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case int:
		quoted := t.Kind() != types.YearKind && t.BitSize() == 64
		if quoted {
			b = append(b, '"')
		}
		b = strconv.AppendInt(b, int64(v), 10)
		if quoted {
			b = append(b, '"')
		}
	case uint:
		if t.BitSize() == 64 {
			b = append(b, '"')
		}
		b = strconv.AppendUint(b, uint64(v), 10)
		if t.BitSize() == 64 {
			b = append(b, '"')
		}
	case float64:
		if math.IsNaN(v) {
			b = append(b, `"NaN"`...)
		} else if math.IsInf(v, 0) {
			if v > 0 {
				b = append(b, `"Infinity"`...)
			} else {
				b = append(b, `"-Infinity"`...)
			}
		} else {
			b = strconv.AppendFloat(b, v, 'g', -1, t.BitSize())
		}
	case decimal.Decimal:
		b = append(b, '"')
		b = append(b, v.String()...)
		b = append(b, '"')
	case time.Time:
		b = append(b, '"')
		switch t.Kind() {
		case types.DateTimeKind:
			b = v.AppendFormat(b, time.RFC3339Nano)
		case types.DateKind:
			b = v.AppendFormat(b, time.DateOnly)
		case types.TimeKind:
			b = v.AppendFormat(b, "15:04:05.999999999")
		}
		b = append(b, '"')
	case string:
		var err error
		b, err = jsontext.AppendQuote(b, v)
		if err != nil {
			return nil, err
		}
	default:
		rv := reflect.ValueOf(v)
		switch t.Kind() {
		case types.ArrayKind:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				var err error
				b, err = marshal(b, t.Elem(), item)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, ']')
		case types.ObjectKind:
			b = append(b, '{')
			var err error
			i := 0
			for _, p := range t.Properties() {
				rv := rv.MapIndex(reflect.ValueOf(p.Name))
				if !rv.IsValid() {
					continue
				}
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '"')
				b = append(b, p.Name...)
				b = append(b, '"', ':')
				b, err = marshal(b, p.Type, rv.Interface())
				if err != nil {
					return nil, err
				}
				i++
			}
			b = append(b, '}')
		case types.MapKind:
			type entry struct {
				k string
				v any
			}
			s := make([]entry, rv.Len())
			iter := rv.MapRange()
			i := 0
			for iter.Next() {
				s[i].k = iter.Key().String()
				s[i].v = iter.Value().Interface()
				i++
			}
			slices.SortFunc(s, func(a, b entry) int {
				return strings.Compare(a.k, b.k)
			})
			var err error
			vt := t.Elem()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				b, err = jsontext.AppendQuote(b, e.k)
				if err != nil {
					return nil, err
				}
				b = append(b, ':')
				b, err = marshal(b, vt, e.v)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, '}')
		default:
			return nil, fmt.Errorf("apis/encoding: unexpected type %s", t)
		}
	}
	return b, nil
}
