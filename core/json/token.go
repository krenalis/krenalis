// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"strconv"
	"strings"

	"github.com/meergo/meergo/core/json/internal/json/jsontext"
)

type Token struct {
	tok jsontext.Token
}

// Bool reports whether t is the boolean value true.
func (t Token) Bool() bool {
	return t.tok.Kind() == 't'
}

// Float returns the floating-point value for a JSON number with the provided
// bit size. It returns an error if t is not a JSON number, is out of range, or
// bitSize is neither 32 nor 64.
func (t Token) Float(bitSize int) (float64, error) {
	return strconv.ParseFloat(t.tok.String(), bitSize)
}

// Int returns the integer value for a JSON number. It returns an error if t is
// not a valid JSON number, does not represent an integer, or is out of range.
// As a special case, an integer followed by ".0" is considered valid;
// for instance, "1" and "1.0" are both valid.
func (t Token) Int() (int, error) {
	n, err := strconv.ParseInt(strings.TrimSuffix(t.tok.String(), ".0"), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// Kind returns the kind of t.
func (t Token) Kind() Kind {
	return Kind(t.tok.Kind())
}

// String returns the unescaped string value for a JSON string, and the raw JSON
// representation for other JSON types.
//
// If the caller does not use the returned string in an escaping context
// (see https://blog.filippo.io/efficient-go-apis-with-the-inliner/),
// it does not allocate.
func (t Token) String() string {
	return t.tok.String()
}

// Uint returns the unsigned integer value for a JSON number. It returns an
// error if t is not a valid JSON number, does not represent an integer, or is
// out of range. As a special case, an integer followed by ".0" is considered
// valid; for instance, "1" and "1.0" are both valid.
func (t Token) Uint() (uint, error) {
	n, err := strconv.ParseUint(strings.TrimSuffix(t.tok.String(), ".0"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
