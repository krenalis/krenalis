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
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/telemetry"
	"github.com/meergo/meergo/types"
	"github.com/snowflakedb/gosnowflake"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {

	_, span := telemetry.TraceSpan(ctx, "Snowflake.ResolveIdentities")
	defer span.End()

	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return meergo.Error(err)
	}

	// Start an IdentityResolution operation on the data warehouse, then defer
	// its ending.
	opID, err := warehouse.startOperation(ctx, identityResolution)
	if err != nil {
		return err
	}
	span.AddEvent("data warehouse operation started", "operationID", opID)
	defer func() {
		// In case there are no errors, the 'endOperation' has already been
		// called in the normal execution flow, further down in the
		// ResolveIdentities method. This call is intended to handle error
		// cases, where the IdentityResolution is aborted prematurely.
		err := warehouse.endOperation(ctx, opID, time.Now().UTC())
		if err != nil {
			go func() {
				slog.Error("cannot end data warehouse operation", "id", opID, "err", err)
			}()
		}
	}()

	// Determine the current version of the "users" table and create a copy of
	// it with the incremented version.
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return err
	}
	newUsersVersion := usersVersion + 1
	newUsersName := fmt.Sprintf("_users_%d", newUsersVersion)

	// Create a copy of the current users table and set the related index in the
	// operations table.
	_, span = telemetry.TraceSpan(ctx, "Switching user table", "current version", usersVersion, "next version", newUsersVersion)
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		likeTable := fmt.Sprintf(`_users_%d`, usersVersion)
		_, err = tx.Exec(fmt.Sprintf(`CREATE TABLE %s LIKE %s`, quoteTable(newUsersName), quoteTable(likeTable)))
		if err != nil {
			return meergo.Error(fmt.Errorf("cannot create users table (with name %s) like table %s: %s", quoteTable(newUsersName), quoteTable(likeTable), err))
		}
		_, err = tx.Exec(`UPDATE "_operations" SET "users_version" = ? WHERE "operation" = 'IdentityResolution' AND "end_time" IS NULL`, newUsersVersion)
		if err != nil {
			return meergo.Error(err)
		}
		return nil
	})
	span.End()
	if err != nil {
		return err
	}

	// Generate the SQL function that determines if two identities are the same
	// user.
	var sameUser strings.Builder
	if len(identifiers) > 0 {
		sameUser.WriteString("( CASE\n")
		for _, ident := range identifiers {
			id := quoteColumn(ident.Name)
			sameUser.WriteString(`                WHEN "i1".`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL AND "i2".`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL THEN "i1".`)
			sameUser.WriteString(id)
			sameUser.WriteString(`::text = "i2".`)
			sameUser.WriteString(id)
			sameUser.WriteString(`::text`)
			sameUser.WriteByte('\n')
		}
		sameUser.WriteString("                ELSE false END )")
	} else {
		sameUser.WriteString("false")
	}

	// Generate the SQL queries that merge the identities to obtain the users.
	var mergeUsers strings.Builder
	mergeUsers.WriteString(`EXECUTE IMMEDIATE 'INSERT INTO `)
	mergeUsers.WriteString(quoteTable(newUsersName))
	mergeUsers.WriteString(` (`)
	for _, c := range userColumns {
		mergeUsers.WriteString(quoteColumn(c.Name))
		mergeUsers.WriteByte(',')
	}
	mergeUsers.WriteString(`"__identities__", "__id__", "__last_change_time__"`)
	mergeUsers.WriteString(") SELECT\n")
	for _, c := range userColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeUsers.WriteString(`CASE WHEN ARRAY_AGG(` + quoteColumn(c.Name) +
				`) = [] THEN NULL ELSE ARRAY_SORT(ARRAY_DISTINCT(ARRAY_FLATTEN(ARRAY_AGG(` + quoteColumn(c.Name) + `)))) END`)
		} else {
			mergeUsers.WriteString(`(ARRAY_CAT(`)
			if s, ok := userPrimarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeUsers.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL AND "__connection__" = %d THEN %s END) WITHIN GROUP (ORDER BY "__last_change_time__" DESC)`, quoteColumn(c.Name), s, quoteColumn(c.Name)))
			} else {
				mergeUsers.WriteString(`ARRAY_CONSTRUCT()`)
			}
			mergeUsers.WriteString(", ")
			// Concatenate the values ​​of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeUsers.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL THEN %s END) WITHIN GROUP (ORDER BY "__last_change_time__" DESC)`, quoteColumn(c.Name), quoteColumn(c.Name)))
			mergeUsers.WriteString(`))[0]`)
		}
		mergeUsers.WriteString(" AS ")
		mergeUsers.WriteString(quoteColumn(c.Name))
		mergeUsers.WriteByte(',')
	}
	// Write the "__identities__" column.
	mergeUsers.WriteString(`ARRAY_AGG(DISTINCT "__pk__"), `)
	// Write the "__id__" column.
	// If all GIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the user's GID; otherwise, if
	// we are in a situation where a previously split user is now merged, in
	// this case, create a new random GID. If the identities are all new, also
	// in this case, create a new random GID.
	mergeUsers.WriteString(`COALESCE(
		CASE
			WHEN COUNT(CASE WHEN "__gid__" IS NOT NULL THEN 1 ELSE 0 END) > 0
				THEN MAX("__gid__"::text)::varchar
			ELSE UUID_STRING()
		END,
		UUID_STRING()
	),`)
	// Write the "__last_change_time__" column.
	mergeUsers.WriteString(`MAX("__last_change_time__")`)
	mergeUsers.WriteString(` FROM "_user_identities" GROUP BY "__cluster__"';` + "\n")

	// If two users who were previously one are split, they will end up having
	// the same GID, which is incorrect. So this query, in that situation,
	// replaces the GID of both users with new random GIDs.
	mergeUsers.WriteString(`UPDATE `)
	mergeUsers.WriteString(quoteTable(newUsersName))
	mergeUsers.WriteString(` "u"
		SET
			"__id__" = UUID_STRING()
		WHERE
			"u"."__id__" IN (
				SELECT
					"u2"."__id__"
				FROM
					` + quoteTable(newUsersName) + ` "u2"
				GROUP BY
					"u2"."__id__"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_user }}", sameUser.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_users }}", mergeUsers.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_users_name }}", quoteTable(newUsersName))
	query = strings.ReplaceAll(query, "{{ new_users_version }}", strconv.Itoa(newUsersVersion))
	_, span = telemetry.TraceSpan(ctx, "Creation of support objects and stored procedures")
	ctxMulti, err := gosnowflake.WithMultiStatement(ctx, 5) // TODO(Gianluca): is there a better way?
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctxMulti, query)
	span.End()
	if err != nil {
		return meergo.Error(err)
	}

	// Call the 'resolve_identities' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, span = telemetry.TraceSpan(ctx, "CALL resolve_identities()")
	_, err = conn.ExecContext(ctx, "CALL resolve_identities()")
	span.End()
	if err != nil {
		return meergo.Error(err)
	}

	// End the IdentityResolution operation.
	err = warehouse.endOperation(ctx, opID, time.Now().UTC())
	if err != nil {
		return err
	}

	// Replace the current "users" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_users" that the view refers to
	// has changed its name.
	_, err = conn.ExecContext(ctx, createViewQuery(newUsersName, userColumns, true))
	if err != nil {
		return meergo.Error(err)
	}

	// Drop the 'users' table that existed before executing this Identity
	// Resolution.
	_, err = conn.ExecContext(ctx, `DROP TABLE IF EXISTS "_users_`+strconv.Itoa(usersVersion)+`"`)
	if err != nil {
		return meergo.Error(err)
	}

	return nil
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
	err = warehouse.fixOperationsTable(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := `SELECT "start_time", "end_time" FROM "_operations" WHERE ` +
		`"operation" = 'IdentityResolution' ORDER BY "id" DESC LIMIT 1`
	err = conn.QueryRowContext(ctx, query).Scan(&startTime, &endTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, meergo.Error(err)
	}
	return startTime, endTime, nil
}
