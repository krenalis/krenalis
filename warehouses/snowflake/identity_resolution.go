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
	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/telemetry"
	"github.com/meergo/meergo/types"

	"github.com/snowflakedb/gosnowflake"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, opID string, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	status, err := warehouse.executeOperation(ctx, opID, identityResolution2)
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

func (warehouse *Snowflake) resolveIdentities(ctx context.Context, opID string, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	_, span := telemetry.TraceSpan(ctx, "Snowflake.ResolveIdentities")
	defer span.End()

	// Start an IdentityResolution operation on the data warehouse, then defer
	// its ending.
	// TODO(Gianluca): this will be removed, see https://github.com/meergo/meergo/issues/1475.
	obsoleteOpID, err := warehouse.startOperation(ctx, identityResolution)
	if err != nil {
		return err
	}
	span.AddEvent("data warehouse operation started", "operationID", opID)
	defer func() {
		// In case there are no errors, the 'endOperation' has already been
		// called in the normal execution flow, further down in the
		// ResolveIdentities method. This call is intended to handle error
		// cases, where the IdentityResolution is aborted prematurely.
		err := warehouse.endOperation(ctx, obsoleteOpID, time.Now().UTC())
		if err != nil {
			go func() {
				slog.Error("warehouses/snowflake: cannot end data warehouse operation", "id", opID, "err", err)
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
	newUsersName := fmt.Sprintf("_USERS_%d", newUsersVersion)

	// Create a copy of the current users table and set its new version in
	// '_USER_SCHEMA_VERSIONS'.
	_, span = telemetry.TraceSpan(ctx, "Switching user table", "current version", usersVersion, "next version", newUsersVersion)
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		likeTable := fmt.Sprintf(`_USERS_%d`, usersVersion)
		_, err = tx.Exec(fmt.Sprintf(`CREATE TABLE %s LIKE %s`, quoteIdent(newUsersName), quoteIdent(likeTable)))
		if err != nil {
			return fmt.Errorf("cannot create users table (with name %s) like table %s: %s", quoteIdent(newUsersName), quoteIdent(likeTable), err)
		}
		_, err = tx.Exec(`INSERT INTO "_USER_SCHEMA_VERSIONS" ("VERSION", "OPERATION", "TIMESTAMP")`+
			` VALUES (?, ?, ?)`, newUsersVersion, opID, time.Now().UTC())
		if err != nil {
			return err
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
			id := quoteIdent(ident.Name)
			sameUser.WriteString(`                WHEN "I1".`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL AND "I2".`)
			sameUser.WriteString(id)
			sameUser.WriteString(` IS NOT NULL THEN "I1".`)
			sameUser.WriteString(id)
			sameUser.WriteString(`::text = "I2".`)
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
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` (`)
	for _, c := range userColumns {
		mergeUsers.WriteString(quoteIdent(c.Name))
		mergeUsers.WriteByte(',')
	}
	mergeUsers.WriteString(`"__IDENTITIES__", "__ID__", "__LAST_CHANGE_TIME__"`)
	mergeUsers.WriteString(") SELECT\n")
	for _, c := range userColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeUsers.WriteString(`CASE WHEN ARRAY_AGG(` + quoteIdent(c.Name) +
				`) = [] THEN NULL ELSE ARRAY_SORT(ARRAY_DISTINCT(ARRAY_FLATTEN(ARRAY_AGG(` + quoteIdent(c.Name) + `)))) END`)
		} else {
			mergeUsers.WriteString(`(ARRAY_CAT(`)
			if s, ok := userPrimarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeUsers.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL AND "__CONNECTION__" = %d THEN %s END) WITHIN GROUP (ORDER BY "__LAST_CHANGE_TIME__" DESC)`, quoteIdent(c.Name), s, quoteIdent(c.Name)))
			} else {
				mergeUsers.WriteString("ARRAY_CONSTRUCT()")
			}
			mergeUsers.WriteString(", ")
			// Concatenate the values of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeUsers.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL THEN %s END) WITHIN GROUP (ORDER BY "__LAST_CHANGE_TIME__" DESC)`, quoteIdent(c.Name), quoteIdent(c.Name)))
			mergeUsers.WriteString(`))[0]`)
		}
		mergeUsers.WriteString(" AS ")
		mergeUsers.WriteString(quoteIdent(c.Name))
		mergeUsers.WriteByte(',')
	}
	// Write the "__identities__" column.
	mergeUsers.WriteString(`ARRAY_AGG(DISTINCT "__PK__"), `)
	// Write the "__id__" column.
	// If all GIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the user's GID; otherwise, if
	// we are in a situation where a previously split user is now merged, in
	// this case, create a new random GID. If the identities are all new, also
	// in this case, create a new random GID.
	mergeUsers.WriteString(`COALESCE(
		CASE
			WHEN COUNT(CASE WHEN "__GID__" IS NOT NULL THEN 1 ELSE 0 END) > 0
				THEN MAX("__GID__"::text)::varchar
			ELSE UUID_STRING()
		END,
		UUID_STRING()
	),`)
	// Write the "__last_change_time__" column.
	mergeUsers.WriteString(`MAX("__LAST_CHANGE_TIME__")`)
	mergeUsers.WriteString(` FROM "_USER_IDENTITIES" GROUP BY "__CLUSTER__"';` + "\n")

	// If two users who were previously one are split, they will end up having
	// the same GID, which is incorrect. So this query, in that situation,
	// replaces the GID of both users with new random GIDs.
	mergeUsers.WriteString(`UPDATE `)
	mergeUsers.WriteString(quoteIdent(newUsersName))
	mergeUsers.WriteString(` "U"
		SET
			"__ID__" = UUID_STRING()
		WHERE
			"U"."__ID__" IN (
				SELECT
					"U2"."__ID__"
				FROM
					` + quoteIdent(newUsersName) + ` "U2"
				GROUP BY
					"U2"."__ID__"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_user }}", sameUser.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_users }}", mergeUsers.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_users_name }}", quoteIdent(newUsersName))
	query = strings.ReplaceAll(query, "{{ new_users_version }}", strconv.Itoa(newUsersVersion))
	_, span = telemetry.TraceSpan(ctx, "Creation of support objects and stored procedures")
	ctxMulti, err := gosnowflake.WithMultiStatement(ctx, 5) // TODO(Gianluca): is there a better way?
	if err != nil {
		return err
	}
	db := warehouse.openDB()
	_, err = db.ExecContext(ctxMulti, query)
	span.End()
	if err != nil {
		return snowflake(err)
	}

	// Call the 'RESOLVE_IDENTITIES' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, span = telemetry.TraceSpan(ctx, "CALL RESOLVE_IDENTITIES()")
	_, err = db.ExecContext(ctx, "CALL RESOLVE_IDENTITIES()")
	span.End()
	if err != nil {
		return snowflake(err)
	}

	// End the IdentityResolution operation.
	err = warehouse.endOperation(ctx, obsoleteOpID, time.Now().UTC())
	if err != nil {
		return err
	}

	// Replace the current "users" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_users" that the view refers to
	// has changed its name.
	_, err = db.ExecContext(ctx, createViewQuery(newUsersName, userColumns, true))
	if err != nil {
		return snowflake(err)
	}

	// Drop the 'users' table that existed before executing this Identity
	// Resolution.
	_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS "_USERS_`+strconv.Itoa(usersVersion)+`"`)
	if err != nil {
		return snowflake(err)
	}

	return nil
}
