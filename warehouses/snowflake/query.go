//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/meergo/meergo"
)

// Query executes a query and returns the results as Rows.
func (warehouse *Snowflake) Query(ctx context.Context, query meergo.RowQuery, withCount bool) (meergo.Rows, int, error) {

	db, err := warehouse.connection()
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
	var count int
	if withCount {
		b.WriteString(`SELECT COUNT(*) FROM `)
		b.WriteString(quoteTable(query.Table))
		err = appendJoins(&b, query.Joins)
		if err != nil {
			return nil, 0, err
		}
		if query.Where != nil {
			b.WriteString(` WHERE `)
			b.WriteString(whereExpr)
		}
		err = db.QueryRowContext(ctx, b.String()).Scan(&count)
		if err != nil {
			return nil, 0, meergo.Error(err)
		}
		b.Reset()
	}

	// Build the query.
	b.WriteString(`SELECT `)
	for i, c := range query.Columns {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteColumn(c.Name))
	}
	b.WriteString(` FROM `)
	b.WriteString(quoteTable(query.Table))

	err = appendJoins(&b, query.Joins)
	if err != nil {
		return nil, 0, err
	}

	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}

	if query.OrderBy.Name != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(quoteColumn(query.OrderBy.Name))
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
	rows, err := db.QueryContext(ctx, b.String())
	if err != nil {
		return nil, 0, meergo.Error(err)
	}

	return newScanner(query.Columns, rows), count, nil
}

// appendJoins appends the string serialization of the provided joins to b.
func appendJoins(b *strings.Builder, joins []meergo.Join) error {
	for _, join := range joins {
		switch join.Type {
		case meergo.Inner:
			b.WriteString(` JOIN `)
		case meergo.Left:
			b.WriteString(` LEFT JOIN `)
		case meergo.Right:
			b.WriteString(` RIGHT JOIN `)
		case meergo.Full:
			b.WriteString(` FULL JOIN `)
		}
		b.WriteString(quoteTable(join.Table))
		b.WriteString(` ON `)
		err := renderExpr(b, join.Condition)
		if err != nil {
			return fmt.Errorf("cannot build JOIN condition: %s", err)
		}
	}
	return nil
}
