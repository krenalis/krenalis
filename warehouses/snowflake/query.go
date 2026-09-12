// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/warehouses"
)

// Count returns the number of rows in table after applying joins and where.
// A nil where expression does not filter the rows.
func (warehouse *Snowflake) Count(ctx context.Context, table string, joins []warehouses.Join, where warehouses.Expr) (int, error) {

	db, err := warehouse.openDB(ctx)
	if err != nil {
		return 0, snowflake(err)
	}
	statement, err := renderCountQuery(table, joins, where)
	if err != nil {
		return 0, err
	}
	var total int
	err = db.QueryRowContext(ctx, statement).Scan(&total)
	if err != nil {
		return 0, snowflake(err)
	}

	return total, nil
}

// Query executes a query and returns the results as Rows.
func (warehouse *Snowflake) Query(ctx context.Context, query warehouses.RowQuery, withTotal bool) (warehouses.Rows, int, error) {

	db, err := warehouse.openDB(ctx)
	if err != nil {
		return nil, 0, snowflake(err)
	}

	// Build the WHERE expression, if necessary.
	var whereExpr string
	if query.Where != nil {
		var s strings.Builder
		err := renderExpr(&s, query.Where)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
		whereExpr = s.String()
	}

	var b strings.Builder

	var total int
	if withTotal {
		statement, countErr := renderCountQuery(query.Table, query.Joins, query.Where)
		if countErr != nil {
			return nil, 0, countErr
		}
		err = db.QueryRowContext(ctx, statement).Scan(&total)
		if err != nil {
			return nil, 0, snowflake(err)
		}
	}

	// Build the query.
	b.WriteString(`SELECT `)
	for i, c := range query.Columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(quoteIdent(c.Name))
	}
	b.WriteString(` FROM `)
	b.WriteString(quoteIdent(query.Table))

	err = appendJoins(&b, query.Joins)
	if err != nil {
		return nil, 0, err
	}

	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}

	if len(query.OrderBy) > 0 {
		b.WriteString(" ORDER BY ")
		for i, column := range query.OrderBy {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteIdent(column.Name))
			if query.OrderDesc {
				b.WriteString(" DESC")
			}
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
	rows, err := db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, 0, snowflake(err)
	}

	return newScanner(query.Columns, rows), total, nil
}

// appendJoins appends the string serialization of the provided joins to b.
func appendJoins(b *strings.Builder, joins []warehouses.Join) error {
	for _, join := range joins {
		switch join.Type {
		case warehouses.InnerJoin:
			b.WriteString(` JOIN `)
		case warehouses.LeftJoin:
			b.WriteString(` LEFT JOIN `)
		case warehouses.RightJoin:
			b.WriteString(` RIGHT JOIN `)
		case warehouses.FullJoin:
			b.WriteString(` FULL JOIN `)
		}
		b.WriteString(quoteIdent(join.Table))
		b.WriteString(` ON `)
		err := renderExpr(b, join.Condition)
		if err != nil {
			return fmt.Errorf("cannot build JOIN condition: %s", err)
		}
	}
	return nil
}

// renderCountQuery renders a query that counts rows in table after applying joins and where.
func renderCountQuery(table string, joins []warehouses.Join, where warehouses.Expr) (string, error) {

	var b strings.Builder
	b.WriteString(`SELECT COUNT(*) FROM `)
	b.WriteString(quoteIdent(table))
	err := appendJoins(&b, joins)
	if err != nil {
		return "", err
	}
	if where != nil {
		b.WriteString(` WHERE `)
		err = renderExpr(&b, where)
		if err != nil {
			return "", fmt.Errorf("cannot build WHERE expression: %s", err)
		}
	}

	return b.String(), nil
}
