// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

// Kind represents a specific kind of JSON value or JSON token.
//
//   - 'n' ([Null]): null
//   - 'f' ([False]): false
//   - 't' ([True]): true
//   - '"' ([String]): string
//   - '0' ([Number]): number
//   - '{' ([Object]): begin object
//   - '}': end object
//   - '[' ([Array]): begin array
//   - ']': end array
type Kind byte

const (
	Invalid Kind = 0
	Null    Kind = 'n'
	True    Kind = 't'
	False   Kind = 'f'
	String  Kind = '"'
	Number  Kind = '0'
	Object  Kind = '{'
	Array   Kind = '['
)

// String returns the name of k.
// It returns "invalid" if is not a [Kind] or is [Invalid].
func (k Kind) String() string {
	switch k {
	case 'n':
		return "null"
	case 't':
		return "true"
	case 'f':
		return "false"
	case '"':
		return "string"
	case '0':
		return "number"
	case '{':
		return "object"
	case '}':
		return "object end"
	case '[':
		return "array"
	case ']':
		return "array end"
	}
	return "invalid"
}
