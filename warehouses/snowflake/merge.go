// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

// Keep this file in sync between the warehouse and the connector, except for
// the "warehouses" and "connectors" import lines.

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

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/snowflakedb/gosnowflake"
)

// merge performs a table merge operation.
func merge(ctx context.Context, conn *sql.Conn, table warehouses.Table, rows [][]any, deleted []any) error {

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
		keys := make([]warehouses.Column, len(table.Keys))
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
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, rowsCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
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
		_, err = conn.ExecContext(gosnowflake.WithFileStream(ctx, deletedCSV), `PUT file://rows.csv @%"`+tempTableName+`"`)
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

// serializeRowsToCSV serializes rows as CSV, using columns as header, and
// returns it as an io.Reader. It also appends a boolean column called $PURGE
// with the value of the 'deleted' argument as value for each row.
func serializeRowsToCSV(columns []warehouses.Column, rows [][]any, deleted bool) (io.Reader, error) {
	var err error
	var b bytes.Buffer
	for i, c := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strings.ToUpper(c.Name))
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
// A string must be quoted if is empty, or starts with the character "'", or
// contains characters "," or "\n".
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
		if v == "" || v[0] == '\'' || strings.ContainsAny(v, ",\n") {
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
		quoteCSVBytes(b, v)
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
			quoteCSVBytes(b, value)
		case types.MapKind:
			value, err := types.Marshal(v.(map[string]any), t)
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
