//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

// MarshalJavaScript marshals values, according to the schema of a single value,
// to a JavaScript array and appends it into b. The encoded value can be put in
// a JSON string without escape.
func MarshalJavaScript(b []byte, schema types.Type, values []map[string]any) []byte {
	b = append(b, '[')
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = marshalJavaScript(b, schema, v)
		i++
	}
	return append(b, ']')
}

func marshalJavaScript(b []byte, t types.Type, v any) []byte {
	switch v := v.(type) {
	case nil:
		b = append(b, "null"...)
	case bool:
		if v {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case int:
		b = strconv.AppendInt(b, int64(v), 10)
		if t.PhysicalType() == types.PtInt64 {
			b = append(b, 'n')
		}
	case uint:
		b = strconv.AppendUint(b, uint64(v), 10)
		if t.PhysicalType() == types.PtUInt64 {
			b = append(b, 'n')
		}
	case float64:
		bs := 64
		if t.PhysicalType() == types.PtFloat32 {
			bs = 32
		}
		b = strconv.AppendFloat(b, v, 'g', -1, bs)
	case decimal.Decimal:
		b = append(b, '\'')
		b = append(b, v.String()...)
		b = append(b, '\'')
	case time.Time:
		b = append(b, "new Date("...)
		b = strconv.AppendInt(b, v.UnixMilli(), 10)
		b = append(b, ')')
	case string:
		b = append(b, '\'')
		b = jsStringEscape(b, v)
		b = append(b, '\'')
	default:
		rv := reflect.ValueOf(v)
		switch t.PhysicalType() {
		case types.PtArray:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				b = marshalJavaScript(b, t.ItemType(), item)
			}
			b = append(b, ']')
		case types.PtObject:
			b = append(b, '{')
			for i, p := range t.Properties() {
				if i > 0 {
					b = append(b, ',')
				}
				rv := rv.MapIndex(reflect.ValueOf(p.Name))
				if rv.IsValid() {
					b = append(b, p.Name...)
					b = append(b, ':')
					b = marshalJavaScript(b, p.Type, rv.Interface())
					i++
				}
			}
			b = append(b, '}')
		case types.PtMap:
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
			vt := t.ValueType()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = jsStringEscape(b, e.k)
				b = append(b, '\'', ':')
				b = marshalJavaScript(b, vt, e.v)
			}
			b = append(b, '}')
		default:
			panic(fmt.Sprintf("unexpected type %s", t.PhysicalType()))
		}
	}
	return b
}

// jsStringEscapes contains the runes that must be escaped when placed within
// a JavaScript and JSON string with single or double quotes, in addition to
// the runes U+2028 and U+2029.
var jsStringEscapes = []string{
	0:    `\u0000`,
	1:    `\u0001`,
	2:    `\u0002`,
	3:    `\u0003`,
	4:    `\u0004`,
	5:    `\u0005`,
	6:    `\u0006`,
	7:    `\u0007`,
	'\b': `\b`,
	'\t': `\t`,
	'\n': `\n`,
	'\v': `\u000b`,
	'\f': `\f`,
	'\r': `\r`,
	14:   `\u000e`,
	15:   `\u000f`,
	16:   `\u0010`,
	17:   `\u0011`,
	18:   `\u0012`,
	19:   `\u0013`,
	20:   `\u0014`,
	21:   `\u0015`,
	22:   `\u0016`,
	23:   `\u0017`,
	24:   `\u0018`,
	25:   `\u0019`,
	26:   `\u001a`,
	27:   `\u001b`,
	28:   `\u001c`,
	29:   `\u001d`,
	30:   `\u001e`,
	31:   `\u001f`,
	'"':  `\"`,
	'&':  `\u0026`,
	'\'': `\u0027`,
	'<':  `\u003c`,
	'>':  `\u003e`,
	'\\': `\\`,
}

// jsStringEscape escapes the input string s to ensure it can be safely embedded
// within a JavaScript, whether enclosed in single or double quotes. The escaped
// string is stored in the byte slice b.
func jsStringEscape(b []byte, s string) []byte {
	last := 0
	for i, c := range s {
		var esc string
		switch {
		case int(c) < len(jsStringEscapes):
			esc = jsStringEscapes[c]
		case c == '\u2028':
			esc = `\u2028`
		case c == '\u2029':
			esc = `\u2029`
		}
		if esc == "" {
			continue
		}
		if last != i {
			b = append(b, s[last:i]...)
		}
		b = append(b, esc...)
		if c == '\u2028' || c == '\u2029' {
			last = i + 3
		} else {
			last = i + 1
		}
	}
	if last != len(s) {
		b = append(b, s[last:]...)
	}
	return b
}
