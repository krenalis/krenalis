//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

var (
	errInvalidNumber      = errors.New("number syntax is not valid")
	errNoStringMapKey     = errors.New("map key is not a string")
	errNoTerminatedArgs   = errors.New("arguments are not terminated")
	errNoTerminatedString = errors.New("string is not terminated")
	errUnexpectedPeriod   = errors.New("unexpected period in path")
	errUnterminatedPath   = errors.New("path is not terminated")
	errZeroByteInString   = errors.New("character 0x00 is not allowed in strings")
)

// numArguments reports the number of arguments for each expression function.
// It is used to initialize the slice of arguments before parsing them.
var numArguments = map[string]int{
	"and":        2,
	"array":      2,
	"coalesce":   2,
	"eq":         2,
	"if":         3,
	"initcap":    1,
	"json_parse": 1,
	"len":        1,
	"lower":      1,
	"ltrim":      1,
	"ne":         2,
	"not":        1,
	"or":         2,
	"rtrim":      1,
	"substring":  3,
	"trim":       1,
	"upper":      1,
}

// path represents a property path or a function name.
type path struct {

	// Elements of the property path or function name.
	// If the path represents a function name, it consists only of the function name (no parameters).
	elements []string

	// If path represents a property path, decorators indicates, for each path element, its decorators:
	//   - If element i was decorated with an indexing (e.g., a["b"]), decorators[i].indexing() returns true.
	//   - If element i was decorated by '?', decorators[i].optional() returns true.
	// If path represents a function (and not a property path), decorators is nil.
	decorators []decorators
}

// slice returns a slice of p.
func (p path) slice(i, j int) path {
	if i == j {
		return path{}
	}
	return path{
		elements:   p.elements[i:j],
		decorators: p.decorators[i:j],
	}
}

// String returns p as a string.
func (p path) String() string {
	if p.elements == nil {
		return ""
	}
	s := p.elements[0]
	for i, name := range p.elements {
		if i == 0 {
			continue
		}
		dec := p.decorators[i]
		if dec.indexing() {
			s += "[" + strconv.Quote(name) + "]"
		} else {
			s += "." + name
		}
		if dec.optional() {
			s += "?"
		}
	}
	return s
}

// decorators represents the bit flags for the indexing ("x?") and optional
// ("[x]") decorators.
type decorators uint8

const (
	indexing decorators = 1 << iota // indexing represents the "x?" decorator
	optional                        // optional represents the "[x]" decorator
)

// indexing reports whether d has the indexing ("x?") decorator
func (d decorators) indexing() bool {
	return d&indexing != 0
}

// optional reports whether d has the optional ("[x]") decorator.
func (d decorators) optional() bool {
	return d&optional != 0
}

// part represents an expression part within an Expression. An expression part
// can take different forms:
//
//   - value             example: "foo"
//   - path              example: x
//   - path(args)        example: add(x, 5)
//   - value path        example: 5 a.b
//   - value path(args)  example: "foo" coalesce(a.b, c)
//
// For instance, the Expression `"foo" x " " true a.b` is parsed as `"foo" x`,
// `" true" a.b`.
type part struct {
	// Value. If there is a path, value, if present, can only be of type text.
	value any

	// Property path or function name.
	path path

	// Function call arguments.
	args [][]part

	// If there is a path, it represents the type of the property or the type of the function call.
	// Otherwise, it represents the type of the value. For some function calls, as coalesce, it is
	// the invalid type, indicating that the call can return different types.
	typ types.Type
}

// appendValue appends v to p.value, converting it to type text is necessary,
// and returns the appended value and its new type.
// multipart reports whether p is a part of a multipart expression.
func (p part) appendValue(v any, multipart bool) (any, types.Type) {
	// If a value is not already present, it sets it.
	if !multipart && p.typ.Kind() == types.InvalidKind {
		switch v := v.(type) {
		case nil:
			return nil, types.JSON()
		case string:
			return v, types.Text()
		case decimal.Decimal:
			i, err := decimalToInt(v)
			if err != nil {
				return v, types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)
			}
			return i, types.Int(32)
		case bool:
			return v, types.Boolean()
		}
		panic("unexpected value type")
	}
	// Convert the value to text.
	var s string
	switch v := v.(type) {
	case string:
		s = v
	case bool:
		s = strconv.FormatBool(v)
	case decimal.Decimal:
		s = v.String()
	}
	// Append the value.
	t := types.Text()
	switch p.typ.Kind() {
	case types.InvalidKind:
		return s, t
	case types.BooleanKind:
		return strconv.FormatBool(p.value.(bool)) + s, t
	case types.IntKind:
		return strconv.Itoa(p.value.(int)) + s, t
	case types.DecimalKind:
		return p.value.(decimal.Decimal).String() + s, t
	case types.TextKind:
		return p.value.(string) + s, t
	}
	panic("unexpected value type")
}

// isPathByte reports whether c is 'a'-'z', 'A-Z', '0'-'9', or '_'.
func isPathByte(c byte) bool {
	return 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// parse parses an expression from the provided source string and returns the
// parsed expression along with the remaining unparsed source. If no expression
// is found, it returns nil. Leading and trailing spaces are trimmed, except
// when they occur within a string.
func parse(src string) ([]part, string, error) {
	var expr []part
	var err error
	var dot bool
Expression:
	for src != "" {
		var p part
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
				p.value, p.typ = p.appendValue(s, len(expr) > 0)
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
					// For '-' followed by a path, return a clearer and more descriptive error message.
					// See issue https://github.com/meergo/meergo/issues/1344.
					if c == '-' && len(src) > 1 {
						if c = src[1]; 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
							if _, _, err := parsePath(src[1:]); err == nil {
								return nil, "", errors.New("the ‘-’ character is used for negation. Did you mean the ‘_’ character?")
							}
						}
					}
					return nil, "", err
				}
				p.value, p.typ = p.appendValue(n, len(expr) > 0)
			default:
				if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z') {
					if p.typ.Valid() {
						expr = append(expr, p)
					}
					break Expression
				}
				if !dot {
					if v, t, src2 := parsePredeclaredIdentifier(src); t.Valid() {
						p.value, p.typ = p.appendValue(v, len(expr) > 0)
						src = src2
						continue Expr
					}
				}
				dot = false
				// If there is a non-text value, convert it to text.
				switch p.typ.Kind() {
				case types.BooleanKind:
					p.value = strconv.FormatBool(p.value.(bool))
				case types.IntKind:
					p.value = strconv.Itoa(p.value.(int))
				case types.DecimalKind:
					p.value = p.value.(decimal.Decimal).String()
				case types.JSONKind:
					p.value = ""
				}
				// Parse the path.
				p.path, src, err = parsePath(src)
				if err != nil {
					return nil, "", err
				}
				src = skipSpaces(src)
				if len(src) == 0 || src[0] != '(' {
					break Expr
				}
				// Parse function call.
				src = src[1:]
				name := p.path.elements[0]
				n, ok := numArguments[name]
				if !ok || len(p.path.elements) > 1 {
					return nil, "", fmt.Errorf("function %q does not exist", p.path)
				}
				p.args = make([][]part, 0, n)
				for {
					src = skipSpaces(src)
					if src == "" {
						return nil, "", errNoTerminatedArgs
					}
					if src[0] == ')' {
						break
					}
					var arg []part
					arg, src, err = parse(src)
					if err != nil {
						return nil, "", err
					}
					if arg == nil {
						return nil, "", fmt.Errorf("expected argument, got %q", src[0])
					}
					src = skipSpaces(src)
					if src != "" && src[0] == ',' {
						src = skipSpaces(src[1:])
						if src != "" && src[0] == ')' {
							return nil, "", fmt.Errorf("expected argument, got ')'")
						}
					}
					p.args = append(p.args, arg)
				}
				src = src[1:]
				break Expr
			}
		}
		expr = append(expr, p)
	}
	return expr, src, nil
}

// parseNumber parses a number and returns the parsed number and the remaining
// unparsed source. It expects that src starts with '0'-'9', '-', or '.'.
// If an error occurs, it returns 0, src, errInvalidNumber.
func parseNumber(src string) (decimal.Decimal, string, error) {
	var i int
Number:
	for i < len(src) {
		switch c := src[i]; c {
		case '-', '+':
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		case '.':
		case 'e', 'E':
		default:
			if isPathByte(c) {
				return decimal.Decimal{}, src, errInvalidNumber
			}
			break Number
		}
		i++
	}
	n, err := decimal.Parse(src[:i], 0, 0)
	if err != nil {
		return decimal.Decimal{}, src, errInvalidNumber
	}
	return n, src[i:], nil
}

// parsePath parses a path and returns the parsed path, its decorators, and the
// remaining unparsed source.
// It expects that src starts with 'a'-'z', 'A'-'Z', or '_'.
func parsePath(src string) (path, string, error) {
	var err error
	p := path{
		elements:   make([]string, 0, 1),
		decorators: make([]decorators, 0, 1),
	}
	i := 1
	for i < len(src) {
		c := src[i]
		if 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			i++
			continue
		}
		if i == 0 {
			return path{}, "", errUnexpectedPeriod
		}
		p.elements = append(p.elements, src[:i])
		if c == '?' {
			p.decorators = append(p.decorators, optional)
			i++
		} else {
			p.decorators = append(p.decorators, 0)
		}
		src, i = src[i:], 0
		if len(src) == 0 {
			break
		}
		for src[0] == '[' {
			src = skipSpaces(src[1:])
			if src == "" {
				return path{}, "", errUnterminatedPath
			}
			if src[0] != '"' && src[0] != '\'' {
				return path{}, "", errNoStringMapKey
			}
			var key string
			key, src, err = parseString(src)
			if err != nil {
				return path{}, "", err
			}
			p.elements = append(p.elements, key)
			p.decorators = append(p.decorators, indexing)
			src = skipSpaces(src)
			if src == "" || src[0] != ']' {
				return path{}, "", errUnterminatedPath
			}
			src, i = src[1:], 0
			if src == "" {
				break
			}
			if src[0] == '?' {
				p.decorators[len(p.decorators)-1] = indexing | optional
				src = src[1:]
			}
			if src == "" {
				break
			}
		}
		if len(src) == 0 || src[0] != '.' {
			break
		}
		src = src[1:]
		if src == "" {
			return path{}, "", errUnterminatedPath
		}
	}
	if i > 0 {
		p.elements = append(p.elements, src[:i])
		p.decorators = append(p.decorators, 0)
		src = src[i:]
	}
	return p, src, nil
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
