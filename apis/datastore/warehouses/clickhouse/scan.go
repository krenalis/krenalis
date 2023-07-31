//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"chichi/apis/normalization"
	"chichi/connector/types"
)

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	property    types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(properties []types.Property, rows *[][]any) []any {
	values := make([]any, len(properties))
	for i, p := range properties {
		values[i] = scanValue{
			property:    p,
			rows:        rows,
			columnIndex: i,
			columnCount: len(properties),
		}
	}
	return values
}

func (sv scanValue) Scan(src any) error {
	p := sv.property
	value, err := normalization.NormalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
	if err != nil {
		return err
	}
	var row []any
	if sv.columnIndex == 0 {
		row = make([]any, sv.columnCount)
		*sv.rows = append(*sv.rows, row)
	} else {
		row = (*sv.rows)[len(*sv.rows)-1]
	}
	row[sv.columnIndex] = value
	return nil
}
