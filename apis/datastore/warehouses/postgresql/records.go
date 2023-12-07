//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"errors"
	"fmt"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
	"chichi/connector/types"
)

type records struct {
	columns              []types.Property
	properties           []types.Property
	rows                 *postgres.Rows
	key                  string
	dst                  []any
	err                  error
	closed               bool
	removeKeyFromRecords bool
}

var _ warehouses.Records = (*records)(nil)

// newRecords return a new records.
// It could change the columns slice and the column names.
// removeKeyFromRecords controls whether the key property should be removed by
// the returned records.
func newRecords(rows *postgres.Rows, columns []types.Property, key string, removeKeyFromRecords bool) (*records, error) {
	properties, err := warehouses.ColumnsToProperties(columns)
	if err != nil {
		return nil, err
	}
	records := records{
		columns:              columns,
		properties:           properties,
		rows:                 rows,
		key:                  key,
		dst:                  make([]any, len(columns)),
		removeKeyFromRecords: removeKeyFromRecords,
	}
	return &records, nil
}

func (r *records) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.rows.Close()
	return nil
}

func (r *records) Err() error {
	return r.err
}

func (r *records) For(yield func(warehouses.Record) error) error {
	if r.closed {
		r.err = errors.New("for called on a closed Records")
		return nil
	}
	defer r.Close()
	var rows [][]any
	values := newScanValues(r.columns, &rows)
	for r.rows.Next() {
		if err := r.rows.Scan(values...); err != nil {
			r.err = err
			return nil
		}
		var record warehouses.Record
		props, _ := warehouses.DeserializeRowAsMap(r.properties, rows[len(rows)-1])
		id, ok := props[r.key].(int)
		if !ok {
			r.err = fmt.Errorf("row has no integer key %q", r.key)
			return nil
		}
		record.ID = id
		if r.removeKeyFromRecords {
			delete(props, r.key)
		}
		record.Properties = props
		if err := yield(record); err != nil {
			return err
		}
	}
	if err := r.rows.Err(); err != nil {
		r.err = err
	}
	return nil
}
