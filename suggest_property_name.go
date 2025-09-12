//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package meergo

import (
	"strings"
	"unicode"

	"github.com/meergo/meergo/core/types"

	"golang.org/x/text/unicode/norm"
)

// SuggestPropertyName suggests a valid property name based on s. It returns the
// suggested property name and true if a suggestion is available; otherwise, it
// returns an empty string and false.
func SuggestPropertyName(s string) (string, bool) {
	if types.IsValidPropertyName(s) {
		return s, true
	}
	s = strings.TrimSpace(s)
	var underscore bool
	var b strings.Builder
	for i, r := range s {
		if 'a' <= r && r <= 'z' || r == '_' || 'A' <= r && r <= 'Z' || i > 0 && '0' <= r && r <= '9' {
			b.WriteRune(r)
			underscore = r == '_'
			continue
		}
		if unicode.IsLetter(r) {
			c := baseChar(r)
			if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
				return "", false
			}
			b.WriteRune(c)
			underscore = false
			continue
		}
		if !underscore {
			b.WriteRune('_')
			underscore = true
		}
		continue
	}
	if b.Len() == 0 {
		return "", false
	}
	s = b.String()
	if s == "_" {
		return "", false
	}
	return b.String(), true
}

// baseChar returns the base character for a given accented character.
func baseChar(r rune) rune {
	decomposed := norm.NFD.String(string(r))
	for _, dr := range decomposed {
		if unicode.IsLetter(dr) && !unicode.Is(unicode.Mn, dr) {
			return dr
		}
	}
	return r
}
