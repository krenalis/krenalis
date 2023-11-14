//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

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
		if t.PhysicalType() == types.PtText {
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
