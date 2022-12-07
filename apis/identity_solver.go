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
	row := ids.connections.myDB.QueryRow("SELECT `id` from `warehouse_users` where `Email` = ?", email)
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
	_, err = ids.connections.myDB.Exec(
		"UPDATE `connections_users` SET `golden_record` = ? WHERE `connection` = ? AND `user` = ?",
		goldenRecordID, connection, user,
	)
	if err != nil {
		return 0, fmt.Errorf("cannot update relation between entity and identity: %s", err)
	}
	// Clean orphan Golden Records.
	_, err = ids.connections.myDB.Exec("DELETE FROM `warehouse_users` WHERE `id` NOT IN (SELECT `golden_record` FROM `connections_users`)")
	if err != nil {
		return 0, fmt.Errorf("cannot clean orphan Golden Records: %s", err)
	}
	return goldenRecordID, nil

}

// createIdentity creates a new identity.
func (ids *identitySolver) createIdentity() (int, error) {
	result, err := ids.connections.myDB.Exec("INSERT INTO `warehouse_users` () VALUES()")
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	// TODO(Gianluca): review this conversion, depending on the data type we
	// will adopt on the database.
	return int(id), nil
}

// entityToIdentity maps an entity to the corresponding Golden Record identity,
// if found, otherwise returns 0, false and nil.
func (ids *identitySolver) entityToIdentity(connection int, user string) (int, bool, error) {
	query := "SELECT `golden_record` FROM `connections_users` WHERE `connection` = ? AND `user` = ?"
	row := ids.connections.myDB.QueryRow(query, connection, user)
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
	query := "SELECT `connection`, `user` FROM `connections_users`\n" +
		"WHERE `connection` <> ? AND `user` <> ? AND `golden_record` = \n" +
		"(SELECT `golden_record` FROM `connections_users` WHERE `connection` = ? AND `user` = ?)"
	rows, err := ids.connections.myDB.Query(query, connection, user, connection, user)
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
