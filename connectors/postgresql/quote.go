//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/types"
)

// quoteIdent quotes the identifier name.
func quoteIdent(name string) string {
	name = strings.ReplaceAll(name, `"`, `""`)
	return `"` + name + `"`
}

var jsonZeroByte = []byte(`\u0000`)

// quoteJSON quotes s, containing JSON code, as a string and writes it into b.
// Zero bytes (\u0000) in s are removed to handle invalid Unicode sequences, as
// PostgreSQL's jsonb type and certain JSON functions do not support \u0000.
//
// See the documentation at
// https://www.postgresql.org/docs/17/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
// (for the escaping of single quotes) and at
// https://www.postgresql.org/docs/current/datatype-json.html
func quoteJSON(b *strings.Builder, s []byte) {
	b.WriteByte('\'')
	for len(s) > 0 {
		p := bytes.IndexByte(s, '\'')
		if p == -1 {
			p = len(s)
		}
		// Check if the zero byte is present, that is, the sequence \x0000.
		for {
			z := bytes.Index(s[:p], jsonZeroByte)
			if z == -1 {
				break
			}
			// Check if it is preceded by an even number or zero of backslashes.
			even := true
			for i := z - 1; i >= 0 && s[i] == '\\'; i-- {
				even = !even
			}
			if !even {
				z += 6
			}
			b.Write(s[:z])
			if even {
				z += 6
			}
			s = s[z:]
			p -= z
		}
		b.Write(s[:p])
		if p == len(s) {
			break
		}
		b.WriteString("''")
		s = s[p+1:]
	}
	b.WriteByte('\'')
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
// NOTE: keep this function in sync with the one within the data warehouse
// driver of PostgreSQL.
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

// quoteValue quotes value and writes it into b.
func quoteValue(b *strings.Builder, v any, t types.Type) {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case string:
		if t.Kind() == types.TextKind {
			quoteString(b, v)
		} else {
			b.WriteByte('\'')
			b.WriteString(v)
			b.WriteByte('\'')
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, t.BitSize()))
	case decimal.Decimal:
		b.WriteString(v.String())
	case time.Time:
		b.WriteByte('\'')
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		case types.DateKind:
			b.WriteString(v.Format("2006-01-02"))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999"))
		}
		b.WriteByte('\'')
	case bool:
		if v {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	case json.Value:
		quoteJSON(b, v)
	default:
		panic(fmt.Errorf("unsupported value type '%T'", v))
	}
}
