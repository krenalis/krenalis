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
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

// ParseTime parses a time formatted as "hh:mm:ss.nnnnnnnnn" and returns it as
// the time on January 1, 1970 UTC. The sub-second part can contain from 1 to 9
// digits or can be missing. The hour must be in range [0, 23], minute and
// second must be in range [0, 59], and any trailing characters are discarded.
// The boolean return value indicates whether the time was successfully parsed.
func ParseTime[bytes []byte | string](p bytes) (t time.Time, ok bool) {
	if len(p) < 8 {
		return
	}
	parse := func(n bytes) int {
		if n[0] < '0' || n[0] > '9' || n[1] < '0' || n[1] > '9' {
			return -1
		}
		return int(n[0]-'0')*10 + int(n[1]-'0')
	}
	h, m, s := parse(p[0:2]), parse(p[3:5]), parse(p[6:8])
	if h < 0 || h > 23 || p[2] != ':' || m < 0 || m > 59 || p[5] != ':' || s < 0 || s > 59 {
		return
	}
	p = p[8:]
	var ns int
	if len(p) > 0 && p[0] == '.' {
		p = p[1:]
		var i int
		for ; i < 9 && i < len(p) && '0' <= p[i] && p[i] <= '9'; i++ {
			ns = ns*10 + int(p[i]-'0')
		}
		if i == 0 {
			return
		}
		for ; i < 9; i++ {
			ns *= 10
		}
	}
	return time.Date(1970, 1, 1, h, m, s, ns, time.UTC), true
}

// ParseUUID parses s as a UUID in the standard form xxxx-xxxx-xxxx-xxxxxxxxxxxx
// and returns it in the canonical form without uppercase letters. The boolean
// return value reports whether s is a UUID in the standard form.
func ParseUUID(s string) (string, bool) {
	if len(s) != 36 {
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

// PropertiesToColumns returns the columns of properties of t.
func PropertiesToColumns(t types.Type) []meergo.Column {

	columns := make([]meergo.Column, 0, types.NumProperties(t))
	for _, p := range t.Properties() {
		if p.Type.Kind() == types.ObjectKind {
			for _, column := range PropertiesToColumns(p.Type) {
				column.Name = p.Name + "_" + column.Name
				columns = append(columns, column)
			}
			continue
		}
		columns = append(columns, meergo.Column{
			Name:     p.Name,
			Type:     p.Type,
			Nullable: p.Nullable,
		})
	}
	return columns
}

// TransformationFunctionName returns the name of the transformation function
// for an action in the specified language.
func TransformationFunctionName(action int) string {
	now := time.Now().UTC()
	return fmt.Sprintf("meergo_action%d_%s-%09d", action, now.Format("2006-01-02T15-04-05"), now.Nanosecond())
}

// UUIDFromBytes returns the UUID corresponding to the given byte slice
// (representing the 128 bit of the UUID, so it must have lenght 16) it in the
// canonical string form without uppercase letters. The boolean return value
// reports whether s represent an UUID or not.
func UUIDFromBytes(s []byte) (string, bool) {
	id, err := uuid.FromBytes(s)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

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
