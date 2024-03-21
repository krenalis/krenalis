//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chichi/types"

	"github.com/shopspring/decimal"
)

// quoteTable quotes a table name.
func quoteTable(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// quoteString quotes s as a string and writes it into b.
//
// See the documentation at
// https://www.postgresql.org/docs/16/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
// (for the escaping of single quotes) and at
// https://www.postgresql.org/docs/13/datatype-character.html (for the character
// with code 0).
//
// NOTE: keep this function in sync with the one within the data warehouse
// driver of PostgreSQL.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexByte(s, '\'')
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		b.WriteString("''")
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
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
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		case types.DateKind:
			b.WriteString(v.Format("2006-01-02"))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999"))
		}
	case bool:
		if v {
			b.WriteString("TRUE")
		}
		b.WriteString("FALSE")
	case json.RawMessage:
		quoteString(b, string(v))
	default:
		panic(fmt.Errorf("unsupported value type '%T'", v))
	}
}
