// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"strings"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// csvColumnName returns the column name to write in Snowflake CSV headers.
func csvColumnName(name string) string {
	return name
}

// quoteIdent quotes the identifier name.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// quoteTable quotes a table name.
func quoteTable(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
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

// quoteValue quotes v and writes it into b.
// It only supports the datetime, date, json, and text types, and for json only
// a string value.
func quoteValue(b *strings.Builder, v any, t types.Type) {
	switch t.Kind() {
	case types.StringKind:
		quoteString(b, v.(string))
	case types.DateTimeKind:
		b.WriteByte('\'')
		b.WriteString(v.(time.Time).Format("2006-01-02 15:04:05.000000000"))
		b.WriteByte('\'')
	case types.DateKind:
		b.WriteByte('\'')
		b.WriteString(v.(time.Time).Format("2006-01-02"))
		b.WriteByte('\'')
	case types.JSONKind:
		b.WriteString("TO_VARIANT(")
		quoteString(b, string(v.(json.Value)))
		b.WriteString(")")
	default:
		panic(fmt.Errorf("unsupported value type %s", t.Kind()))
	}
}
