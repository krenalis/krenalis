// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

// Keep this file in sync between the warehouse and the connector, except for
// the "warehouses" and "connectors" imports and the package path prefix in
// error messages.

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/snowflakedb/gosnowflake/v2"
)

// merge performs a table merge operation.
func merge(ctx context.Context, conn *sql.Conn, table connectors.Table, rows [][]any, deleted []any) error {

	quotedColumn := make(map[string]string, len(table.Columns))
	for _, column := range table.Columns {
		quotedColumn[column.Name] = quoteIdent(column.Name)
	}

	// Generate a unique name for the temporary table.
	tempTableName := "TEMP_TABLE_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Prepare the "create temporary table" statement.
	var b strings.Builder
	b.WriteString(`CREATE TEMPORARY TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range table.Columns {
		b.WriteString(quotedColumn[c.Name])
		b.WriteByte(',')
	}
	b.WriteString(` FALSE AS "$PURGE" FROM `)
	b.WriteString(quoteIdent(table.Name))
	b.WriteString(" LIMIT 0")
	create := b.String()

	// Create the 'merge' statement.
	b.Reset()
	b.WriteString(`MERGE INTO `)
	b.WriteString(quoteIdent(table.Name))
	b.WriteString(" \"D\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" \"S\"\nON ")
	for i, key := range table.Keys {
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(`"D".`)
		b.WriteString(quotedColumn[key])
		b.WriteString(` = "S".`)
		b.WriteString(quotedColumn[key])
	}
	if len(rows) > 0 {
		b.WriteString("\nWHEN MATCHED AND \"S\".\"$PURGE\" IS NULL THEN\n  UPDATE SET ")
		i := 0
		for _, c := range table.Columns {
			if slices.Contains(table.Keys, c.Name) {
				continue
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quotedColumn[c.Name])
			b.WriteString(` = "S".`)
			b.WriteString(quotedColumn[c.Name])
			i++
		}
		if i == 0 {
			return errors.New("snowflake.Merge: there must be at least one column in 'columns' apart from the keys")
		}
		b.WriteString("\nWHEN NOT MATCHED AND \"S\".\"$PURGE\" IS NULL THEN\n  INSERT (")
		for i, c := range table.Columns {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quotedColumn[c.Name])
		}
		b.WriteString(")\n  VALUES (")
		for i, c := range table.Columns {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`"S".`)
			b.WriteString(quotedColumn[c.Name])
		}
		b.WriteString(`)`)
	}
	if len(deleted) > 0 {
		b.WriteString("\nWHEN MATCHED THEN\n  DELETE")
	}
	merge := b.String()

	// Serialize the rows in CSV format.
	var rowsCSV io.Reader
	if len(rows) > 0 {
		var err error
		rowsCSV, err = serializeRowsToCSV(table.Columns, rows, false)
		if err != nil {
			return err
		}
	}

	// Serialize the deleted rows in CSV format.
	var deletedCSV io.Reader
	if len(deleted) > 0 {
		n := len(table.Keys)
		rows := make([][]any, 0, len(deleted)/n)
		for i := 0; i < len(deleted); i += n {
			rows = append(rows, deleted[i:i+n])
		}
		keys := make([]connectors.Column, len(table.Keys))
		for i, key := range table.Keys {
			for _, c := range table.Columns {
				if c.Name == key {
					keys[i] = c
					break
				}
			}
		}
		var err error
		deletedCSV, err = serializeRowsToCSV(keys, rows, true)
		if err != nil {
			return err
		}
	}

	// Create the temporary table.
	_, err := conn.ExecContext(ctx, create)
	if err != nil {
		return err
	}

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		// Put the rows into the temporary table's stage.
		err = conn.Raw(func(driverConn any) error {
			sc, ok := driverConn.(gosnowflake.SnowflakeConnection)
			if !ok {
				return fmt.Errorf("snowflake: unexpected driver connection type %T", driverConn)
			}
			return sc.UploadStream(ctx, `@%"`+tempTableName+`"`, "rows.csv", rowsCSV)
		})
		if err != nil {
			return err
		}
		// Copy the rows from the stage into the temporary table.
		b.Reset()
		b.WriteString("COPY INTO \"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFROM @%\"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, b.String())
		if err != nil {
			return err
		}
	}

	// Copy the deleted rows into the temporary table.
	if len(deleted) > 0 {
		// Put the deleted rows into the temporary table's stage.
		err = conn.Raw(func(driverConn any) error {
			sc, ok := driverConn.(gosnowflake.SnowflakeConnection)
			if !ok {
				return fmt.Errorf("snowflake: unexpected driver connection type %T", driverConn)
			}
			return sc.UploadStream(ctx, `@%"`+tempTableName+`"`, "rows.csv", deletedCSV)
		})
		if err != nil {
			return err
		}
		// Copy the deleted rows from the stage into the temporary table.
		b.Reset()
		b.WriteString("COPY INTO \"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFROM @%\"")
		b.WriteString(tempTableName)
		b.WriteString("\"\nFILE_FORMAT = (TYPE=CSV PARSE_HEADER=TRUE FIELD_OPTIONALLY_ENCLOSED_BY='0x27' ESCAPE_UNENCLOSED_FIELD=NONE EMPTY_FIELD_AS_NULL=TRUE NULL_IF=())\n" +
			"FILES = ('rows.csv.gz')\n" +
			"MATCH_BY_COLUMN_NAME = CASE_SENSITIVE\n" +
			"OVERWRITE = TRUE\n" +
			"ON_ERROR = ABORT_STATEMENT")
		_, err = conn.ExecContext(ctx, b.String())
		if err != nil {
			return err
		}
	}

	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.ExecContext(ctx, merge)
	if err != nil {
		return err
	}

	return nil
}

// plainJSONNumber expands a valid scientific JSON number only when its exact
// value fits NUMBER's maximum precision 38 and scale 37. A nil result leaves
// the original token unchanged; no floating-point conversion is performed.
func plainJSONNumber(text string) []byte {

	e := strings.IndexAny(text, "eE")
	if e < 0 {
		return nil
	}
	mantissa := text[:e]
	negative := mantissa[0] == '-'
	if negative {
		mantissa = mantissa[1:]
	}
	first, last, digits, leading, trailing := -1, -1, 0, 0, 0
	for i := range len(mantissa) {
		if mantissa[i] == '.' {
			continue
		}
		digits++
		if mantissa[i] == '0' {
			trailing++
			if first < 0 {
				leading++
			}
		} else {
			if first < 0 {
				first = i
			}
			last, trailing = i, 0
		}
	}
	if first < 0 {
		if negative {
			return []byte("-0")
		}
		return []byte("0")
	}
	significant := digits - leading - trailing
	if significant > 38 {
		return nil
	}
	fraction := 0
	if point := strings.IndexByte(mantissa, '.'); point >= 0 {
		fraction = len(mantissa) - point - 1
	}
	exponent, err := strconv.ParseInt(text[e+1:], 10, 64)
	if err != nil {
		return nil
	}
	// Bound the exponent before adding the mantissa's offset, including cases
	// where a long mantissa cancels a large exponent.
	offset := int64(trailing - fraction)
	if exponent < -37-offset || exponent > 38-offset {
		return nil
	}
	shift := int(exponent + offset)
	scale := max(-shift, 0)
	integer := significant + shift
	if scale > 37 || max(integer, 0)+scale > 38 {
		return nil
	}

	result := make([]byte, 0, 40)
	if negative {
		result = append(result, '-')
	}
	if integer <= 0 {
		result = append(result, '0', '.')
		for range -integer {
			result = append(result, '0')
		}
	}
	position := 0
	for i := first; i <= last; i++ {
		if mantissa[i] == '.' {
			continue
		}
		if position == integer && integer > 0 {
			result = append(result, '.')
		}
		result = append(result, mantissa[i])
		position++
	}
	for position < integer {
		result = append(result, '0')
		position++
	}

	return result
}

// plainJSONNumbers preserves the document's bytes except for scientific
// number tokens that can be stored exactly as INTEGER/DECIMAL in Snowflake.
// Values outside that domain retain the existing loading behavior.
func plainJSONNumbers(value json.Value) (json.Value, error) {

	// A numeric exponent always follows a digit. Strings can cause harmless
	// false positives, but documents without such a pair need no decoding.
	hasExponent := false
	for i := 1; i < len(value); i++ {
		if (value[i] == 'e' || value[i] == 'E') && value[i-1] >= '0' && value[i-1] <= '9' {
			hasExponent = true
			break
		}
	}
	if !hasExponent {
		return value, nil
	}

	decoder := json.NewDecoder(bytes.NewBuffer(value))
	var result json.Value
	copied := 0
	for {
		token, err := decoder.ReadToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("connectors/snowflake: invalid JSON value for CSV serialization")
		}
		if token.Kind() != json.Number {
			continue
		}
		text := token.String()
		plain := plainJSONNumber(text)
		if plain == nil {
			continue
		}
		end := int(decoder.ByteOffset())
		result = append(result, value[copied:end-len(text)]...)
		result = append(result, plain...)
		copied = end
	}
	if result == nil {
		return value, nil
	}
	result = append(result, value[copied:]...)

	return result, nil
}

// serializeRowsToCSV serializes rows as CSV, using columns as header, and
// returns it as an io.Reader. It also appends a boolean column called $PURGE
// with the value of the 'deleted' argument as value for each row.
func serializeRowsToCSV(columns []connectors.Column, rows [][]any, deleted bool) (io.Reader, error) {
	var err error
	var b bytes.Buffer
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(csvColumnName(c.Name))
	}
	b.WriteString(",$PURGE\n")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, v := range row {
			if j > 0 {
				b.WriteByte(',')
			}
			err = serializeValueToCSV(&b, columns[j].Type, v)
			if err != nil {
				return nil, err
			}
		}
		// Add the value for the column $PURGE.
		if deleted {
			b.WriteString(",true")
		} else {
			b.WriteString(",")
		}
	}
	return &b, nil
}

// quoteCSVBytes is like quoteCSVString but gets a []byte value as argument.
func quoteCSVBytes(b *bytes.Buffer, s []byte) {
	b.WriteByte('\'')
	for len(s) > 0 {
		i := bytes.IndexByte(s, '\'')
		if i == -1 {
			b.Write(s)
			break
		}
		b.Write(s[:i+1])
		b.WriteByte('\'')
		s = s[i+1:]
	}
	b.WriteByte('\'')
}

// quoteCSVString quotes the string s for use in a CSV file and writes it to b.
// A string must be quoted if it is empty, or contains characters "'", ","
// or "\n".
func quoteCSVString(b *bytes.Buffer, s string) {
	b.WriteByte('\'')
	for len(s) > 0 {
		i := strings.IndexByte(s, '\'')
		if i == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i+1])
		b.WriteByte('\'')
		s = s[i+1:]
	}
	b.WriteByte('\'')
}

func serializeValueToCSV(b *bytes.Buffer, t types.Type, v any) error {
	switch v := v.(type) {
	case nil:
	case string:
		if v == "" || strings.ContainsAny(v, "',\n") {
			quoteCSVString(b, v)
		} else {
			b.WriteString(v)
		}
	case bool:
		if v {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case decimal.Decimal:
		_, _ = v.WriteTo(b)
	case json.Value:
		value, err := plainJSONNumbers(v)
		if err != nil {
			return err
		}
		quoteCSVBytes(b, value)
	default:
		switch k := t.Kind(); k {
		case types.DateTimeKind:
			b.WriteString(v.(time.Time).Format("2006-01-02 15:04:05.999999999"))
		case types.DateKind:
			b.WriteString(v.(time.Time).Format("2006-01-02"))
		case types.TimeKind:
			b.WriteString(v.(time.Time).Format("15:04:05.999999999"))
		case types.ArrayKind:
			value, err := types.Marshal(v.([]any), t)
			if err != nil {
				return err
			}
			value, err = plainJSONNumbers(value)
			if err != nil {
				return err
			}
			quoteCSVBytes(b, value)
		case types.MapKind:
			value, err := types.Marshal(v.(map[string]any), t)
			if err != nil {
				return err
			}
			value, err = plainJSONNumbers(value)
			if err != nil {
				return err
			}
			quoteCSVBytes(b, value)
		default:
			return fmt.Errorf("cannot serialize as Snowflake CSV: unsupported type %T for column type %s", v, k)
		}
	}
	return nil
}
