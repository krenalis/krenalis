// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
)

// quoteIdent quotes the identifier name.
func quoteIdent(name string) string {
	name = strings.ReplaceAll(name, `"`, `""`)
	return `"` + name + `"`
}

// quoteString quotes s as a string and writes it into b.
// Null bytes ('\x00') in s are removed.
//
// See the documentation at
// https://www.postgresql.org/docs/17/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
// (for the escaping of single quotes) and at
// https://www.postgresql.org/docs/17/datatype-character.html (for the character
// with code 0).
//
// NOTE: keep this function in sync with the one within the PostgreSQL
// connector.
func quoteString(b *strings.Builder, s string) {
	b.WriteByte('\'')
	for len(s) > 0 {
		p := strings.IndexAny(s, "'\x00")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		if s[p] == '\'' {
			b.WriteString("''")
		}
		s = s[p+1:]
	}
	b.WriteByte('\'')
}
