//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mysql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// quoteColumn quotes a column name.
func quoteColumn(name string) (string, error) {
	if strings.Contains(name, "`") {
		return "", errors.New("column name contains a backtick character")
	}
	return "`" + name + "`", nil
}

// quoteTable quotes a table name.
func quoteTable(name string) (string, error) {
	if strings.Contains(name, "`") {
		return "", errors.New("table name contains a backtick character")
	}
	return "`" + name + "`", nil
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
			b.WriteByte('"')
			b.WriteString(v)
			b.WriteByte('"')
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, t.BitSize()))
	case decimal.Decimal:
		b.WriteString(v.String())
	case time.Time:
		b.WriteByte('"')
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		case types.DateKind:
			b.WriteString(v.Format("2006-01-02"))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999"))
		}
		b.WriteByte('"')
	case bool:
		if v {
			b.WriteString("1")
		}
		b.WriteString("0")
	default:
		panic(fmt.Errorf("unsupported value type '%T'", v))
	}
}

// quoteString quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString(`""`)
		return
	}
	b.WriteByte('"')
	for {
		p := strings.IndexAny(s, "\x00'\"\b\n\r\t\032\\")
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
		case '"':
			c = '"'
		case '\b':
			c = 'b'
		case '\n':
			c = 'n'
		case '\r':
			c = 'r'
		case '\t':
			c = 't'
		case '\032':
			c = 'Z'
		case '\\':
			c = '\\'
		}
		b.WriteByte(c)
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('"')
}
