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

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"

	"golang.org/x/exp/maps"
)

// IdentitiesWriter returns an IdentitiesWriter for writing user identities,
// relative to the action, on the data warehouse.
// fromEvent indicates if the user identities are imported from an event or not.
// ack is the ack function (see the documentation of IdentitiesWriter for more
// details about it).
func (warehouse *PostgreSQL) IdentitiesWriter(ctx context.Context, action int, fromEvent bool, ack warehouses.IdentitiesAckFunc) warehouses.IdentitiesWriter {
	if ack == nil {
		panic("ack function is missing")
	}
	return &identitiesWriter{
		warehouse: warehouse,
		action:    action,
		fromEvent: fromEvent,
		ack:       ack,
	}
}

type identitiesWriter struct {
	warehouse *PostgreSQL
	action    int
	fromEvent bool
	ack       warehouses.IdentitiesAckFunc
	err       error
	closed    bool
}

var _ warehouses.IdentitiesWriter = (*identitiesWriter)(nil)

// Close closes the IdentitiesWriter, ensuring the completion of all pending or
// ongoing write operations. In the event of a canceled context, it interrupts
// ongoing writes, discards pending ones, and returns.
//
// In case an error occurs with the data warehouse, a DataWarehouseError error
// is returned.
//
// If the IdentitiesWriter is already closed, it does nothing and returns
// immediately.
func (iw *identitiesWriter) Close(ctx context.Context) error {
	iw.closed = false
	return iw.err
}

// Write writes a user identity. Typically, Write returns immediately, deferring
// the actual write operation to a later time. If it returns false, no further
// Write operations can be performed, and a call to Close will return an error.
//
// If the user identity is successfully written, the ack function is invoked
// with a nil error and the record's ID as arguments. If writing the record
// fails, the ack function is invoked with a non-nil error and the user
// identity's ID as arguments. The ack function is invoked even if Write returns
// false.
//
// It panics if called on a closed writer.
func (iw *identitiesWriter) Write(ctx context.Context, identity warehouses.Identity) bool {
	if iw.closed {
		panic("the Write method have been called on a closed IdentitiesWriter")
	}
	db, err := iw.warehouse.connection()
	if err != nil {
		iw.err = err
		return false
	}
	// TODO(Gianluca): This implementation, instead of returning immediately
	// after buffering the user identities to be written all together, directly
	// calls the underlying data warehouse to write. This needs to be optimized
	// for bulk writing rather than writing individual users.
	err = writeUserIdentity(ctx, db, identity.Properties, identity.ID, identity.AnonymousID, iw.action, iw.fromEvent, identity.Timestamp)
	iw.ack(err, []string{identity.ID})
	if err != nil {
		iw.err = err
		return false
	}
	return true
}

func writeUserIdentity(ctx context.Context, db *postgres.DB, identity map[string]any, id string, anonID string, action int, fromEvent bool, timestamp time.Time) error {

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
