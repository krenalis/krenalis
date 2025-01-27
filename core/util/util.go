//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package util

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValidateStringField validates a string field identified by the provided name.
// It returns an error if any of the following conditions are met:
//   - The string s is empty.
//   - The string s contains invalid UTF-8 runes.
//   - The string s contains a NUL byte.
//   - The string s exceeds maxLen runes.
func ValidateStringField(name string, s string, maxLen int) error {
	if s == "" {
		return fmt.Errorf("%s is empty", name)
	}
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s contains invalid UTF-8 encoded characters", name)
	}
	if strings.ContainsRune(s, '\x00') {
		return fmt.Errorf("%s contains the NUL byte", name)
	}
	if utf8.RuneCountInString(s) > maxLen {
		return fmt.Errorf("%s is longer than %s runes", name, strings.ReplaceAll(strconv.Itoa(maxLen), ".", ","))
	}
	return nil
}
