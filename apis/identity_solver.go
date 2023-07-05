//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"fmt"

	"chichi/apis/state"
)

// An identitySolver performs the identity resolution process.
//
// TODO(Gianluca): remove this type and also consider removing this file.
type identitySolver struct {
	ctx        context.Context
	connection *state.Connection
}

// ResolveEntity resolves the user entity, from the given connection, matching
// on their email.
func (ids *identitySolver) ResolveEntity(connection int, user string, email string) (int, error) {
	goldenRecordID, ok, err := ids.entityToIdentity(connection, user)
	if err != nil {
		return 0, fmt.Errorf("cannot determine identity from entity: %s", err)
	}
	if ok {
		return goldenRecordID, nil // already resolved.
	}
	// Lookup a Golden Record with this email.
	ws := ids.connection.Workspace()
	row := ws.Warehouse.QueryRow(ids.ctx, `SELECT id from users where "Email" = $1`, email)
	err = row.Scan(&goldenRecordID)
	if err != nil {
		if err != sql.ErrNoRows {
			return 0, err
		}
		goldenRecordID, err = ids.createIdentity()
		if err != nil {
			return 0, fmt.Errorf("cannot create identity: %s", err)
		}
	}
	// Update the relations.
	result, err := ws.Warehouse.Exec(ids.ctx,
		"UPDATE connections_users SET golden_record = $1 WHERE connection = $2 AND \"user\" = $3",
		goldenRecordID, connection, user,
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, fmt.Errorf("BUG: should have updated one row, got %d", affected)
	}
	// Clean orphan Golden Records.
	_, err = ws.Warehouse.Exec(ids.ctx, "DELETE FROM users WHERE id NOT IN (SELECT golden_record FROM connections_users)")
	if err != nil {
		return 0, fmt.Errorf("cannot clean orphan Golden Records: %s", err)
	}
	return goldenRecordID, nil

}

// createIdentity creates a new identity.
func (ids *identitySolver) createIdentity() (int, error) {
	var id int
	ws := ids.connection.Workspace()
	err := ws.Warehouse.QueryRow(ids.ctx, "INSERT INTO users DEFAULT VALUES RETURNING id").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// entityToIdentity maps an entity to the corresponding Golden Record identity,
// if found, otherwise returns 0, false and nil.
func (ids *identitySolver) entityToIdentity(connection int, user string) (int, bool, error) {
	ws := ids.connection.Workspace()
	query := "SELECT golden_record FROM connections_users WHERE connection = $1 AND \"user\" = $2"
	row := ws.Warehouse.QueryRow(ids.ctx, query, connection, user)
	var goldenRecord int
	err := row.Scan(&goldenRecord)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	if goldenRecord == 0 {
		return 0, false, nil
	}
	return goldenRecord, true, nil
}
