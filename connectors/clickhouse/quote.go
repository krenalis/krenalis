// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package clickhouse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

// quoteColumn quotes a column name.
func quoteColumn(name string) (string, error) {
	if !strings.Contains(name, "`") {
		return "`" + name + "`", nil
	}
	if !strings.Contains(name, `"`) {
		return `"` + name + `"`, nil
	}
	return "", errors.New("column name contains both '`' and '\"' characters")
}

// quoteTable quotes a table name.
func quoteTable(name string) (string, error) {
	for i := 0; i < len(name); i++ {
		c := name[i]
		if 'a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9' {
			continue
		}
		return "", errors.New("table name must match the regex ^[a-zA-Z_][0-9a-zA-Z_]*$")
	}
	return "`" + name + "`", nil
}

// quoteValue quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexAny(s, "\\'")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		b.WriteByte('\\')
		b.WriteByte(s[p])
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}

// quoteValue quotes v and writes it into b.
func quoteValue(b *strings.Builder, v any, t types.Type) {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case string:
		if t.Kind() == types.StringKind {
			quoteString(b, v)
		} else {
			b.WriteByte('\'')
			b.WriteString(v)
			b.WriteByte('\'')
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
		b.WriteByte('\'')
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.Format("2006-01-02 15:04:05.999999999"))
		case types.DateKind:
			b.WriteString(v.Format("2006-01-02"))
		}
		b.WriteByte('\'')
	case bool:
		if v {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	case json.Value:
		quoteString(b, string(v))
	case []any:
		vt := t.Elem()
		b.WriteString("array(")
		for i, ev := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			quoteValue(b, ev, vt)
		}
		b.WriteByte(')')
	case map[string]any:
		vt := t.Elem()
		b.WriteByte('{')
		i := 0
		for k, ev := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			quoteString(b, k)
			b.WriteByte(':')
			quoteValue(b, ev, vt)
			i++
		}
		b.WriteByte('}')
	default:
		panic(fmt.Errorf("unsupported value type '%T'", v))
	}
}
