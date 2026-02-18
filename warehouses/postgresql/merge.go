// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

// Keep this file in sync between the warehouse and the connector, except for
// the "warehouses" and "connectors" import lines.

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// merge performs a table merge operation.
func merge(ctx context.Context, conn *pgxpool.Conn, table warehouses.Table, rows [][]any, deleted []any) error {

	quotedColumn := make(map[string]string, len(table.Columns))
	for _, column := range table.Columns {
		quotedColumn[column.Name] = quoteIdent(column.Name)
	}

	// Generate a unique name for the temporary table.
	tempTableName := "temp_table_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Prepare the "create temporary table" statement.
	var b strings.Builder
	b.WriteString(`CREATE TEMPORARY TABLE "`)
	b.WriteString(tempTableName)
	b.WriteString("\" AS\n  SELECT ")
	for _, c := range table.Columns {
		b.WriteString(quotedColumn[c.Name])
		b.WriteByte(',')
	}
	b.WriteString(`FALSE AS "$purge" FROM `)
	b.WriteString(quoteIdent(table.Name))
	b.WriteString("\nWITH NO DATA")
	create := b.String()

	// Create the 'merge' statement.
	b.Reset()
	b.WriteString(`MERGE INTO `)
	b.WriteString(quoteIdent(table.Name))
	b.WriteString(" \"d\"\nUSING \"")
	b.WriteString(tempTableName)
	b.WriteString("\" \"s\"\nON ")
	for i, key := range table.Keys {
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(`"d".`)
		b.WriteString(quotedColumn[key])
		b.WriteString(` = "s".`)
		b.WriteString(quotedColumn[key])
	}
	if len(rows) > 0 {
		b.WriteString("\nWHEN MATCHED AND \"s\".\"$purge\" IS NULL THEN\n  UPDATE SET ")
		i := 0
		for _, c := range table.Columns {
			if slices.Contains(table.Keys, c.Name) {
				continue
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quotedColumn[c.Name])
			b.WriteString(` = "s".`)
			b.WriteString(quotedColumn[c.Name])
			i++
		}
		if i == 0 {
			return errors.New("postgresql.Merge: there must be at least one column in 'columns' apart from the keys")
		}
		b.WriteString("\nWHEN NOT MATCHED AND \"s\".\"$purge\" IS NULL THEN\n  INSERT (")
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
			b.WriteString(`"s".`)
			b.WriteString(quotedColumn[c.Name])
		}
		b.WriteString(`)`)
	}
	if len(deleted) > 0 {
		b.WriteString("\nWHEN MATCHED THEN\n  DELETE")
	}
	merge := b.String()

	// Encode the rows and prepare the columns names.
	var columnNames []string
	if len(rows) > 0 {
		columnNames = make([]string, len(table.Columns))
		for i, c := range table.Columns {
			columnNames[i] = c.Name
		}
		if enc, ok := newRowEncoder(table.Columns); ok {
			for _, row := range rows {
				err := enc.encode(row)
				if err != nil {
					return err
				}
			}
		}
	}

	// Prepare the columns names for the rows to delete.
	var columnsNamesDeleted []string
	if len(deleted) > 0 {
		columnsNamesDeleted = make([]string, len(table.Keys)+1)
		copy(columnsNamesDeleted, table.Keys)
		columnsNamesDeleted[len(columnsNamesDeleted)-1] = "$purge"
	}

	// Create the temporary table.
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return err
	}

	// Copy the rows into the temporary table.
	if len(rows) > 0 {
		_, err = conn.CopyFrom(ctx, []string{tempTableName}, columnNames, pgx.CopyFromRows(rows))
		if err != nil {
			return err
		}
	}
	// Copy the rows to delete into the temporary table.
	if len(deleted) > 0 {
		_, err = conn.CopyFrom(ctx, []string{tempTableName}, columnsNamesDeleted, newCopyForDeleteFrom(len(table.Keys), deleted))
		if err != nil {
			return err
		}
	}
	// Merge the temporary table's rows with the destination table's rows.
	_, err = conn.Exec(ctx, merge)
	if err != nil {
		return err
	}

	return nil
}

// copyForDeleteFrom implements the pgx.CopyFromSource interface.
type copyForDeleteFrom struct {
	keys []any
	row  []any
}

// newCopyForDeleteFrom returns a pgx.CopyFromSource implementation used to
// delete rows from a table. Rows are read from keys, where each row contains
// numTableKeys consecutive elements from keys and true at the end.
func newCopyForDeleteFrom(numTableKeys int, keys []any) pgx.CopyFromSource {
	c := &copyForDeleteFrom{
		keys: keys,
		row:  make([]any, numTableKeys+1),
	}
	c.row[numTableKeys] = true
	return c
}

func (c *copyForDeleteFrom) Err() error {
	return nil
}

func (c *copyForDeleteFrom) Next() bool {
	return len(c.keys) > 0
}

func (c *copyForDeleteFrom) Values() ([]any, error) {
	n := len(c.row) - 1
	for i := range n {
		c.row[i] = c.keys[i]
	}
	c.keys = c.keys[n:]
	return c.row, nil
}

// stripZeroBytes removes all zero bytes ('\x00') from s and returns the
// modified slice.
func stripZeroBytes(s string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(s, '\x00')
		if i == -1 {
			break
		}
		b.WriteString(s[:i])
		s = s[i+1:]
	}
	if b.Len() > 0 {
		b.WriteString(s)
		return b.String()
	}
	return s
}

// rowEncoder implements a row encoder that encodes rows to be used in a merge
// function.
type rowEncoder struct {
	ct map[int]types.Type
}

// newRowEncoder returns a new row encoder that encodes rows with the provided
// columns. If there are no columns to encode, it returns nil and false;
// otherwise, it returns the new encoder and true.
func newRowEncoder(columns []warehouses.Column) (*rowEncoder, bool) {
	var ct map[int]types.Type
	for i, c := range columns {
		switch c.Type.Kind() {
		case types.ArrayKind:
			if k := c.Type.Elem().Kind(); k != types.StringKind && k != types.JSONKind {
				continue
			}
			fallthrough
		case types.StringKind, types.JSONKind, types.MapKind:
			if ct == nil {
				ct = map[int]types.Type{i: c.Type}
			} else {
				ct[i] = c.Type
			}
		}
	}
	if ct == nil {
		return nil, false
	}
	return &rowEncoder{ct: ct}, true
}

// encode encodes a row to be used with a merge method. It removes zero bytes
// from text, json, array(text), array(json), and map values, and encodes map
// values as JSON.
func (enc rowEncoder) encode(row []any) error {
	for i, t := range enc.ct {
		if row[i] == nil {
			continue
		}
		switch t.Kind() {
		case types.StringKind:
			row[i] = stripZeroBytes(row[i].(string))
		case types.JSONKind:
			row[i] = json.Value(json.StripZeroBytes(row[i].(json.Value)))
		case types.ArrayKind:
			arr := row[i].([]any)
			if k := t.Elem().Kind(); k == types.JSONKind {
				for j, s := range arr {
					arr[j] = json.Value(json.StripZeroBytes(s.(json.Value)))
				}
			} else {
				for j, s := range arr {
					arr[j] = stripZeroBytes(s.(string))
				}
			}
		case types.MapKind:
			b, err := types.Marshal(row[i].(map[string]any), t)
			if err != nil {
				return err
			}
			row[i] = json.Value(json.StripZeroBytes(b))
		}
	}
	return nil
}
