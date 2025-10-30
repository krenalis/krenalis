// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	jsonstd "encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
)

// An encoder writes JSON values to a slice of bytes.
type encoder struct {
	indent             bool
	generateASCII      bool
	allowSpecialFloats bool
	keys               []string
	depth              int
}

// newEncoder returns a new encoder.
func newEncoder(indent, generateASCII, allowSpecialFloats bool) *encoder {
	return &encoder{
		indent:             indent,
		generateASCII:      generateASCII,
		allowSpecialFloats: allowSpecialFloats,
	}
}

// Append appends the JSON representation of v of type t to b, and returns the
// extended buffer.
func (enc *encoder) Append(b []byte, t types.Type, v any) []byte {
	if v == nil {
		return append(b, "null"...)
	}
	switch k := t.Kind(); k {
	case types.TextKind:
		return enc.appendString(b, v.(string))
	case types.BooleanKind:
		return strconv.AppendBool(b, v.(bool))
	case types.IntKind, types.YearKind:
		return strconv.AppendInt(b, int64(v.(int)), 10)
	case types.UintKind:
		return strconv.AppendUint(b, uint64(v.(uint)), 10)
	case types.FloatKind:
		v := v.(float64)
		switch {
		case math.IsNaN(v):
			if enc.allowSpecialFloats {
				return append(b, "NaN"...)
			}
			return append(b, "null"...)
		case math.IsInf(v, 1):
			if enc.allowSpecialFloats {
				return append(b, "Infinity"...)
			}
			return append(b, "null"...)
		case math.IsInf(v, -1):
			if enc.allowSpecialFloats {
				return append(b, "-Infinity"...)
			}
			return append(b, "null"...)
		}
		return enc.appendFloat(b, v, t.BitSize())
	case types.DecimalKind:
		return append(b, v.(decimal.Decimal).String()...)
	case types.DateTimeKind:
		b = append(b, '"')
		b = v.(time.Time).AppendFormat(b, time.RFC3339Nano)
		return append(b, '"')
	case types.DateKind:
		b = append(b, '"')
		b = v.(time.Time).AppendFormat(b, time.DateOnly)
		return append(b, '"')
	case types.TimeKind:
		b = append(b, '"')
		b = v.(time.Time).AppendFormat(b, "15:04:05.999999999Z")
		return append(b, '"')
	case types.UUIDKind, types.InetKind:
		b = append(b, '"')
		b = append(b, v.(string)...)
		return append(b, '"')
	case types.JSONKind:
		dec := jsonstd.NewDecoder(bytes.NewReader(v.(json.Value)))
		dec.UseNumber()
		var jv any
		err := dec.Decode(&jv)
		if err != nil {
			panic(err)
		}
		return enc.appendJSONValue(b, jv)
	case types.ArrayKind:
		v := v.([]any)
		b = append(b, '[')
		if len(v) == 0 {
			return append(b, ']')
		}
		itemType := t.Elem()
		enc.depth++
		for i, v := range v {
			if i > 0 {
				b = append(b, ',')
			}
			if enc.indent {
				b = enc.appendIndentation(b)
			}
			b = enc.Append(b, itemType, v)
		}
		enc.depth--
		if enc.indent {
			b = enc.appendIndentation(b)
		}
		return append(b, ']')
	case types.ObjectKind:
		b = append(b, '{')
		enc.depth++
		switch v := v.(type) {
		case []any:
			for i, p := range t.Properties().All() {
				if i > 0 {
					b = append(b, ',')
				}
				if enc.indent {
					b = enc.appendIndentation(b)
				}
				b = append(b, '"')
				b = append(b, p.Name...)
				b = append(b, '"', ':')
				if enc.indent {
					b = append(b, ' ')
				}
				b = enc.Append(b, p.Type, v[i])
			}
		case map[string]any:
			var i int
			for _, p := range t.Properties().All() {
				x, ok := v[p.Name]
				if !ok {
					continue
				}
				if i > 0 {
					b = append(b, ',')
				}
				if enc.indent {
					b = enc.appendIndentation(b)
				}
				b = append(b, '"')
				b = append(b, p.Name...)
				b = append(b, '"', ':')
				if enc.indent {
					b = append(b, ' ')
				}
				b = enc.Append(b, p.Type, x)
				i++
			}
		}
		enc.depth--
		if enc.indent {
			b = enc.appendIndentation(b)
		}
		return append(b, '}')
	case types.MapKind:
		v := v.(map[string]any)
		b = append(b, '{')
		if len(v) == 0 {
			return append(b, '}')
		}
		vType := t.Elem()
		enc.depth++
		for i, key := range enc.sortKeys(v) {
			if i > 0 {
				b = append(b, ',')
			}
			if enc.indent {
				b = enc.appendIndentation(b)
			}
			b = enc.appendString(b, key)
			b = append(b, ':')
			if enc.indent {
				b = append(b, ' ')
			}
			b = enc.Append(b, vType, v[key])
		}
		enc.depth--
		if enc.indent {
			b = enc.appendIndentation(b)
		}
		return append(b, '}')
	}
	panic(fmt.Sprintf("unexpected type %s", t))
}

// sortKeys returns the keys of v sorted. The returned slice can be used until
// the next call to sortKeys.
func (enc *encoder) sortKeys(v map[string]any) []string {
	if cap(enc.keys) < len(v) {
		enc.keys = make([]string, len(v))
	} else {
		enc.keys = enc.keys[0:len(v)]
	}
	i := 0
	for key := range v {
		enc.keys[i] = key
		i++
	}
	slices.Sort(enc.keys)
	return enc.keys
}

// appendJSONValue appends the JSON representation of v to b, and returns the
// extended buffer. v is a value returned by the Decoder of the json package
// with numbers decoded as Number.
func (enc *encoder) appendJSONValue(b []byte, v any) []byte {
	if v == nil {
		return append(b, "null"...)
	}
	switch v := v.(type) {
	case string:
		return enc.appendString(b, v)
	case jsonstd.Number:
		return append(b, v...)
	case bool:
		return strconv.AppendBool(b, v)
	case []any:
		b = append(b, '[')
		if len(v) == 0 {
			return append(b, ']')
		}
		enc.depth++
		for i, v := range v {
			if i > 0 {
				b = append(b, ',')
			}
			if enc.indent {
				b = enc.appendIndentation(b)
			}
			b = enc.appendJSONValue(b, v)
		}
		enc.depth--
		if enc.indent {
			b = enc.appendIndentation(b)
		}
		return append(b, ']')
	case map[string]any:
		b = append(b, '{')
		if len(v) == 0 {
			return append(b, '}')
		}
		enc.depth++
		for i, key := range enc.sortKeys(v) {
			if i > 0 {
				b = append(b, ',')
			}
			if enc.indent {
				b = enc.appendIndentation(b)
			}
			b = enc.appendString(b, key)
			b = append(b, ':')
			if enc.indent {
				b = append(b, ' ')
			}
			b = enc.appendJSONValue(b, v[key])
		}
		enc.depth--
		if enc.indent {
			b = enc.appendIndentation(b)
		}
		return append(b, '}')
	}
	panic(fmt.Sprintf("unexpected JSON value %#v", v))
}

// appendIndentation appends a new line and one or more tabs according to
// enc.depth.
func (enc *encoder) appendIndentation(b []byte) []byte {
	b = append(b, '\n')
	for i := 0; i < enc.depth; i++ {
		b = append(b, '\t')
	}
	return b
}

// appendFloat appends the JSON representation of f to b and returns the
// extended buffer.
//
// The code of this method is taken form the floatEncoder.encode method of the
// json package in the Go standard library, and it is copyright The Go Authors.
func (enc *encoder) appendFloat(b []byte, f float64, bits int) []byte {
	// Convert as if by ES6 number to string conversion.
	// This matches most other JSON generators.
	// See golang.org/issue/6384 and golang.org/issue/14135.
	// Like fmt %g, but the exponent cutoffs are different
	// and exponents themselves are not padded to two digits.
	abs := math.Abs(f)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if bits == 64 && (abs < 1e-6 || abs >= 1e21) || bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	b = strconv.AppendFloat(b, f, fmt, -1, int(bits))
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	return b
}

// appendString escapes s so that it can be safely placed within a JSON string.
// The escaped value is then appended to b and returns the extended buffer.
func (enc *encoder) appendString(b []byte, s string) []byte {
	b = append(b, '"')
	last := 0
	for i, r := range s {
		if enc.generateASCII && r > utf8.RuneSelf || r == '\u2028' || r == '\u2029' {
			if last != i {
				b = append(b, s[last:i]...)
			}
			if r < 0x10000 {
				b = appendRune(b, r)
			} else {
				r1, r2 := utf16.EncodeRune(r)
				b = appendRune(b, r1)
				b = appendRune(b, r2)
			}
			last = i + utf8.RuneLen(r)
		} else if int(r) < len(stringEscapes) {
			esc := stringEscapes[r]
			if esc != "" {
				if last != i {
					b = append(b, s[last:i]...)
				}
				b = append(b, stringEscapes[r]...)
				last = i + 1
			}
		}
	}
	if last != len(s) {
		b = append(b, s[last:]...)
	}
	return append(b, '"')
}

// appendRune escapes r with r < 0x10000, appends the escaped value to b and
// returns the extended buffer.
func appendRune(b []byte, r rune) []byte {
	b = append(b, '\\', 'u')
	for s := 12; s >= 0; s -= 4 {
		const hex = "0123456789abcdef"
		b = append(b, hex[r>>uint(s)&0xF])
	}
	return b
}

// stringEscapes contains the runes that must be escaped when placed within a
// JSON string, in addition to the runes U+2028 and U+2029.
var stringEscapes = []string{
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
