//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/types"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *PostgreSQL) ResolveIdentities(ctx context.Context, identifiers, userColumns []warehouses.Column, userPrimarySources map[string]int) error {

	db, err := warehouse.connection()
	if err != nil {
		return err
	}

	// Check if the Identity Resolution or an alter schema operation is already
	// in progress; if they are, return an error. If not, «acquire a lock» on
	// other executions by inserting a row into the '_operations' table.
	err = db.Transaction(ctx, func(tx *postgres.Tx) error {
		inExecution, err := alterSchemaInProgress(ctx, tx)
		if err != nil {
			return err
		}
		if inExecution {
			return warehouses.ErrAlterSchemaInProgress
		}
		startTime, endTime, err := lastIdentityResolution(ctx, tx, warehouse.settings.Database)
		if err != nil {
			return err
		}
		if startTime != nil && endTime == nil {
			return warehouses.ErrIdentityResolutionInProgress
		}
		_, err = tx.Exec(ctx, `INSERT INTO _operations (operation, start_time, end_time) `+
			`VALUES ('IdentityResolution', (clock_timestamp() at time zone 'utc')::timestamp, NULL)`)
		if err != nil {
			return warehouses.Error(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

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
	err = db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err = tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE "_users_%d")`, postgres.QuoteIdent(newUsersName), usersVersion))
		if err != nil {
			return warehouses.Error(err)
		}
		_, err := db.Exec(ctx, `UPDATE _operations SET users_version = $1 WHERE operation = 'IdentityResolution' AND end_time IS NULL`, newUsersVersion)
		if err != nil {
			return warehouses.Error(err)
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
		sameUser.WriteString("same_user(")
		for i, ident := range identifiers {
			if i > 0 {
				sameUser.WriteByte(',')
			}
			sameUser.WriteString(`i1."`)
			sameUser.WriteString(ident.Name)
			sameUser.WriteString(`"::text,i2."`)
			sameUser.WriteString(ident.Name)
			sameUser.WriteString(`"::text`)
		}
		sameUser.WriteString(")")
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
	_, err = warehouse.db.Exec(ctx, aggregateFunction)
	if err != nil {
		return warehouses.Error(fmt.Errorf("cannot create aggregate function 'array_cat_agg': %s", err))
	}

	// Generate the SQL queries that merge the identities to obtain the users.
	var mergeUsers strings.Builder
	mergeUsers.WriteString(`INSERT INTO `)
	mergeUsers.WriteString(postgres.QuoteIdent(newUsersName))
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
	mergeUsers.WriteString(postgres.QuoteIdent(newUsersName))
	mergeUsers.WriteString(` u
		SET
			"__id__" = gen_random_uuid()
		WHERE
			"u"."__id__" IN (
				SELECT
					"u2"."__id__"
				FROM
					` + postgres.QuoteIdent(newUsersName) + ` "u2"
				GROUP BY
					"u2"."__id__"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_user }}", sameUser.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_users }}", mergeUsers.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_users_name }}", postgres.QuoteIdent(newUsersName))
	query = strings.ReplaceAll(query, "{{ new_users_version }}", strconv.Itoa(newUsersVersion))
	_, err = warehouse.db.Exec(ctx, query)
	if err != nil {
		return warehouses.Error(err)
	}

	// Call the 'do_identity_resolution' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = db.Exec(ctx, "CALL do_identity_resolution()")
	if err != nil {
		return warehouses.Error(err)
	}

	// Replace the current "users" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_users" that the view refers to
	// has changed its name.
	_, err = db.Exec(ctx, createViewQuery(newUsersName, userColumns, true))
	if err != nil {
		return warehouses.Error(err)
	}

	// Drop the 'users' table that existed before executing this Identity
	// Resolution.
	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS "_users_`+strconv.Itoa(usersVersion)+`"`)
	if err != nil {
		return warehouses.Error(err)
	}

	return nil
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
func (warehouse *PostgreSQL) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, nil, err
	}
	return lastIdentityResolution(ctx, db, warehouse.settings.Database)
}

type queryExec interface {
	QueryRow(ctx context.Context, query string, args ...any) *postgres.Row
	Exec(ctx context.Context, query string, args ...any) (*postgres.Result, error)
}

// lastIdentityResolution returns information about the last Identity
// Resolution.
//
// In particular:
//
//   - if the Identity Resolution has been started and completed, returns its
//     start time and end time;
//   - if it is in progress, returns its start time and nil for the end time;
//   - if no Identity Resolution has ever been executed, returns nil and nil.
//
// If an error occurs with the data warehouse, it returns a
// warehouses.DataWarehouseError.
func lastIdentityResolution(ctx context.Context, db queryExec, databaseName string) (startTime, endTime *time.Time, err error) {
	query := "SELECT start_time, end_time FROM _operations WHERE operation = 'IdentityResolution' ORDER BY id DESC LIMIT 1"
	err = db.QueryRow(ctx, query).Scan(&startTime, &endTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, warehouses.Error(err)
	}
	// Check the consistency of the start time and the end time.
	if endTime != nil {
		if startTime == nil {
			return nil, nil, warehouses.Error(errors.New("table '_operations' has" +
				" an Identity Resolution execution with end time but without start time"))
		}
		if startTime != nil && startTime.After(*endTime) {
			return nil, nil, warehouses.Error(errors.New("table '_operations' has" +
				" an Identity Resolution execution with start time after the end time"))
		}
	}
	// If the end time is not set, ensure that an Identity Resolution procedure
	// is actually running; otherwise it means that PostgreSQL went down while
	// the Identity Resolution was running, and therefore the execution
	// information must be updated and made consistent.
	if endTime == nil {
		var count int
		query := `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1 and query = 'CALL do_identity_resolution()'`
		err := db.QueryRow(ctx, query, databaseName).Scan(&count)
		if err != nil {
			return nil, nil, warehouses.Error(err)
		}
		switch count {
		case 0:
			// Fix the end time.
			now := time.Now().UTC()
			_, err := db.Exec(ctx, `UPDATE _operations SET end_time = $1 WHERE operation = 'IdentityResolution' AND end_time IS NULL`, now)
			if err != nil {
				return nil, nil, warehouses.Error(err)
			}
			endTime = &now
		case 1:
			// Ok, it means that there is actually an Identity Resolution in
			// progress.
		default:
			return nil, nil, warehouses.Error(fmt.Errorf("the 'pg_stat_activity' table reported a total of %d Identity Resolution procedures", count))
		}
	}
	return startTime, endTime, nil
}
