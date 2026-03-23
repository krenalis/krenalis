// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// Marshal encodes records, based on the schema of their elements into a
// JavaScript array or a Python list, and appends it to b. The resulting value
// can be included in a JSON string without escaping.
//
// schema must be an object or invalid. If it is invalid, values is marshaled as
// an array of empty objects.
//
// Marshalled Python type names are fully qualified, meaning they also include
// the module name (e.g., 'decimal.Decimal', not just 'Decimal').
//
// Unlike Unmarshal, Marshal does not validate the values against the schema.
// The values must already be validated.
func Marshal(b []byte, schema types.Type, records []Record, language state.Language, preserveJSON bool) ([]byte, error) {
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("core/transformers: schema is not an object")
	}
	var marshal func([]byte, types.Type, any, bool) ([]byte, error)
	switch language {
	case state.JavaScript:
		marshal = marshalJavaScript
	case state.Python:
		marshal = marshalPython
	default:
		return nil, errors.New("core/transformers: language is not valid")
	}
	var err error
	b = append(b, '[')
	for i, v := range records {
		if i > 0 {
			b = append(b, ',')
		}
		if schema.Valid() && len(v.Attributes) > 0 {
			b, err = marshal(b, schema, v.Attributes, preserveJSON)
			if err != nil {
				return nil, err
			}
		} else {
			b = append(b, "{}"...)
		}
	}
	return append(b, ']'), nil
}

// marshalJavaScript marshals v as a JavaScript value.
func marshalJavaScript(b []byte, t types.Type, v any, preserveJSON bool) ([]byte, error) {
	if v == nil {
		return append(b, "null"...), nil
	}
	k := t.Kind()
	if k == types.JSONKind {
		value := v.(json.Value)
		if preserveJSON {
			b = append(b, '\'')
			b = jsStringEscape(b, string(value))
			b = append(b, '\'')
			return b, nil
		}
		var comma bool
		dec := json.NewDecoder(bytes.NewReader(value))
		for {
			kind := dec.PeekKind()
			if kind == json.Invalid {
				break
			}
			if comma && kind != '}' && kind != ']' {
				b = append(b, ',')
			}
			comma = true
			switch kind {
			case '"':
				tok, _ := dec.ReadToken()
				b = append(b, '\'')
				b = jsStringEscape(b, tok.String())
				b = append(b, '\'')
				if dec.IsKey() {
					b = append(b, ':')
					comma = false
				}
			case '{', '}', '[', ']':
				b = append(b, byte(kind))
				_ = dec.SkipToken()
				if kind == '{' || kind == '[' {
					comma = false
				}
			default:
				v, _ := dec.ReadValue()
				b = append(b, v...)
			}
		}
		return b, nil
	}
	switch v := v.(type) {
	case string:
		b = append(b, '\'')
		b = jsStringEscape(b, v)
		b = append(b, '\'')
	case bool:
		if v {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case int:
		b = strconv.AppendInt(b, int64(v), 10)
		if k == types.IntKind && t.BitSize() == 64 {
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
		b = append(b, '\'')
		switch k {
		case types.DateTimeKind:
			b = v.AppendFormat(b, "2006-01-02T15:04:05.999999999Z")
		case types.DateKind:
			b = v.AppendFormat(b, "2006-01-02")
		case types.TimeKind:
			b = v.AppendFormat(b, "15:04:05.999999999")
		}
		b = append(b, '\'')
	case []any:
		b = append(b, '[')
		elem := t.Elem()
		n := len(v)
		var err error
		for i := range n {
			if i > 0 {
				b = append(b, ',')
			}
			b, err = marshalJavaScript(b, elem, v[i], preserveJSON)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, ']')
	case map[string]any:
		var err error
		b = append(b, '{')
		i := 0
		if k == types.ObjectKind {
			for _, p := range t.Properties().All() {
				e, ok := v[p.Name]
				if !ok {
					continue
				}
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, p.Name...)
				b = append(b, ':')
				b, err = marshalJavaScript(b, p.Type, e, preserveJSON)
				if err != nil {
					return nil, err
				}
				i++
			}
		} else {
			elem := t.Elem()
			for k, e := range v {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = jsStringEscape(b, k)
				b = append(b, '\'', ':')
				b, err = marshalJavaScript(b, elem, e, preserveJSON)
				if err != nil {
					return nil, err
				}
				i++
			}
		}
		b = append(b, '}')
	default:
		return nil, fmt.Errorf("core/transformers: unexpected type %s", t)
	}
	return b, nil
}

// marshalPython marshals v as a Python value.
//
// Marshalled Python type names are fully qualified, meaning they also include
// the module name (e.g., 'decimal.Decimal', not just 'Decimal').
func marshalPython(b []byte, t types.Type, v any, preserveJSON bool) ([]byte, error) {
	if v == nil {
		return append(b, "None"...), nil
	}
	k := t.Kind()
	if k == types.JSONKind {
		value := v.(json.Value)
		if preserveJSON {
			b = append(b, '\'')
			b = pyStringEscape(b, string(value))
			b = append(b, '\'')
			return b, nil
		}
		var comma bool
		dec := json.NewDecoder(bytes.NewReader(value))
		for {
			kind := dec.PeekKind()
			if kind == json.Invalid {
				break
			}
			if comma && kind != '}' && kind != ']' {
				b = append(b, ',')
			}
			comma = true
			switch kind {
			case 'n':
				b = append(b, "None"...)
				_ = dec.SkipToken()
			case 't':
				b = append(b, "True"...)
				_ = dec.SkipToken()
			case 'f':
				b = append(b, "False"...)
				_ = dec.SkipToken()
			case '"':
				tok, _ := dec.ReadToken()
				b = append(b, '\'')
				b = pyStringEscape(b, tok.String())
				b = append(b, '\'')
				if dec.IsKey() {
					b = append(b, ':')
					comma = false
				}
			case '{', '}', '[', ']':
				b = append(b, byte(kind))
				_ = dec.SkipToken()
				if kind == '{' || kind == '[' {
					comma = false
				}
			default:
				v, _ := dec.ReadValue()
				b = append(b, v...)
			}
		}
		return b, nil
	}
	switch v := v.(type) {
	case string:
		if k == types.UUIDKind {
			b = append(b, "uuid.UUID('"...)
			b = append(b, v...)
			b = append(b, '\'', ')')
		} else {
			b = append(b, '\'')
			b = pyStringEscape(b, v)
			b = append(b, '\'')
		}
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
		b = append(b, "decimal.Decimal('"...)
		b = append(b, v.String()...)
		b = append(b, '\'', ')')
	case time.Time:
		switch k {
		case types.DateTimeKind:
			b = fmt.Appendf(b, "datetime.datetime(%d,%d,%d,%d,%d,%d,%d)", v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		case types.DateKind:
			b = fmt.Appendf(b, "datetime.date(%d,%d,%d)", v.Year(), v.Month(), v.Day())
		case types.TimeKind:
			b = fmt.Appendf(b, "datetime.time(%d,%d,%d,%d)", v.Hour(), v.Minute(), v.Second(), v.Nanosecond()/1000)
		}
	case []any:
		b = append(b, '[')
		elem := t.Elem()
		var err error
		n := len(v)
		for i := range n {
			if i > 0 {
				b = append(b, ',')
			}

			b, err = marshalPython(b, elem, v[i], preserveJSON)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, ']')
	case map[string]any:
		var err error
		b = append(b, '{')
		i := 0
		if k == types.ObjectKind {
			for _, p := range t.Properties().All() {
				e, ok := v[p.Name]
				if !ok {
					continue
				}
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = append(b, p.Name...)
				b = append(b, '\'', ':')
				b, err = marshalPython(b, p.Type, e, preserveJSON)
				if err != nil {
					return nil, err
				}
				i++
			}
		} else {
			elem := t.Elem()
			for k, e := range v {
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '\'')
				b = pyStringEscape(b, k)
				b = append(b, '\'', ':')
				b, err = marshalPython(b, elem, e, preserveJSON)
				if err != nil {
					return nil, err
				}
				i++
			}
		}
		b = append(b, '}')
	default:
		return nil, fmt.Errorf("core/transformers: unexpected type %s", k)
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
