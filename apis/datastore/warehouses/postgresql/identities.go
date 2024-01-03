//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"chichi/apis/datastore/warehouses"
)

// SetIdentity sets the identity id (which may have an anonymous ID) imported
// from the action. fromEvents indicates if the identity has been imported from
// an event or not.
// timestamp is the timestamp that will be associated to the imported identity.
func (warehouse *PostgreSQL) SetIdentity(ctx context.Context, identity map[string]any, id string, anonID string, action int, fromEvent bool, timestamp time.Time) error {

	// Retrieve the database connection.
	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	// Query the matching user identities, which can be 0 (the identity is a new
	// identity), 1 (the identity already exists and must be updated) or more
	// (the new identity requires a merging of already existing identities).
	var query string
	var args []any
	if fromEvent {
		if isAnon := id == ""; isAnon {
			query = "SELECT __identity_id__ FROM users_identities WHERE " +
				"$1 IN __anonymous_ids__ ORDER BY __timestamp__, __identity_id__"
			args = []any{anonID}
		} else {
			query = "SELECT __identity_id__ FROM users_identities WHERE " +
				"(__external_id__ = $1) OR ($2 = ANY(__anonymous_ids__)) ORDER BY __timestamp__, __identity_id__"
			args = []any{id, anonID}
		}
	} else { // app, file or database.
		query = "SELECT __identity_id__ FROM users_identities WHERE " +
			"__external_id__ = $1 ORDER BY __timestamp__, __identity_id__"
		args = []any{id}
	}
	var matchingIdentities []int
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return warehouses.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		var e int
		if err = rows.Scan(&e); err != nil {
			return warehouses.Error(err)
		}
		matchingIdentities = append(matchingIdentities, e)
	}
	rows.Close()
	if rows.Err() != nil {
		return warehouses.Error(err)
	}

	// Create the new identity.
	var newIdentityID int
	identity["__action__"] = action
	identity["__external_id__"] = id
	identity["__timestamp__"] = timestamp.Format(time.DateTime)
	if anonID != "" {
		identity["__anonymous_ids__"] = []string{anonID}
	}
	b := strings.Builder{}
	b.WriteString("INSERT INTO users_identities (")
	properties := maps.Keys(identity)
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(name)
		b.WriteByte('"')
	}
	b.WriteString(") VALUES (")
	values := make([]any, len(properties))
	for i, name := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(i + 1))
		values[i] = identity[name]
	}
	b.WriteString(") RETURNING __identity_id__")
	err = db.QueryRow(ctx, b.String(), values...).Scan(&newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// There are no matching identities, so the identity has been created and
	// there's nothing else to do.
	if len(matchingIdentities) == 0 {
		return nil
	}

	// Merge the matching identity (or identities) into the new one.

	var idsStr strings.Builder
	for _, id := range matchingIdentities {
		idsStr.WriteString(strconv.Itoa(id))
		idsStr.WriteByte(',')
	}
	idsStr.WriteString(strconv.Itoa(newIdentityID))

	// Merge the anonymous IDS.
	b.Reset()
	b.WriteString(`UPDATE users_identities SET __anonymous_ids__ = (
		SELECT array_agg(anon_ids.ids) as __anonymous_ids__
		FROM (
			SELECT unnest("__anonymous_ids__") as ids
			FROM users_identities
			WHERE __identity_id__ IN (`)
	b.WriteString(idsStr.String())
	b.WriteString(`) AND __anonymous_ids__ IS NOT NULL
			) AS anon_ids
		) WHERE __identity_id__ = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// Merge the other fields.
	b.Reset()
	b.WriteString("UPDATE users_identities SET ")
	comma := false
	for _, p := range properties {
		if p == "__action__" || p == "__anonymous_ids__" || p == "__external_id__" || p == "__timestamp__" {
			continue
		}
		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(p)
		b.WriteString(`" = (SELECT "`)
		b.WriteString(p)
		b.WriteString(`" FROM users_identities WHERE "`)
		b.WriteString(p)
		b.WriteString(`" IS NOT NULL AND __identity_id__ IN (`)
		b.WriteString(idsStr.String())
		b.WriteString(") ORDER BY __timestamp__ DESC, __identity_id__ DESC LIMIT 1)\n")
		comma = true
	}
	b.WriteString(` WHERE __identity_id__ = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// Delete the merged identities.
	var idsToDelete strings.Builder
	for i, id := range matchingIdentities {
		if i > 0 {
			idsToDelete.WriteByte(',')
		}
		idsToDelete.WriteString(strconv.Itoa(id))
	}
	b.Reset()
	b.WriteString("DELETE FROM users_identities WHERE __identity_id__ IN (")
	b.WriteString(idsToDelete.String())
	b.WriteByte(')')
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}
