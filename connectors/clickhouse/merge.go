//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"context"
	"strings"

	"github.com/meergo/meergo"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// merge performs a table merge operation.
func merge(ctx context.Context, db driver.Conn, table meergo.Table, rows [][]any) error {

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
		b.WriteByte('"')
		b.WriteString(column.Name)
		b.WriteByte('"')
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
			quoteValue(&b, row[j], column.Type)
		}
		b.WriteByte(')')
	}
	query := b.String()

	err = db.Exec(ctx, query)

	return err
}
