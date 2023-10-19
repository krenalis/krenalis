//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"encoding/json"
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

// MarshalPython marshals values, according to the schema of a single value,
// to a Python array and appends it into b. The encoded value can be put in a
// JSON string without escape.
func MarshalPython(b []byte, schema types.Type, values []map[string]any) []byte {
	b = append(b, '[')
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = marshalPython(b, schema, v)
		i++
	}
	return append(b, ']')
}

func marshalJavaScript(b []byte, t types.Type, v any) []byte {
	pt := t.PhysicalType()
	if pt == types.PtJSON {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		if err != nil {
			panic("unexpected value")
		}
		s := buf.String()
		s = s[:len(s)-1]
		b = append(b, '\'')
		b = jsStringEscape(b, s)
		b = append(b, '\'')
		return b
	}
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
		if pt == types.PtInt64 {
			b = append(b, 'n')
		}
	case uint:
		b = strconv.AppendUint(b, uint64(v), 10)
		if pt == types.PtUInt64 {
			b = append(b, 'n')
		}
	case float64:
		bs := 64
		if pt == types.PtFloat32 {
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
		switch pt {
		case types.PtArray:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				b = marshalJavaScript(b, t.Elem(), item)
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
			vt := t.Elem()
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
			panic(fmt.Sprintf("unexpected type %s", pt))
		}
	}
	return b
}

func marshalPython(b []byte, t types.Type, v any) []byte {
	pt := t.PhysicalType()
	if pt == types.PtJSON {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		if err != nil {
			panic("unexpected value")
		}
		s := buf.String()
		s = s[:len(s)-1]
		b = append(b, '\'')
		b = pyStringEscape(b, s)
		b = append(b, '\'')
		return b
	}
	switch v := v.(type) {
	case nil:
		b = append(b, "None"...)
	case bool:
		if v {
			b = append(b, "True"...)
		} else {
			b = append(b, "False"...)
		}
	case int:
		b = strconv.AppendInt(b, int64(v), 10)
	case uint:
		b = strconv.AppendUint(b, uint64(v), 10)
	case float64:
		bs := 64
		if pt == types.PtFloat32 {
			bs = 32
		}
		b = strconv.AppendFloat(b, v, 'g', -1, bs)
	case decimal.Decimal:
		b = append(b, "Decimal('"...)
		b = append(b, v.String()...)
		b = append(b, '\'', ')')
	case time.Time:
		switch pt {
		case types.PtDateTime:
			b = fmt.Appendf(b, "datetime(%d,%d,%d,%d,%d,%d,%d)", v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		case types.PtDate:
			b = fmt.Appendf(b, "date(%d,%d,%d)", v.Year(), v.Month(), v.Day())
		case types.PtTime:
			b = fmt.Appendf(b, "time(%d,%d,%d,%d)", v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		}
	case string:
		if pt == types.PtUUID {
			b = append(b, "UUID('"...)
			b = append(b, v...)
			b = append(b, '\'', ')')
		} else {
			b = append(b, '\'')
			b = pyStringEscape(b, v)
			b = append(b, '\'')
		}
	default:
		rv := reflect.ValueOf(v)
		switch pt {
		case types.PtArray:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				b = marshalPython(b, t.Elem(), item)
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
					b = append(b, '\'')
					b = append(b, p.Name...)
					b = append(b, '\'', ':')
					b = marshalPython(b, p.Type, rv.Interface())
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
			vt := t.Elem()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = pyStringEscape(b, e.k)
				b = append(b, '\'', ':')
				b = marshalPython(b, vt, e.v)
			}
			b = append(b, '}')
		default:
			panic(fmt.Sprintf("unexpected type %s", pt))
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

// pyStringEscapes contains the runes that must be escaped when placed within
// a Python and JSON string with single or double quotes, in addition to
// the runes U+2028 and U+2029.
var pyStringEscapes = []string{
	0:    `\x00`,
	1:    `\x01`,
	2:    `\x02`,
	3:    `\x03`,
	4:    `\x04`,
	5:    `\x05`,
	6:    `\x06`,
	7:    `\a`,
	'\b': `\b`,
	'\t': `\t`,
	'\n': `\n`,
	'\v': `\v`,
	'\f': `\f`,
	'\r': `\r`,
	14:   `\x0e`,
	15:   `\x0f`,
	16:   `\x10`,
	17:   `\x11`,
	18:   `\x12`,
	19:   `\x13`,
	20:   `\x14`,
	21:   `\x15`,
	22:   `\x16`,
	23:   `\x17`,
	24:   `\x18`,
	25:   `\x19`,
	26:   `\x1a`,
	27:   `\x1b`,
	28:   `\x1c`,
	29:   `\x1d`,
	30:   `\x1e`,
	31:   `\x1f`,
	'"':  `\"`,
	'&':  `\x26`,
	'\'': `\x27`,
	'<':  `\x3c`,
	'>':  `\x3e`,
	'\\': `\\`,
}

// pyStringEscape escapes the input string s to ensure it can be safely embedded
// within a Python code, whether enclosed in single or double quotes. The
// escaped string is stored in the byte slice b.
func pyStringEscape(b []byte, s string) []byte {
	last := 0
	for i, c := range s {
		var esc string
		switch {
		case int(c) < len(pyStringEscapes):
			esc = pyStringEscapes[c]
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
