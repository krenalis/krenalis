//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package mapexp

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

var (
	errNoTerminatedString = errors.New("string is not terminated")
	errNoTerminatedArgs   = errors.New("arguments are not terminated")
	errUnexpectedPeriod   = errors.New("unexpected period in path")
	errUnterminatedPath   = errors.New("path is not terminated")
	errZeroByteInString   = errors.New("character 0x00 is not allowed in strings")
	errInvalidNumber      = errors.New("number syntax is not valid")
)

// parseExpression parses an expression from the provided source string and
// returns the parsed expression along with the remaining unparsed source.
// If no expression is found, it returns nil. Leading and trailing spaces are
// trimmed, except when they occur within a string.
func parseExpression(src string, schema types.Type) ([]part, string, error) {
	var expression []part
	var err error
	var dot bool
Expression:
	for src != "" {
		var part part
		hasText := false
	Expr:
		for src != "" {
			switch c := src[0]; c {
			case ' ', '\t', '\n', '\r':
				src = src[1:]
			case '\'', '"':
				var s string
				s, src, err = parseString(src)
				if err != nil {
					return nil, "", err
				}
				part.text += s
				hasText = true
			case '.':
				src = src[1:]
				if len(src) == 0 {
					return nil, "", io.ErrUnexpectedEOF
				}
				if c := src[0]; !('0' <= c && c <= '9' || 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z') {
					return nil, "", errors.New("unexpected period")
				}
				dot = true
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
				var n decimal.Decimal
				n, src, err = parseNumber(src)
				if err != nil {
					return nil, "", err
				}
				if !hasText && len(expression) == 0 {
					part.value = n
					part.typ = types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)
					break Expr
				}
				part.text += n.String()
			default:
				if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z') {
					if part.text != "" {
						expression = append(expression, part)
					}
					break Expression
				}
				if !dot {
					var v any
					var t types.Type
					v, t, src = parsePredeclaredIdentifier(src)
					if t.Valid() {
						if !hasText && len(expression) == 0 {
							part.value = v
							part.typ = t
							break Expr
						}
						switch v {
						case true:
							part.text += "true"
						case false:
							part.text += "false"
						}
						continue Expr
					}
				}
				dot = false
				// Handle the Special case: "" a.b
				if hasText && part.text == "" {
					expression = append(expression, part)
					hasText = false
				}
				part.path, src, err = parsePath(src)
				if err != nil {
					return nil, "", err
				}
				src = skipSpaces(src)
				if len(src) > 0 && src[0] == '(' {
					if len(part.path) > 1 {
						return nil, "", errors.New("function name is not valid")
					}
					switch name := part.path[0]; name {
					case "coalesce":
					default:
						return nil, "", fmt.Errorf("function %q does not exist", name)
					}
					part.args, src, err = parseArgs(src, schema)
					if err != nil {
						return nil, "", err
					}
				} else {
					p, err := schema.PropertyByPath(part.path)
					if err != nil {
						return nil, "", err
					}
					if hasText || len(expression) > 0 {
						if pt := p.Type.PhysicalType(); !convertibleTo(pt, types.PtText) {
							return nil, "", fmt.Errorf("cannot convert %q of type %s to Text", part.path, pt)
						}
					}
					part.typ = p.Type
				}
				break Expr
			}
		}
		expression = append(expression, part)
	}

	return expression, src, nil
}

// parsePredeclaredIdentifier parses the predeclared identifiers true, false,
// and null from the given source and returns the parsed value along with the
// remaining unparsed source. If no predeclared identifier is parsed, it returns
// nil and src.
func parsePredeclaredIdentifier(src string) (any, types.Type, string) {
	if n := len("true"); strings.HasPrefix(src, "true") && (len(src) == n || !isPathByte(src[n])) {
		return true, types.Boolean(), src[n:]
	}
	if n := len("false"); strings.HasPrefix(src, "false") && (len(src) == n || !isPathByte(src[n])) {
		return false, types.Boolean(), src[n:]
	}
	if n := len("null"); strings.HasPrefix(src, "null") && (len(src) == n || !isPathByte(src[n])) {
		return nil, types.JSON(), src[n:]
	}
	return nil, types.Type{}, src
}

// isPathByte reports whether c is 'a'-'z', 'A-Z', '0'-'9', or '_'.
func isPathByte(c byte) bool {
	return 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// skipSpaces skips the leading spaces in src and returns the remaining string.
func skipSpaces(src string) string {
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case ' ', '\t', '\n', '\r':
		default:
			return src[i:]
		}
	}
	return ""
}

// parseString parses a string and returns the parsed string and the remaining
// unparsed source. It expects that src starts with ' or ".
func parseString(src string) (string, string, error) {
	quote := src[0]
	// First the common case: string without escape sequences.
	t := strings.IndexByte(src[1:], quote)
	if t == -1 {
		return "", "", errNoTerminatedString
	}
	src = src[1:]
	p := strings.IndexByte(src[:t], '\\')
	if p == -1 {
		if strings.IndexByte(src[:t], '\x00') != -1 {
			return "", "", errZeroByteInString
		}
		return src[:t], src[t+1:], nil
	}
	var b strings.Builder
LOOP:
	for {
		switch src[p] {
		case '\\':
			if p+1 == len(src) {
				return "", "", errNoTerminatedString
			}
			if p > 0 {
				b.WriteString(src[:p])
			}
			p, src = 0, src[p+1:]
			switch c := src[0]; c {
			case 'u', 'U':
				var n = 4
				if c == 'U' {
					n = 8
				}
				if p+n >= len(src) {
					return "", "", errNoTerminatedString
				}
				var r rune
				for i := 0; i < n; i++ {
					r = r * 16
					c = src[p+1+i]
					switch {
					case '0' <= c && c <= '9':
						r += rune(c - '0')
					case 'a' <= c && c <= 'f':
						r += rune(c - 'a' + 10)
					case 'A' <= c && c <= 'F':
						r += rune(c - 'A' + 10)
					default:
						return "", "", fmt.Errorf("hexadecimal escape has an invalid character %q", c)
					}
				}
				if r == 0x00 {
					return "", "", errZeroByteInString
				}
				if 0xD800 <= r && r < 0xE000 || r > '\U0010FFFF' {
					return "", "", fmt.Errorf("U+%X is not valid Unicode code point", r)
				}
				b.WriteRune(r)
				src = src[2+n:]
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'', '"':
				switch c {
				case 'a':
					c = '\a'
				case 'b':
					c = '\b'
				case 'f':
					c = '\f'
				case 'n':
					c = '\n'
				case 'r':
					c = '\r'
				case 't':
					c = '\t'
				case 'v':
					c = '\v'
				}
				b.WriteByte(c)
				src = src[1:]
			}
		case '\x00':
			return "", "", errZeroByteInString
		case quote:
			break LOOP
		default:
			p++
		}
		if p == len(src) {
			return "", "", errNoTerminatedString
		}
	}
	if p > 0 {
		b.WriteString(src[:p])
	}
	return b.String(), src[p+1:], nil
}

// parseNumber parses a number and returns the parsed number and the remaining
// unparsed source. It expects that src starts with '0'-'9', '-', or '.'.
func parseNumber(src string) (decimal.Decimal, string, error) {
	var i int
Number:
	for i < len(src) {
		switch src[i] {
		case '-':
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		case '.':
		case 'e', 'E':
		case ' ', '\t', '\n', '\r', '\'', '"', ',', ')':
			break Number
		default:
			return decimal.Decimal{}, "", errInvalidNumber
		}
		i++
	}
	n, err := decimal.NewFromString(src[:i])
	if err != nil {
		return decimal.Decimal{}, "", errInvalidNumber
	}
	return n, src[i:], nil
}

// parsePath parses a path and returns the parsed path and the remaining
// unparsed source. It expects that src starts with 'a'-'z', 'A'-'Z', or '_'.
func parsePath(src string) (types.Path, string, error) {
	path := make(types.Path, 0, 1)
	s := 0
	i := 1
	for ; i < len(src); i++ {
		c := src[i]
		if 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			continue
		}
		if s == i {
			return nil, "", errUnexpectedPeriod
		}
		path = append(path, src[s:i])
		s = i + 1
		if c != '.' {
			break
		}
	}
	if i == len(src) && src[i-1] == '.' {
		return nil, "", errUnterminatedPath
	}
	if s < i {
		path = append(path, src[s:i])
	}
	return path, src[i:], nil
}

// parseArgs parses call arguments and returns the parsed arguments along with
// the remaining unparsed source. It expects the source to start with a '('.
func parseArgs(src string, schema types.Type) ([][]part, string, error) {
	args := make([][]part, 0)
	var err error
	src = src[1:]
	for {
		var arg []part
		arg, src, err = parseExpression(src, schema)
		if err != nil {
			return nil, "", err
		}
		if src == "" {
			return nil, "", errNoTerminatedArgs
		}
		if arg == nil {
			if len(args) > 0 || src[0] == ',' {
				return nil, "", errors.New("missing function call argument")
			}
			if len(args) == 0 && src[0] != ')' {
				return nil, "", fmt.Errorf("unexpected %q, expecting function call argument", src[0])
			}
		} else {
			args = append(args, arg)
		}
		if src[0] == ')' {
			break
		}
		if src[0] != ',' {
			return nil, "", fmt.Errorf("unexpected %q, expecting ','", src[0])
		}
		src = src[1:]
	}
	return args, src[1:], nil
}
