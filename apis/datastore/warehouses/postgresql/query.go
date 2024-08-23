//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/meergo/meergo/apis/datastore/warehouses"
)

// Query executes a query and returns the results as Rows.
func (warehouse *PostgreSQL) Query(ctx context.Context, query warehouses.RowQuery, withCount bool) (warehouses.Rows, int, error) {

	db, err := warehouse.connection()
	if err != nil {
		return nil, 0, err
	}

	// Build the WHERE expression, if necessary.
	var whereExpr string
	if query.Where != nil {
		whereExpr, err = renderExpr(query.Where)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
	}

	var b strings.Builder

	// Count the total number of records.
	var count int
	if withCount {
		b.WriteString(`SELECT COUNT(*) FROM "`)
		b.WriteString(query.Table)
		b.WriteByte('"')
		if query.Where != nil {
			b.WriteString(` WHERE `)
			b.WriteString(whereExpr)
		}
		err = db.QueryRow(ctx, b.String()).Scan(&count)
		if err != nil {
			return nil, 0, warehouses.Error(err)
		}
		b.Reset()
	}

	// Build the query.
	b.WriteString(`SELECT `)
	for i, c := range query.Columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteByte('"')
	}
	b.WriteString(` FROM "`)
	b.WriteString(query.Table)
	b.WriteByte('"')
	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}

	if query.OrderBy.Name != "" {
		b.WriteString(" ORDER BY \"")
		b.WriteString(query.OrderBy.Name)
		b.WriteRune('"')
		if query.OrderDesc {
			b.WriteString(" DESC")
		}
	}
	if query.Limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(strconv.Itoa(query.Limit))
	}
	if query.First > 0 {
		b.WriteString(" OFFSET ")
		b.WriteString(strconv.Itoa(query.First))
	}

	// Execute the query.
	rows, err := db.Query(ctx, b.String())
	if err != nil {
		return nil, 0, warehouses.Error(err)
	}

	return rows, count, nil
}
