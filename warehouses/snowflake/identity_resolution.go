//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	"database/sql"
	_ "embed"
	"time"

	"github.com/meergo/meergo"
)

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	panic("TODO: not implemented")
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
func (warehouse *Snowflake) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, nil, meergo.Error(err)
	}
	defer conn.Close()
	// TODO(Gianluca).
	// err = warehouse.fixOperationsTable(ctx)
	// if err != nil {
	// 	return nil, nil, err
	// }
	query := `SELECT "start_time", "end_time" FROM "_operations" WHERE ` +
		`"operation" = 'IdentityResolution' ORDER BY "id" DESC LIMIT 1`
	err = conn.QueryRowContext(ctx, query).Scan(&startTime, &endTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, meergo.Error(err)
	}
	return startTime, endTime, nil
}
