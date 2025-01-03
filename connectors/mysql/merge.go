//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package mysql

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/meergo/meergo"
)

// merge performs a table merge operation.
//
// It is necessary for the table keys to match the primary keys of the table in
// order to make this method operate correctly.
func merge(ctx context.Context, conn *sql.Conn, table meergo.Table, rows [][]any) error {

	name, err := quoteTable(table.Name)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(name)
	b.WriteString(" (")
	for i, column := range table.Columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteByte('`')
	}
	b.WriteString(") VALUES ")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("(")
		for j, column := range table.Columns {
			if j > 0 {
				b.WriteByte(',')
			}
			if err = quoteValue(&b, row[j], column.Type); err != nil {
				return err
			}
		}
		b.WriteByte(')')
	}
	b.WriteString(` ON DUPLICATE KEY UPDATE `)
	i := 0
	for _, column := range table.Columns {
		if slices.Contains(table.Keys, column.Name) {
			continue
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteString("` = VALUES(`")
		b.WriteString(column.Name)
		b.WriteString("`)")
		i++
	}
	query := b.String()

	_, err = conn.ExecContext(ctx, query)

	return err
}
