//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/types"

	"github.com/jackc/pgx/v5"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *PostgreSQL) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	// Start an IdentityResolution operation on the data warehouse, then defer
	// its ending.
	opID, err := warehouse.startOperation(ctx, identityResolution)
	if err != nil {
		return err
	}
	defer func() {
		// In case there are no errors, the 'endOperation' has already been
		// called in the normal execution flow, further down in the
		// ResolveIdentities method. This call is intended to handle error
		// cases, where the IdentityResolution is aborted prematurely.
		err := warehouse.endOperation(ctx, opID, time.Now().UTC())
		if err != nil {
			if err != nil {
				go func() {
					slog.Error("cannot end data warehouse operation", "id", opID, "err", err)
				}()
			}
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
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err = tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE "_users_%d")`, postgres.QuoteIdent(newUsersName), usersVersion))
		if err != nil {
			return meergo.Error(fmt.Errorf("cannot create users table (with name %s): %s", postgres.QuoteIdent(newUsersName), err))
		}
		_, err = tx.Exec(ctx, `UPDATE _operations SET users_version = $1 WHERE operation = 'IdentityResolution' AND end_time IS NULL`, newUsersVersion)
		if err != nil {
			return meergo.Error(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Generate the SQL function that determines if two identities are the same
	// user.
	var sameUser strings.Builder
	if len(identifiers) > 0 {
		sameUser.WriteString("( CASE\n")
		for _, ident := range identifiers {
			id := quoteIdent(ident.Name)
			sameUser.WriteString(`                WHEN i1.`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL AND i2.`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL THEN i1.`)
			sameUser.WriteString(id)
			sameUser.WriteString(`::text = i2.`)
			sameUser.WriteString(id)
			sameUser.WriteString(`::text`)
			sameUser.WriteByte('\n')
		}
		sameUser.WriteString("                ELSE false END )")
	} else {
		sameUser.WriteString("false")
	}

	// Drop (if exists) and create the aggregation function "array_cat_agg"
	// which is used by the identities merge query.
	const aggregateFunction = `
		DROP AGGREGATE IF EXISTS array_cat_agg(anycompatiblearray);
		CREATE AGGREGATE array_cat_agg(anycompatiblearray) (
			SFUNC=array_cat,
			STYPE=anycompatiblearray
		);`
	_, err = pool.Exec(ctx, aggregateFunction)
	if err != nil {
		return meergo.Error(fmt.Errorf("cannot create aggregate function 'array_cat_agg': %s", err))
	}

	// Generate the SQL queries that merge the identities to obtain the users.
	var mergeUsers strings.Builder
	mergeUsers.WriteString(`INSERT INTO `)
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` (`)
	for _, c := range userColumns {
		mergeUsers.WriteByte('"')
		mergeUsers.WriteString(c.Name)
		mergeUsers.WriteByte('"')
		mergeUsers.WriteByte(',')
	}
	mergeUsers.WriteString(`"__identities__", "__id__", "__last_change_time__"`)
	mergeUsers.WriteString(") SELECT\n")
	for _, c := range userColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeUsers.WriteString(`array_cat_agg(
				DISTINCT "` + c.Name + `"
				ORDER BY
					"` + c.Name + `"
			) FILTER (
				WHERE
					"` + c.Name + `" IS NOT NULL
			)`)
		} else {
			mergeUsers.WriteByte('(')
			if s, ok := userPrimarySources[c.Name]; ok {
				// If there is a user primary source S defined for this column,
				// then add to the concatenation the expression that returns the
				// values ​​for the column c.Name read from the identities
				// coming from S, excluding the NULL values.
				mergeUsers.WriteString(`(
						ARRAY_AGG(
							"` + c.Name + `"
							ORDER BY
								"__last_change_time__" DESC
						) FILTER (
							WHERE
								"` + c.Name + `" IS NOT NULL
								AND __connection__ = ` + strconv.Itoa(s) + `
						)
					) || `)
			}
			// Concatenates the values ​​of all identities for which the value
			// is not NULL, sorted by last change time. At the end is appended
			// "NULL", which handles the case where none of the identities have
			// a non-NULL value for the column, so that the indexing operation
			// that takes the first value does not fail and explicitly returns
			// "NULL" instead.
			mergeUsers.WriteString(`(
					ARRAY_AGG(
						"` + c.Name + `"
						ORDER BY
							"__last_change_time__" DESC
					) FILTER (
						WHERE
							"` + c.Name + `" IS NOT NULL
					)
				) || '{NULL}'
			)[1]`)
		}
		mergeUsers.WriteString(` AS "`)
		mergeUsers.WriteString(c.Name)
		mergeUsers.WriteByte('"')
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
			WHEN COUNT(DISTINCT "__gid__") FILTER ( WHERE "__gid__" IS NOT NULL ) = 1
				THEN MAX("__gid__"::text)::uuid
			ELSE gen_random_uuid()
		END,
		gen_random_uuid()
	),`)
	// Write the "__last_change_time__" column.
	mergeUsers.WriteString(`MAX("__last_change_time__")`)
	mergeUsers.WriteString(" FROM _user_identities GROUP BY __cluster__; ")

	// If two users who were previously one are split, they will end up having
	// the same GID, which is incorrect. So this query, in that situation,
	// replaces the GID of both users with new random GIDs.
	mergeUsers.WriteString(`UPDATE `)
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` u
		SET
			"__id__" = gen_random_uuid()
		WHERE
			"u"."__id__" IN (
				SELECT
					"u2"."__id__"
				FROM
					` + quoteIdent(newUsersName) + ` "u2"
				GROUP BY
					"u2"."__id__"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_user }}", sameUser.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_users }}", mergeUsers.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_users_name }}", quoteIdent(newUsersName))
	query = strings.ReplaceAll(query, "{{ new_users_version }}", strconv.Itoa(newUsersVersion))
	_, err = pool.Exec(ctx, query)
	if err != nil {
		return meergo.Error(err)
	}

	// Call the 'resolve_identities' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = pool.Exec(ctx, "CALL resolve_identities()")
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
	_, err = pool.Exec(ctx, createViewQuery(newUsersName, userColumns, true))
	if err != nil {
		return meergo.Error(err)
	}

	// Drop the 'users' table that existed before executing this Identity
	// Resolution.
	_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS "_users_`+strconv.Itoa(usersVersion)+`"`)
	if err != nil {
		return meergo.Error(err)
	}

	return nil
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
func (warehouse *PostgreSQL) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return nil, nil, err
	}
	err = warehouse.fixOperationsTable(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := "SELECT start_time, end_time FROM _operations WHERE " +
		"operation = 'IdentityResolution' ORDER BY id DESC LIMIT 1"
	err = pool.QueryRow(ctx, query).Scan(&startTime, &endTime)
	if err != nil && err != pgx.ErrNoRows {
		return nil, nil, meergo.Error(err)
	}
	return startTime, endTime, nil
}
