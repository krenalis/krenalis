// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// This file is a copy of the file '/apis/warehouses/clickhouse/types.go'.
// Keep them synchronized.

package clickhouse

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/tools/types"
)

// columnType returns the types.Type corresponding to the ClickHouse type typ
// stored in the information_schema.columns column.
// The boolean return parameter reports whether the column type is nullable.
// It returns an invalid type if typ is not supported.
func columnType(typ string) (types.Type, bool) {
	if !utf8.ValidString(typ) {
		return types.Type{}, false
	}
	t, nullable, _ := parseType(typ, true)
	return t, nullable
}

// The boolean return parameter reports whether the column type is nullable.
func parseType(s string, allowNullable bool) (types.Type, bool, string) {
	s = strings.TrimLeft(s, " ")
	var i int
	for ; i < len(s); i++ {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9') {
			break
		}
	}
	if i == len(s) || s[i] != '(' {
		var t types.Type
		switch s[:i] {
		case "UInt8":
			t = types.Int(8).Unsigned()
		case "UInt16":
			t = types.Int(16).Unsigned()
		case "UInt32":
			t = types.Int(32).Unsigned()
		case "UInt64":
			t = types.Int(64).Unsigned()
		case "Int8":
			t = types.Int(8)
		case "Int16":
			t = types.Int(16)
		case "Int32":
			t = types.Int(32)
		case "Int64":
			t = types.Int(64)
		case "Float32":
			t = types.Float(32)
		case "Float64":
			t = types.Float(64)
		case "Bool":
			t = types.Boolean()
		case "String":
			t = types.String()
		case "UUID":
			t = types.UUID()
		case "Date", "Date32":
			t = types.Date()
		case "DateTime":
			t = types.DateTime()
		case "IPv4", "IPv6":
			t = types.IP()
		default:
			return types.Type{}, false, ""
		}
		return t, false, s[i:]
	}
	switch s[:i] {
	case "Decimal":
		precision, s, _ := parseUint(s[i+1:])
		if precision == 0 || precision > types.MaxDecimalPrecision {
			return types.Type{}, false, ""
		}
		s, ok := trimComma(s)
		if !ok {
			return types.Type{}, false, ""
		}
		scale, s, ok := parseUint(s)
		if !ok || scale > precision || scale > types.MaxDecimalScale || s == "" {
			return types.Type{}, false, ""
		}
		return types.Decimal(precision, scale), false, s[1:]
	case "FixedString":
		n, s, _ := parseUint(s[i+1:])
		if n == 0 || s == "" {
			return types.Type{}, false, ""
		}
		return types.String().WithMaxBytes(n), false, s[1:]
	case "DateTime":
		_, s, ok := parseString(s[i+1:])
		if !ok || s == "" {
			return types.Type{}, false, ""
		}
		return types.DateTime(), false, s[1:]
	case "DateTime64":
		n, s, ok := parseUint(s[i+1:])
		if !ok || n > 9 {
			return types.Type{}, false, ""
		}
		// Skip the timezone if present.
		if s, ok = trimComma(s); ok {
			_, s, ok = parseString(s)
			if !ok {
				return types.Type{}, false, ""
			}
		}
		if s == "" {
			return types.Type{}, false, ""
		}
		return types.DateTime(), false, s[1:]
	case "Enum8", "Enum16":
		s = s[i+1:]
		var values []string
		var item string
		for {
			var ok bool
			item, s, ok = parseString(s)
			if !ok {
				return types.Type{}, false, ""
			}
			if s, ok = trimEqual(s); ok {
				_, s, ok = parseInt(s)
				if !ok {
					return types.Type{}, false, ""
				}
			}
			values = append(values, item)
			if s, ok = trimComma(s); !ok {
				break
			}
		}
		if s == "" {
			return types.Type{}, false, ""
		}
		return types.String().WithValues(values...), false, s[1:]
	case "LowCardinality":
		t, _, s := parseType(s[i+1:], false)
		if s == "" {
			return types.Type{}, false, ""
		}
		return t, false, s[1:]
	case "Array":
		t, _, s := parseType(s[i+1:], false)
		if !t.Valid() || s == "" {
			return types.Type{}, false, ""
		}
		return types.Array(t), false, s[1:]
	case "Nullable":
		if !allowNullable {
			return types.Type{}, false, ""
		}
		t, _, s := parseType(s[i+1:], false)
		if !t.Valid() || s == "" {
			return types.Type{}, false, ""
		}
		return t, true, s[1:]
	case "Map":
		key, _, s := parseType(s[i+1:], false)
		s, comma := trimComma(s)
		if !key.Valid() || key.Kind() != types.StringKind || !comma {
			return types.Type{}, false, ""
		}
		value, _, s := parseType(s, false)
		if !value.Valid() || s == "" {
			return types.Type{}, false, ""
		}
		return types.Map(value), false, s[1:]
	}
	return types.Type{}, false, ""
}

// trimComma returns s without leading spaces (' ') followed by a comma if
// present. The returned boolean reports whether the comma is present.
func trimComma(s string) (string, bool) {
	p := strings.TrimLeft(s, " ")
	if len(p) == 0 || p[0] != ',' {
		return s, false
	}
	return p[1:], true
}

// trimEqual returns s without leading spaces (' ') followed by an equal
// character ('=') if present. The returned boolean reports whether the equal
// character is present.
func trimEqual(s string) (string, bool) {
	p := strings.TrimLeft(s, " ")
	if len(p) == 0 || p[0] != '=' {
		return s, false
	}
	return p[1:], true
}

// parseInt parses s, which begins with a signed integer, and returns the
// integer, the remaining part of s, and true. If s does not begin with a
// number, it returns "", s and false.
func parseInt(s string) (int, string, bool) {
	p := strings.TrimLeft(s, " ")
	var neg bool
	if len(p) > 0 && p[0] == '-' {
		neg = true
		p = p[1:]
	}
	var n, i int
	for ; i < len(p); i++ {
		c := p[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	if i == 0 {
		return 0, s, false
	}
	if neg {
		n = -n
	}
	return n, p[i:], true
}

// parseUint parses s, which begins with an unsigned integer, and returns the
// integer, the remaining part of s, and true. If s does not begin with a
// number, it returns "", s and false.
func parseUint(s string) (int, string, bool) {
	p := strings.TrimLeft(s, " ")
	var n, i int
	for ; i < len(p); i++ {
		c := p[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	if i == 0 {
		return 0, s, false
	}
	return n, p[i:], true
}

// parseString parses s, which begin with a ClickHouse literal string, and
// returns the string content and the remaining part of s. If s does not begin
// with a literal string, it returns an empty string and s.
func parseString(s string) (string, string, bool) {
	p := strings.TrimLeft(s, " ")
	if len(p) == 0 || p[0] != '\'' {
		return "", s, false
	}
	p = p[1:]
	var b bytes.Buffer
	for {
		i := strings.IndexAny(p, `'\`)
		if i == -1 {
			return "", s, false
		}
		b.WriteString(p[:i])
		if p[i] == '\'' {
			if i == len(p)-1 || p[i+1] != '\'' {
				return b.String(), p[i+1:], true
			}
			b.WriteByte('\'')
			p = p[i+2:]
			continue
		}
		if i+2 >= len(p) {
			return "", s, false
		}
		var r rune
		switch p[i+1] {
		case 'b':
			r = '\b'
		case 'f':
			r = '\f'
		case 'r':
			r = '\r'
		case 'n':
			r = '\n'
		case 't':
			r = '\t'
		case '0':
			r = 0
		case 'a':
			r = '\a'
		case 'v':
			r = '\v'
		case 'x':
			if i+4 >= len(p) {
				return "", s, false
			}
			r = 16*unhex(p[i+2]) + unhex(p[i+3])
			if r >= utf8.RuneSelf {
				return "", s, false
			}
			i += 2
		default:
			r = rune(p[i+1])
		}
		b.WriteRune(r)
		p = p[i+2:]
	}
}

func unhex(c byte) rune {
	switch {
	case '0' <= c && c <= '9':
		return rune(c - '0')
	case 'a' <= c && c <= 'f':
		return rune(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return rune(c - 'A' + 10)
	}
	return 0
}
