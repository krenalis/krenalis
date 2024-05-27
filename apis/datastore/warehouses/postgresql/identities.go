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

// DeleteConnectionIdentities deletes the identities of a connection.
func (warehouse *PostgreSQL) DeleteConnectionIdentities(ctx context.Context, connection int) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `DELETE FROM "_users_identities" WHERE "__connection__" = $1`, connection)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// IdentitiesWriter returns an IdentitiesWriter.
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

	// TODO(Gianluca): This implementation, instead of returning immediately
	// after buffering the user identities to be written all together, directly
	// calls the underlying data warehouse to write. This needs to be optimized
	// for bulk writing rather than writing individual users.
	err = writeUserIdentity(ctx, db, identity.Properties, iw.schema, identity.ID,
		identity.AnonymousID, identity.DisplayedProperty, iw.connection, iw.fromEvent, identity.LastChangeTime)
	iw.ack(err, []string{identity.ID})
	if err != nil {
		iw.err = err
		return false
	}
	return true
}

func writeUserIdentity(ctx context.Context, db *postgres.DB, identity map[string]any,
	schema types.Type, id, anonID, displayedProperty string, connection int, fromEvent bool, lastChangeTime time.Time) error {

	// Query the matching user identities, which can be 0 (the identity is a new
	// identity), 1 (the identity already exists and must be updated) or more
	// (the new identity requires a merging of already existing identities).
	var query string
	var args []any
	if fromEvent {
		if isAnon := id == ""; isAnon {
			query = "SELECT __identity_key__ FROM _users_identities WHERE __connection__ = $1" +
				" AND __identity_id__ = '' AND $2 = ANY(__anonymous_ids__) ORDER BY __identity_key__"
			args = []any{connection, anonID}
		} else {
			query = "SELECT __identity_key__ FROM _users_identities WHERE __connection__ = $1" +
				" AND (__identity_id__ = $2) OR (__identity_id__ = '' AND $3 = ANY(__anonymous_ids__)) ORDER BY __identity_key__"
			args = []any{connection, id, anonID}
		}
	} else { // app, file or database.
		query = "SELECT __identity_key__ FROM _users_identities WHERE __connection__ = $1" +
			" AND __identity_id__ = $2 ORDER BY __identity_key__"
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
	var newIdentityKey int

	newIdentity := make(map[string]any, len(identity)+3)
	maps.Copy(newIdentity, identity)

	warehouses.SerializeRow(newIdentity, schema)

	newIdentity["__connection__"] = connection
	newIdentity["__identity_id__"] = id
	newIdentity["__last_change_time__"] = lastChangeTime.Format(time.DateTime)
	newIdentity["__displayed_property__"] = displayedProperty
	if anonID != "" {
		newIdentity["__anonymous_ids__"] = []string{anonID}
	}

	b := strings.Builder{}
	b.WriteString("INSERT INTO _users_identities (")
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
	b.WriteString(") RETURNING __identity_key__")
	err = db.QueryRow(ctx, b.String(), values...).Scan(&newIdentityKey)
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
	idsStr.WriteString(strconv.Itoa(newIdentityKey))

	// Merge the anonymous IDS.
	b.Reset()
	b.WriteString(`UPDATE _users_identities SET __anonymous_ids__ = (
		SELECT array_agg(DISTINCT anon_ids.ids) as __anonymous_ids__
		FROM (
			SELECT unnest("__anonymous_ids__") as ids
			FROM _users_identities
			WHERE __identity_key__ IN (`)
	b.WriteString(idsStr.String())
	b.WriteString(`) AND __anonymous_ids__ IS NOT NULL
			) AS anon_ids
		) WHERE __identity_key__ = $1`)
	_, err = db.Exec(ctx, b.String(), newIdentityKey)
	if err != nil {
		return warehouses.Error(err)
	}

	// Retrieve the column names of the '_users_identities' table.
	rows, err = db.Query(ctx, `SELECT * FROM "_users_identities"`)
	if err != nil {
		return warehouses.Error(err)
	}
	tableColumns := []string{}
	for _, fd := range rows.FieldDescriptions() {
		tableColumns = append(tableColumns, fd.Name)
	}

	// TODO(Gianluca): this code is supposed to implement the specifications of
	// the Connection Identity Resolution indicated in the 'doc', but it is not
	// thoroughly tested. The goal, for now, is to make the tests pass while
	// waiting to implement the writing of the identities through the Merge.

	// Merge the other fields.
	for _, p := range tableColumns {
		if p == "__connection__" || p == "__identity_key__" || p == "__anonymous_ids__" || p == "__identity_id__" || p == "__last_change_time__" {
			continue
		}
		b.Reset()
		b.WriteString(`UPDATE "_users_identities" SET "`)
		b.WriteString(p)
		b.WriteString(`" = (SELECT "`)
		b.WriteString(p)
		b.WriteString(`" FROM _users_identities WHERE "`)
		b.WriteString(p)
		b.WriteString(`" IS NOT NULL AND __identity_key__ IN (`)
		b.WriteString(idsStr.String())
		b.WriteString(") ORDER BY __identity_key__ DESC LIMIT 1)\n")
		b.WriteString(` WHERE __identity_key__ = $1`)
		_, err = db.Exec(ctx, b.String(), newIdentityKey)
		if err != nil {
			return warehouses.Error(err)
		}
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
	b.WriteString("DELETE FROM _users_identities WHERE __identity_key__ IN (")
	b.WriteString(idsToDelete.String())
	b.WriteByte(')')
	_, err = db.Exec(ctx, b.String())
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}
