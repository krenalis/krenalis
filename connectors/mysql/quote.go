// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mysql

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
func quoteValue(b *strings.Builder, v any, t types.Type) error {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case string:
		b.WriteByte('"')
		if t.Kind() == types.StringKind {
			quoteString(b, v)
		} else {
			b.WriteString(v)
		}
		b.WriteByte('"')
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
	case json.Value:
		b.WriteByte('"')
		quoteString(b, string(v))
		b.WriteByte('"')
	// TODO(marco): SET can be implemented as an array(T), but the driver only returns the first element of the set.
	//case []any:
	//	b.WriteByte('"')
	//	for i, s := range v {
	//		s := s.(string)
	//		if strings.Contains(s, ",") {
	//			return errors.New("an array element contains commas")
	//		}
	//		if i > 0 {
	//			b.WriteByte(',')
	//		}
	//		quoteString(b, s)
	//	}
	//	b.WriteByte('"')
	default:
		panic(fmt.Errorf("unsupported value type '%T'", v))
	}
	return nil
}

// quoteString quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		return
	}
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
}
