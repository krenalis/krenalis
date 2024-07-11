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
	"strconv"
	"strings"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// RunIdentityResolution runs the Identity Resolution.
func (warehouse *PostgreSQL) RunIdentityResolution(ctx context.Context, identifiers, userColumns []warehouses.Column, userPrimarySources map[string]int) error {

	db, err := warehouse.connection()
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
	// which is used by the merge query.
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
	mergeUsers.WriteString(`TRUNCATE _users; INSERT INTO _users (`)
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
	mergeUsers.WriteString(`UPDATE "_users" u
		SET
			"__id__" = gen_random_uuid()
		WHERE
			"u"."__id__" IN (
				SELECT
					"u2"."__id__"
				FROM
					"_users" "u2"
				GROUP BY
					"u2"."__id__"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_user }}", sameUser.String(), 1)
	query = strings.Replace(query, "{{ merge_users }}", mergeUsers.String(), 1)
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

	return nil
}
