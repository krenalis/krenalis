// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

const simulatedRecordMaxPageSize = 1000

// SimulatedRecordsCursor identifies a position in simulated-record order.
type SimulatedRecordsCursor struct {
	UpdatedAt time.Time
	ID        string
}

// String returns the cursor encoded as Unix microseconds and record ID.
func (cursor SimulatedRecordsCursor) String() string {
	if cursor.UpdatedAt.IsZero() && cursor.ID == "" {
		return ""
	}
	return strconv.FormatInt(cursor.UpdatedAt.UnixMicro(), 10) + ":" + cursor.ID
}

// SimulatedRecordQuery specifies an incremental simulated-record page.
type SimulatedRecordQuery struct {
	Since time.Time
	After SimulatedRecordsCursor
	Limit int
}

// SimulatedRecord is a complete simulated-record document.
type SimulatedRecord struct {
	ID         string
	Attributes json.Value
	UpdatedAt  time.Time
}

// SimulatedRecordPage is one page of simulated records.
type SimulatedRecordPage struct {
	Records []SimulatedRecord
	Next    SimulatedRecordsCursor
	HasMore bool
}

// SimulatedSystemRecords returns an ordered page of one account's records.
func (store *Store) SimulatedSystemRecords(ctx context.Context, account string, query SimulatedRecordQuery) (SimulatedRecordPage, error) {

	store.mustBeOpen()

	if account == "" {
		return SimulatedRecordPage{}, errors.New("simulated system is missing")
	}
	if query.Limit < 1 || query.Limit > simulatedRecordMaxPageSize {
		return SimulatedRecordPage{}, errors.New("simulated-record limit is out of range")
	}
	if query.After.UpdatedAt.IsZero() != (query.After.ID == "") || strings.Contains(query.After.ID, ":") {
		return SimulatedRecordPage{}, errors.New("simulated-record cursor is malformed")
	}
	hasSince := !query.Since.IsZero()
	var since time.Time
	if hasSince {
		since = time.UnixMicro(query.Since.UnixMicro()).UTC()
		if since.Year() < 1 || since.Year() > 9999 {
			return SimulatedRecordPage{}, errors.New("simulated-record since is out of range")
		}
	}
	hasAfter := !query.After.UpdatedAt.IsZero()
	var after time.Time
	if hasAfter {
		after = time.UnixMicro(query.After.UpdatedAt.UnixMicro()).UTC()
		if after.Year() < 1 || after.Year() > 9999 {
			return SimulatedRecordPage{}, errors.New("simulated-record cursor is malformed")
		}
	}

	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return SimulatedRecordPage{}, err
	}
	defer done()

	columns := simulatedSystemRecordsTable.Columns
	accountColumn := columns[0]
	idColumn := columns[1]
	updatedAtColumn := columns[2]
	dataColumn := columns[3]

	where := []warehouses.Expr{
		warehouses.NewBaseExpr(accountColumn, warehouses.OpIs, account),
	}
	if hasSince {
		where = append(where, warehouses.NewBaseExpr(updatedAtColumn, warehouses.OpIsOnOrAfter, since))
	}
	if hasAfter {
		where = append(where, warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
			warehouses.NewBaseExpr(updatedAtColumn, warehouses.OpIsAfter, after),
			warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
				warehouses.NewBaseExpr(updatedAtColumn, warehouses.OpIs, after),
				warehouses.NewBaseExpr(idColumn, warehouses.OpIsGreaterThan, query.After.ID),
			}),
		}))
	}

	rows, _, err := store.warehouse().Query(ctx, warehouses.RowQuery{
		Columns: []warehouses.Column{idColumn, updatedAtColumn, dataColumn},
		Table:   simulatedSystemRecordsTable.Name,
		Where:   warehouses.NewMultiExpr(warehouses.OpAnd, where),
		OrderBy: []warehouses.Column{updatedAtColumn, idColumn},
		Limit:   query.Limit + 1,
	}, false)
	if err != nil {
		return SimulatedRecordPage{}, err
	}
	defer rows.Close()

	records := make([]SimulatedRecord, 0, query.Limit+1)
	row := make([]any, 3)
	for rows.Next() {
		if len(records) == query.Limit+1 {
			return SimulatedRecordPage{}, wrapWarehouseError(errors.New("warehouse returned too many simulated records"))
		}
		err = rows.Scan(row...)
		if err != nil {
			return SimulatedRecordPage{}, err
		}
		id, idOK := row[0].(string)
		updatedAt, updatedAtOK := row[1].(time.Time)
		attributes, attributesOK := row[2].(json.Value)
		if !idOK || id == "" || strings.Contains(id, ":") || !updatedAtOK || !attributesOK {
			return SimulatedRecordPage{}, wrapWarehouseError(errors.New("warehouse returned an invalid simulated record"))
		}
		records = append(records, SimulatedRecord{
			ID:         id,
			Attributes: slices.Clone(attributes),
			UpdatedAt:  updatedAt.UTC().Truncate(time.Microsecond),
		})
	}
	err = rows.Err()
	if err != nil {
		return SimulatedRecordPage{}, err
	}
	err = rows.Close()
	if err != nil {
		return SimulatedRecordPage{}, err
	}

	page := SimulatedRecordPage{Records: records}
	if len(records) > query.Limit {
		page.Records = records[:query.Limit]
		last := page.Records[len(page.Records)-1]
		page.Next = SimulatedRecordsCursor{UpdatedAt: last.UpdatedAt, ID: last.ID}
		page.HasMore = true
	}

	return page, nil
}

// ParseSimulatedRecordsCursor parses a simulated-record cursor.
func ParseSimulatedRecordsCursor(value string) (SimulatedRecordsCursor, error) {
	if value == "" {
		return SimulatedRecordsCursor{}, nil
	}
	microseconds, id, found := strings.Cut(value, ":")
	if !found || microseconds == "" || id == "" || strings.Contains(id, ":") {
		return SimulatedRecordsCursor{}, errors.New("simulated-record cursor is malformed")
	}
	n, err := strconv.ParseInt(microseconds, 10, 64)
	if err != nil {
		return SimulatedRecordsCursor{}, errors.New("simulated-record cursor is malformed")
	}
	updatedAt := time.UnixMicro(n).UTC()
	if updatedAt.Year() < 1 || updatedAt.Year() > 9999 {
		return SimulatedRecordsCursor{}, errors.New("simulated-record cursor is malformed")
	}
	return SimulatedRecordsCursor{UpdatedAt: updatedAt, ID: id}, nil
}
