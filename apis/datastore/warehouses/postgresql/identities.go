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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/types"

	"golang.org/x/exp/maps"
)

// IdentitiesWriter returns an IdentitiesWriter for writing user identities with
// the given schema, relative to the connection, on the data warehouse.
// fromEvent indicates if the user identities are imported from an event or not.
// ack is the ack function (see the documentation of IdentitiesWriter for more
// details about it).
// If the schema specified is not conform to the schema of the table
// 'users_identities' in the data warehouse, calls to the method 'Write' of the
// returned 'IdentitiesWriter' return a *SchemaError error.
func (warehouse *PostgreSQL) IdentitiesWriter(ctx context.Context, schema types.Type, connection int, fromEvent bool, ack warehouses.IdentitiesAckFunc) warehouses.IdentitiesWriter {
	if ack == nil {
		panic("ack function is missing")
	}
	return &identitiesWriter{
		warehouse:  warehouse,
		schema:     schema,
		connection: connection,
		fromEvent:  fromEvent,
		ack:        ack,
	}
}

type identitiesWriter struct {
	warehouse  *PostgreSQL
	schema     types.Type
	connection int
	fromEvent  bool
	ack        warehouses.IdentitiesAckFunc
	err        error
	closed     bool
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

	// If the IdentitiesWriter has a schema, then check the conformity of the
	// incoming identity schema with the schema of the 'users_identities' table
	// on the data warehouse.
	//
	// TODO(Gianluca): maybe it's not necessary to do this check for every
	// identity; see the issue https://github.com/open2b/chichi/issues/627.
	if iw.schema.Valid() {
		ti, err := iw.warehouse.tableInfo(ctx, "users_identities", false)
		if err != nil {
			iw.err = warehouses.Error(err)
			return false
		}
		err = warehouses.CheckConformity("", iw.schema, ti.schema)
		if err != nil {
			iw.err = err
			return false
		}
	}

	// TODO(Gianluca): This implementation, instead of returning immediately
	// after buffering the user identities to be written all together, directly
	// calls the underlying data warehouse to write. This needs to be optimized
	// for bulk writing rather than writing individual users.
	// See the issue https://github.com/open2b/chichi/issues/627.
	err = writeUserIdentity(ctx, db, identity.Properties, iw.schema, identity.ID,
		identity.AnonymousID, identity.DisplayedID, iw.connection, iw.fromEvent, identity.UpdatedAt)
	iw.ack(err, []string{identity.ID})
	if err != nil {
		iw.err = err
		return false
	}
	return true
}

func writeUserIdentity(ctx context.Context, db *postgres.DB, identity map[string]any,
	schema types.Type, id, anonID, displayedID string, connection int, fromEvent bool, updatedAt time.Time) error {

	// Query the matching user identities, which can be 0 (the identity is a new
	// identity), 1 (the identity already exists and must be updated) or more
	// (the new identity requires a merging of already existing identities).
	var query string
	var args []any
	if fromEvent {
		if isAnon := id == ""; isAnon {
			query = "SELECT _identity_id FROM users_identities WHERE _connection = $1" +
				" AND _external_id = '' AND $2 = ANY(_anonymous_ids) ORDER BY _updated_at, _identity_id"
			args = []any{connection, anonID}
		} else {
			query = "SELECT _identity_id FROM users_identities WHERE _connection = $1" +
				" AND (_external_id = $2) OR (_external_id = '' AND $3 = ANY(_anonymous_ids)) ORDER BY _updated_at, _identity_id"
			args = []any{connection, id, anonID}
		}
	} else { // app, file or database.
		query = "SELECT _identity_id FROM users_identities WHERE _connection = $1" +
			" AND _external_id = $2 ORDER BY _updated_at, _identity_id"
		args = []any{connection, id}
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
	if err := rows.Err(); err != nil {
		return warehouses.Error(err)
	}

	// Create the new identity.
	var newIdentityID int

	newIdentity := make(map[string]any, len(identity)+3)
	maps.Copy(newIdentity, identity)

	warehouses.SerializeRow(newIdentity, schema)

	newIdentity["_connection"] = connection
	newIdentity["_external_id"] = id
	newIdentity["_updated_at"] = updatedAt.Format(time.DateTime)
	newIdentity["_displayed_id"] = displayedID
	if anonID != "" {
		newIdentity["_anonymous_ids"] = []string{anonID}
	}

	b := strings.Builder{}
	b.WriteString("INSERT INTO users_identities (")
	properties := maps.Keys(newIdentity)
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
		values[i] = newIdentity[name]
	}
	b.WriteString(") RETURNING _identity_id")
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
	b.WriteString(`UPDATE users_identities SET _anonymous_ids = (
		SELECT array_agg(DISTINCT anon_ids.ids) as _anonymous_ids
		FROM (
			SELECT unnest("_anonymous_ids") as ids
			FROM users_identities
			WHERE _identity_id IN (`)
	b.WriteString(idsStr.String())
	b.WriteString(`) AND _anonymous_ids IS NOT NULL
			) AS anon_ids
		) WHERE _identity_id = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityID)
	if err != nil {
		return warehouses.Error(err)
	}

	// Merge the other fields.
	b.Reset()
	b.WriteString("UPDATE users_identities SET ")
	comma := false
	for _, p := range properties {
		if p == "_connection" || p == "_anonymous_ids" || p == "_external_id" || p == "_updated_at" {
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
		b.WriteString(`" IS NOT NULL AND _identity_id IN (`)
		b.WriteString(idsStr.String())
		b.WriteString(") ORDER BY _updated_at DESC, _identity_id DESC LIMIT 1)\n")
		comma = true
	}
	b.WriteString(` WHERE _identity_id = $1`)
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
	b.WriteString("DELETE FROM users_identities WHERE _identity_id IN (")
	b.WriteString(idsToDelete.String())
	b.WriteByte(')')
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}
