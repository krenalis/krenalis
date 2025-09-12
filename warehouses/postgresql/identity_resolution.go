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
	"github.com/meergo/meergo/core/backoff"
	"github.com/meergo/meergo/core/types"

	"github.com/jackc/pgx/v5"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *PostgreSQL) ResolveIdentities(ctx context.Context, opID string, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	status, err := warehouse.executeOperation(ctx, opID, identityResolution)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.resolveIdentities(ctx, opID, identifiers, userColumns, userPrimarySources)
	bo := backoff.New(200)
	bo.SetCap(time.Second)
	for bo.Next(ctx) {
		err2 := warehouse.setOperationAsCompleted(ctx, opID, err)
		if err2 != nil {
			slog.Error("cannot set identity resolution operation as completed, retrying", "err", err2, "operationError", err)
			continue
		}
		if err != nil {
			return meergo.NewOperationError(err)
		}
		return nil
	}
	return ctx.Err()
}

func (warehouse *PostgreSQL) resolveIdentities(ctx context.Context, opID string, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {

	pool, err := warehouse.connectionPool(ctx)
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

	// Create a copy of the current users table and set its new version in
	// '_user_schema_versions'.
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE "_users_%d")`, quoteIdent(newUsersName), usersVersion))
		if err != nil {
			return fmt.Errorf("cannot create users table (with name %s): %s", quoteIdent(newUsersName), err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO "_user_schema_versions" ("version", "operation", "timestamp")`+
			` VALUES ($1, $2, $3)`, newUsersVersion, opID, time.Now().UTC())
		if err != nil {
			return err
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
		return fmt.Errorf("cannot create aggregate function 'array_cat_agg': %s", err)
	}

	// Generate the SQL queries that merge the identities to obtain the users.
	var mergeUsers strings.Builder
	mergeUsers.WriteString(`INSERT INTO `)
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` (`)
	for _, c := range userColumns {
		mergeUsers.WriteString(quoteIdent(c.Name))
		mergeUsers.WriteByte(',')
	}
	mergeUsers.WriteString(`"__identities__", "__id__", "__last_change_time__"`)
	mergeUsers.WriteString(") SELECT\n")
	for _, c := range userColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeUsers.WriteString(`(array_cat_agg(DISTINCT ` + quoteIdent(c.Name) + ` ORDER BY ` + quoteIdent(c.Name) + `) FILTER ( WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL))`)
		} else {
			mergeUsers.WriteByte('(')
			if s, ok := userPrimarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeUsers.WriteString(`ARRAY_AGG(` + quoteIdent(c.Name) + ` ORDER BY "__last_change_time__" DESC) FILTER (WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL AND "__connection__" = ` + strconv.Itoa(s) + `) || `)
			}
			// Concatenate the values of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeUsers.WriteString(`ARRAY_AGG(` + quoteIdent(c.Name) + ` ORDER BY "__last_change_time__" DESC) FILTER (WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL)`)
			mergeUsers.WriteString(`)[1]`)
		}
		mergeUsers.WriteString(" AS ")
		mergeUsers.WriteString(quoteIdent(c.Name))
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
	mergeUsers.WriteString(` FROM "_user_identities" GROUP BY "__cluster__";` + "\n")

	// If two users who were previously one are split, they will end up having
	// the same GID, which is incorrect. So this query, in that situation,
	// replaces the GID of both users with new random GIDs.
	mergeUsers.WriteString(`UPDATE `)
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` "u"
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
		return err
	}

	// Call the 'resolve_identities' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = pool.Exec(ctx, "CALL resolve_identities()")
	if err != nil {
		return err
	}

	// Replace the current "users" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_users" that the view refers to
	// has changed its name.
	_, err = pool.Exec(ctx, createViewQuery(newUsersName, userColumns, true))
	if err != nil {
		return err
	}

	// Drop the 'users' table that existed before executing this Identity
	// Resolution.
	_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS "_users_`+strconv.Itoa(usersVersion)+`"`)
	if err != nil {
		return err
	}

	return nil
}
