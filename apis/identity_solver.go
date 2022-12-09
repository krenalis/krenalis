//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"database/sql"
	"fmt"
)

// An identitySolver performs the identity resolution process.
type identitySolver struct {
	*firehose
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
	row := ids.connections.db.QueryRow("SELECT id from warehouse_users where Email = $1", email)
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
	_, err = ids.connections.db.Exec(
		"UPDATE connections_users SET golden_record = ? WHERE connection = $1 AND user = $2",
		goldenRecordID, connection, user,
	)
	if err != nil {
		return 0, fmt.Errorf("cannot update relation between entity and identity: %s", err)
	}
	// Clean orphan Golden Records.
	_, err = ids.connections.db.Exec("DELETE FROM warehouse_users WHERE id NOT IN (SELECT golden_record FROM connections_users)")
	if err != nil {
		return 0, fmt.Errorf("cannot clean orphan Golden Records: %s", err)
	}
	return goldenRecordID, nil

}

// createIdentity creates a new identity.
func (ids *identitySolver) createIdentity() (int, error) {
	var id int
	err := ids.connections.db.QueryRow("INSERT INTO warehouse_users () VALUES() RETURNING id").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// entityToIdentity maps an entity to the corresponding Golden Record identity,
// if found, otherwise returns 0, false and nil.
func (ids *identitySolver) entityToIdentity(connection int, user string) (int, bool, error) {
	query := "SELECT golden_record FROM connections_users WHERE connection = $1 AND user = $2"
	row := ids.connections.db.QueryRow(query, connection, user)
	var goldenRecord int
	err := row.Scan(&goldenRecord)
	if err != nil {
		return 0, false, err
	}
	if goldenRecord == 0 {
		return 0, false, nil
	}
	return goldenRecord, true, nil
}

// LookupSameEntities returns the entities which correspond to the Golden
// Record's identity, in the form of a map from connection to the list of users
// associated to that connection.
func (ids *identitySolver) LookupSameEntities(connection int, user string) (map[int][]string, error) {
	query := "SELECT connection, user FROM connections_users\n" +
		"WHERE connection <> $1 AND user <> $2 AND golden_record = \n" +
		"(SELECT golden_record FROM connections_users WHERE connection = $1 AND user = $2)"
	rows, err := ids.connections.db.Query(query, connection, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sameEntities := map[int][]string{}
	for rows.Next() {
		var connection int
		var user string
		err := rows.Scan(&connection, &user)
		if err != nil {
			return nil, err
		}
		sameEntities[connection] = append(sameEntities[connection], user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sameEntities, nil
}
