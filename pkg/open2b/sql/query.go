// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2013-2017 Open2b
//

package sql

import (
	"database/sql"
	"time"

	"chichi/pkg/open2b/decimal"

	. "github.com/go-sql-driver/mysql"
)

type Query struct {
	table     *Table
	statement string
}

func NewQuery(table *Table, statement string) Query {
	return Query{
		table:     table,
		statement: statement,
	}
}

func (query Query) Scan(scan func(rows *Rows) error) error {
	return query.table.connection.QueryScan(query.statement, scan)
}

func (query Query) Rows() ([]map[string]any, error) {

	innerRows, err := query.table.connection.Query(query.statement)
	if err != nil {
		return nil, err
	}
	defer innerRows.Close()

	columns, err := innerRows.Columns()
	if err != nil {
		return nil, err
	}

	var driverTypeOf = map[string]int{}
	{
		columnTypes, err := innerRows.ColumnTypes()
		if err != nil {
			return nil, err
		}
		for _, column := range columnTypes {
			switch column.DatabaseTypeName() {
			case "INT":
				if _, nullable := column.Nullable(); nullable {
					driverTypeOf[column.Name()] = nullIntColumn
				} else {
					driverTypeOf[column.Name()] = intColumn
				}
			case "BIGINT":
				if _, nullable := column.Nullable(); nullable {
					driverTypeOf[column.Name()] = nullInt64Column
				} else {
					driverTypeOf[column.Name()] = int64Column
				}
			case "DATE", "DATETIME":
				if _, nullable := column.Nullable(); nullable {
					driverTypeOf[column.Name()] = nullTimeColumn
				} else {
					driverTypeOf[column.Name()] = timeColumn
				}
			case "DECIMAL":
				if _, nullable := column.Nullable(); nullable {
					driverTypeOf[column.Name()] = nullDecimalColumn
				} else {
					driverTypeOf[column.Name()] = decimalColumn
				}
			default:
				if _, nullable := column.Nullable(); nullable {
					driverTypeOf[column.Name()] = nullStringColumn
				} else {
					driverTypeOf[column.Name()] = stringColumn
				}
			}
		}
	}

	var innerRow = make([]any, len(columns))

	var scheme = query.table.scheme

	for i, column := range columns {
		t, ok := scheme.columns[column]
		if !ok {
			t = driverTypeOf[column]
		}
		switch t {
		case boolColumn:
			var value bool
			innerRow[i] = &value
		case nullBoolColumn:
			var value sql.NullBool
			innerRow[i] = &value
		case float32Column:
			var value float32
			innerRow[i] = &value
		case float64Column:
			var value float64
			innerRow[i] = &value
		case nullFloat32Column, nullFloat64Column:
			var value sql.NullFloat64
			innerRow[i] = &value
		case intColumn:
			var value int
			innerRow[i] = &value
		case int64Column:
			var value int64
			innerRow[i] = &value
		case uint64Column:
			var value uint64
			innerRow[i] = &value
		case nullIntColumn, nullInt64Column:
			var value sql.NullInt64
			innerRow[i] = &value
		case stringColumn, decimalColumn:
			var value string
			innerRow[i] = &value
		case timeColumn:
			var value time.Time
			innerRow[i] = &value
		case nullTimeColumn:
			var value NullTime
			innerRow[i] = &value
		default:
			var value sql.NullString
			innerRow[i] = &value
		}
	}

	var rows []map[string]any

	for innerRows.Next() {
		err = innerRows.Scan(innerRow...)
		if err != nil {
			return nil, err
		}
		var row = map[string]any{}
		for i, column := range columns {
			t, ok := scheme.columns[column]
			if !ok {
				t = driverTypeOf[column]
			}

			switch val := innerRow[i].(type) {

			case *bool:
				row[column] = *val
			case *sql.NullBool:
				if val.Valid {
					row[column] = *val
				} else {
					row[column] = nil
				}

			case *float32:
				row[column] = *val

			case *float64:
				row[column] = *val
			case *sql.NullFloat64:
				if val.Valid {
					if t == nullFloat32Column {
						row[column] = float32(val.Float64)
					} else {
						row[column] = val.Float64
					}
				} else {
					row[column] = nil
				}

			case *int:
				row[column] = *val

			case *int64:
				row[column] = *val
			case *sql.NullInt64:
				if val.Valid {
					if t == nullIntColumn {
						row[column] = int(val.Int64)
					} else {
						row[column] = val.Int64
					}
				} else {
					row[column] = nil
				}

			case *time.Time:
				row[column] = *val
			case *NullTime:
				if val.Valid {
					row[column] = val.Time
				} else {
					row[column] = nil
				}

			case *string:
				if t == decimalColumn {
					var ok bool
					row[column], ok = decimal.ParseString(*val)
					if !ok {
						panic("Failed to parse decimal.Dec: column " + column)
					}
				} else {
					row[column] = *val
				}
			case *sql.NullString:
				if val.Valid {
					if t == nullDecimalColumn {
						var ok bool
						row[column], ok = decimal.ParseString(val.String)
						if !ok {
							panic("Failed to parse decimal.Dec: column " + column)
						}
					} else {
						row[column] = val.String
					}
				} else {
					row[column] = nil
				}

			}
		}
		rows = append(rows, row)
	}

	if err = innerRows.Err(); err != nil {
		return nil, err
	}

	return rows, nil
}

func (query Query) Statement() string {
	return query.statement
}
