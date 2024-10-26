//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package json

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json/internal/json/jsontext"
	"github.com/meergo/meergo/types"
)

// marshalBySchema marshals v as a JSON value and appends it to b.
func marshalBySchema(b []byte, v any, t types.Type) (Value, error) {
	if v == nil {
		return append(b, "null"...), nil
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
	case Value:
		b, _ = jsontext.AppendQuote(b, v)
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
				b, err = marshalBySchema(b, item, t.Elem())
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
				b, err = marshalBySchema(b, rv.Interface(), p.Type)
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
				b, err = marshalBySchema(b, e.v, vt)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, '}')
		default:
			return nil, fmt.Errorf("json: unexpected type %s", t)
		}
	}
	return b, nil
}
