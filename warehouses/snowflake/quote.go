// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"strings"
)

// quoteBytes quotes s as a string and writes it into b.
func quoteBytes(b *strings.Builder, s []byte) {
	if len(s) == 0 {
		b.WriteString(`''`)
		return
	}
	b.WriteByte('\'')
	for {
		p := bytes.IndexAny(s, "\x00'\b\f\n\r\t\\")
		if p == -1 {
			p = len(s)
		}
		b.Write(s[:p])
		if p == len(s) {
			break
		}
		b.WriteByte('\\')
		var c byte
		switch s[p] {
		case 0:
			c = '0'
		case '\'':
			c = '\''
		case '\b':
			c = 'b'
		case '\f':
			c = 'f'
		case '\n':
			c = 'n'
		case '\r':
			c = 'r'
		case '\t':
			c = 't'
		case '\\':
			c = '\\'
		}
		b.WriteByte(c)
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}

// quoteIdent quotes the identifier name.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(strings.ToUpper(name), `"`, `""`) + `"`
}

// quoteString quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString(`''`)
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexAny(s, "\x00'\b\f\n\r\t\\")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		b.WriteByte('\\')
		var c byte
		switch s[p] {
		case 0:
			c = '0'
		case '\'':
			c = '\''
		case '\b':
			c = 'b'
		case '\f':
			c = 'f'
		case '\n':
			c = 'n'
		case '\r':
			c = 'r'
		case '\t':
			c = 't'
		case '\\':
			c = '\\'
		}
		b.WriteByte(c)
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}
