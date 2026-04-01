// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package util

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
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

// PropertiesToColumns returns the columns of properties.
func PropertiesToColumns(properties types.Properties) []warehouses.Column {
	columns := make([]warehouses.Column, 0, properties.Len())
	for _, p := range properties.All() {
		if p.Type.Kind() == types.ObjectKind {
			for _, column := range PropertiesToColumns(p.Type.Properties()) {
				column.Name = p.Name + "_" + column.Name
				columns = append(columns, column)
			}
			continue
		}
		columns = append(columns, warehouses.Column{
			Name:     p.Name,
			Type:     p.Type,
			Nullable: p.Nullable,
		})
	}
	return columns
}

// ValidateStringField validates a string field identified by the provided name.
// It returns an error if any of the following conditions are met:
//   - The string s is empty.
//   - The string s contains invalid UTF-8 runes.
//   - The string s contains a NUL byte.
//   - The string s exceeds maxLen runes.
func ValidateStringField(name, s string, maxLen int) error {
	if len(s) == 0 {
		return fmt.Errorf("%s is empty", name)
	}
	var count int
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return fmt.Errorf("%s contains invalid UTF-8 encoded characters", name)
		}
		if r == '\x00' {
			return fmt.Errorf("%s contains the NUL byte", name)
		}
		count++
		if count > maxLen {
			return fmt.Errorf("%s is longer than %d runes", name, maxLen)
		}
		i += size
	}
	return nil
}
