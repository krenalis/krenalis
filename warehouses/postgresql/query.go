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

	"github.com/meergo/meergo"
)

// Query executes a query and returns the results as Rows.
func (warehouse *PostgreSQL) Query(ctx context.Context, query meergo.RowQuery, withTotal bool) (meergo.Rows, int, error) {

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Build the WHERE expression, if necessary.
	var whereExpr string
	if query.Where != nil {
		var s strings.Builder
		err = renderExpr(&s, query.Where)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
		whereExpr = s.String()
	}

	var b strings.Builder

	// Count the total number of records.
	var total int
	if withTotal {
		b.WriteString(`SELECT COUNT(*) FROM `)
		b.WriteString(quoteIdent(query.Table))
		err = appendJoins(&b, query.Joins)
		if err != nil {
			return nil, 0, err
		}
		if query.Where != nil {
			b.WriteString(` WHERE `)
			b.WriteString(whereExpr)
		}
		err = pool.QueryRow(ctx, b.String()).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		b.Reset()
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

	if query.OrderBy != nil {
		b.WriteString(" ORDER BY ")
		for i, column := range query.OrderBy {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteIdent(column.Name))
		}
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
	rows, err := pool.Query(ctx, b.String())
	if err != nil {
		return nil, 0, err
	}

	return newScanner(query.Columns, rows), total, nil
}

// appendJoins appends the string serialization of the provided joins to b.
func appendJoins(b *strings.Builder, joins []meergo.Join) error {
	for _, join := range joins {
		switch join.Type {
		case meergo.InnerJoin:
			b.WriteString(` JOIN `)
		case meergo.LeftJoin:
			b.WriteString(` LEFT JOIN `)
		case meergo.RightJoin:
			b.WriteString(` RIGHT JOIN `)
		case meergo.FullJoin:
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
