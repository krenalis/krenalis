//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

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

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// Marshal encodes values, based on the schema of their elements into a
// JavaScript array or a Python list, and appends it to b. The resulting value
// can be included in a JSON string without escaping.
//
// schema must be an Object or invalid. If it is invalid, values is marshaled as
// an array of empty objects.
//
// Unlike Unmarshal, Marshal does not validate the values against the schema.
// The values must already be validated.
func Marshal(b []byte, schema types.Type, values []map[string]any, language state.Language) ([]byte, error) {
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("apis/transformers: schema is not an object")
	}
	var marshal func([]byte, types.Type, any) ([]byte, error)
	switch language {
	case state.JavaScript:
		marshal = marshalJavaScript
	case state.Python:
		marshal = marshalPython
	default:
		return nil, errors.New("apis/transformers: language is not valid")
	}
	var err error
	b = append(b, '[')
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		if schema.Valid() && len(v) > 0 {
			b, err = marshal(b, schema, v)
			if err != nil {
				return nil, err
			}
		} else {
			b = append(b, "{}"...)
		}
		i++
	}
	return append(b, ']'), nil
}

// marshalJavaScript marshals v as a JavaScript value.
func marshalJavaScript(b []byte, t types.Type, v any) ([]byte, error) {
	if t.Kind() == types.JSONKind {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		if err != nil {
			return nil, fmt.Errorf("apis/transformers: cannot marshal to JSON: %s", err)
		}
		s := buf.String()
		s = s[:len(s)-1]
		b = append(b, '\'')
		b = jsStringEscape(b, s)
		b = append(b, '\'')
		return b, nil
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
		if t.Kind() == types.IntKind && t.BitSize() == 64 {
			b = append(b, 'n')
		}
	case uint:
		b = strconv.AppendUint(b, uint64(v), 10)
		if t.BitSize() == 64 {
			b = append(b, 'n')
		}
	case float64:
		if math.IsNaN(v) {
			b = append(b, "NaN"...)
		} else if math.IsInf(v, 0) {
			if v > 0 {
				b = append(b, "Infinity"...)
			} else {
				b = append(b, "-Infinity"...)
			}
		} else {
			b = strconv.AppendFloat(b, v, 'g', -1, t.BitSize())
		}
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
				b, err = marshalJavaScript(b, t.Elem(), item)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, ']')
		case types.ObjectKind:
			b = append(b, '{')
			for i, p := range t.Properties() {
				rv := rv.MapIndex(reflect.ValueOf(p.Name))
				if !rv.IsValid() {
					return nil, fmt.Errorf("apis/transformers: missing property: %s", p.Name)
				}
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, p.Name...)
				b = append(b, ':')
				var err error
				b, err = marshalJavaScript(b, p.Type, rv.Interface())
				if err != nil {
					return nil, err
				}
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
			vt := t.Elem()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = jsStringEscape(b, e.k)
				b = append(b, '\'', ':')
				var err error
				b, err = marshalJavaScript(b, vt, e.v)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, '}')
		default:
			return nil, fmt.Errorf("apis/transformers: unexpected type %s", t)
		}
	}
	return b, nil
}

// marshalPython marshals v as a Python value.
func marshalPython(b []byte, t types.Type, v any) ([]byte, error) {
	k := t.Kind()
	if k == types.JSONKind {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		if err != nil {
			return nil, fmt.Errorf("apis/transformers: cannot marshal to JSON: %s", err)
		}
		s := buf.String()
		s = s[:len(s)-1]
		b = append(b, '\'')
		b = pyStringEscape(b, s)
		b = append(b, '\'')
		return b, nil
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
		if math.IsNaN(v) {
			b = append(b, "float('nan')"...)
		} else if math.IsInf(v, 0) {
			if v > 0 {
				b = append(b, "float('inf')"...)
			} else {
				b = append(b, "float('-inf')"...)
			}
		} else {
			b = strconv.AppendFloat(b, v, 'g', -1, t.BitSize())
		}
	case decimal.Decimal:
		b = append(b, "Decimal('"...)
		b = append(b, v.String()...)
		b = append(b, '\'', ')')
	case time.Time:
		switch k {
		case types.DateTimeKind:
			b = fmt.Appendf(b, "datetime(%d,%d,%d,%d,%d,%d,%d)", v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		case types.DateKind:
			b = fmt.Appendf(b, "date(%d,%d,%d)", v.Year(), v.Month(), v.Day())
		case types.TimeKind:
			b = fmt.Appendf(b, "time(%d,%d,%d,%d)", v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		}
	case string:
		if k == types.UUIDKind {
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
		switch k {
		case types.ArrayKind:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				var err error
				b, err = marshalPython(b, t.Elem(), item)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, ']')
		case types.ObjectKind:
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
					var err error
					b, err = marshalPython(b, p.Type, rv.Interface())
					if err != nil {
						return nil, err
					}
					i++
				}
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
			vt := t.Elem()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = pyStringEscape(b, e.k)
				b = append(b, '\'', ':')
				var err error
				b, err = marshalPython(b, vt, e.v)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, '}')
		default:
			return nil, fmt.Errorf("apis/transformers: unexpected type %s", k)
		}
	}
	return b, nil
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
