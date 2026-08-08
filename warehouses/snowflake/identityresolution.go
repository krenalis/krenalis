// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

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

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/snowflakedb/gosnowflake/v2"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, primarySources map[string]string) (err error) {

	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	status, err := warehouse.executeOperation(ctx, opID, identityResolution)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		if status.executionError != nil {
			return status.executionError
		}
		return nil
	}

	defer func() {
		err = warehouse.finalizeIdentityResolution(ctx, db, opID, err)
	}()

	// Determine the current version of the "krenalis_profiles" table and create
	// a copy of it with the incremented version.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}
	publishedProfilesVersion, err := warehouse.publishedProfilesVersion(ctx)
	if err != nil {
		return err
	}
	newProfilesVersion := profilesVersion + 1
	newProfilesName := fmt.Sprintf("KRENALIS_PROFILES_%d", newProfilesVersion)
	currentProfilesName := fmt.Sprintf("KRENALIS_PROFILES_%d", publishedProfilesVersion)

	// Prepare a candidate profiles version without exposing partial Identity
	// Resolution results.
	removePreviousProfilesTable := false
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		if profilesVersion != publishedProfilesVersion {
			// The latest unpublished profiles version may belong to an operation
			// that is still running. Replace it only if that operation failed.
			err = tx.QueryRowContext(ctx, `SELECT EXISTS (
				SELECT 1
				FROM "KRENALIS_PROFILE_SCHEMA_VERSIONS" "V"
				JOIN "KRENALIS_SYSTEM_OPERATIONS" "O" ON "O"."ID" = "V"."OPERATION"
				WHERE "V"."VERSION" = ?
					AND "O"."COMPLETED_AT" IS NOT NULL
					AND "O"."ERROR" <> ''
			)`, profilesVersion).Scan(&removePreviousProfilesTable)
			if err != nil {
				return snowflake(err)
			}
			if !removePreviousProfilesTable {
				return fmt.Errorf(
					"profiles version %d is unpublished but does not belong to a failed operation", profilesVersion)
			}
		}
		// Create the candidate table by copying the schema from the current profiles table.
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf(`CREATE TABLE %s LIKE "KRENALIS_PROFILES_%d"`, quoteIdent(newProfilesName), profilesVersion))
		if err != nil {
			return fmt.Errorf("cannot create profiles table (with name %s): %s", quoteIdent(newProfilesName), err)
		}
		// Add the identity count columns if they are not already present.
		_, err = tx.ExecContext(ctx, `ALTER TABLE `+quoteIdent(newProfilesName)+` ADD COLUMN IF NOT EXISTS`+
			` "_ANONYMOUS_COUNT" INTEGER NOT NULL, COLUMN IF NOT EXISTS "_RECOGNIZED_COUNT" INTEGER NOT NULL`)
		if err != nil {
			return snowflake(err)
		}
		// Link the candidate version to this operation so it can be published on
		// success or removed on failure.
		_, err = tx.ExecContext(ctx, `INSERT INTO "KRENALIS_PROFILE_SCHEMA_VERSIONS" ("VERSION", "OPERATION", "TIMESTAMP")`+
			` VALUES (?, ?, ?)`, newProfilesVersion, opID, time.Now().UTC())
		if err != nil {
			return snowflake(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Snowflake commits each DDL statement independently. Stage the view before
	// running Identity Resolution so it keeps returning the published profiles
	// and no longer depends on a table left by a previous failed operation.
	_, err = db.ExecContext(ctx, createPendingViewQuery(currentProfilesName, newProfilesName, opID, profileColumns))
	if err != nil {
		return snowflake(err)
	}
	if removePreviousProfilesTable {
		// Drop the failed candidate table after preserving its schema changes.
		_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS "KRENALIS_PROFILES_`+strconv.Itoa(profilesVersion)+`"`)
		if err != nil {
			return snowflake(err)
		}
	}

	// Generate the SQL function that determines if two identities are the same
	// profile.
	var sameProfile strings.Builder
	if len(identifiers) > 0 {
		sameProfile.WriteString("( CASE\n")
		for _, ident := range identifiers {
			id := quoteIdent(ident.Name)
			sameProfile.WriteString(`                WHEN "I1".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(` IS NOT NULL AND "I2".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(` IS NOT NULL THEN "I1".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(`::text = "I2".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(`::text`)
			sameProfile.WriteByte('\n')
		}
		sameProfile.WriteString("                ELSE false END )")
	} else {
		sameProfile.WriteString("false")
	}

	// Generate the SQL queries that merge the identities to obtain the profiles.
	var mergeProfiles strings.Builder
	mergeProfiles.WriteString(`EXECUTE IMMEDIATE 'INSERT INTO `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` (`)
	for _, c := range profileColumns {
		mergeProfiles.WriteString(quoteIdent(c.Name))
		mergeProfiles.WriteByte(',')
	}
	mergeProfiles.WriteString(`"_IDENTITIES", "_ANONYMOUS_COUNT", "_RECOGNIZED_COUNT", "_KPID", "_UPDATED_AT"`)
	mergeProfiles.WriteString(") SELECT\n")
	for _, c := range profileColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeProfiles.WriteString(`CASE WHEN ARRAY_AGG(` + quoteIdent(c.Name) +
				`) = [] THEN NULL ELSE ARRAY_SORT(ARRAY_DISTINCT(ARRAY_FLATTEN(ARRAY_AGG(` + quoteIdent(c.Name) + `)))) END`)
		} else {
			mergeProfiles.WriteString(`(ARRAY_CAT(`)
			if s, ok := primarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeProfiles.WriteString(`ARRAY_AGG(CASE WHEN `)
				mergeProfiles.WriteString(quoteIdent(c.Name))
				mergeProfiles.WriteString(` IS NOT NULL AND "_CONNECTION" = `)
				quoteStringForDynamicSQL(&mergeProfiles, s)
				mergeProfiles.WriteString(` THEN `)
				mergeProfiles.WriteString(quoteIdent(c.Name))
				mergeProfiles.WriteString(` END) WITHIN GROUP (ORDER BY "_UPDATED_AT" DESC)`)
			} else {
				mergeProfiles.WriteString("ARRAY_CONSTRUCT()")
			}
			mergeProfiles.WriteString(", ")
			// Concatenate the values of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeProfiles.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL THEN %s END) WITHIN GROUP (ORDER BY "_UPDATED_AT" DESC)`, quoteIdent(c.Name), quoteIdent(c.Name)))
			mergeProfiles.WriteString(`))[0]`)
		}
		mergeProfiles.WriteString(" AS ")
		mergeProfiles.WriteString(quoteIdent(c.Name))
		mergeProfiles.WriteByte(',')
	}
	// Write the "_identities" column.
	mergeProfiles.WriteString(`ARRAY_AGG(DISTINCT "_PK"), `)
	mergeProfiles.WriteString(`COUNT(DISTINCT
		CASE WHEN "_IS_ANONYMOUS" THEN "_CONNECTION" END,
		CASE WHEN "_IS_ANONYMOUS" THEN "_IDENTITY_ID" END
	), `)
	mergeProfiles.WriteString(`COUNT(DISTINCT
		CASE WHEN NOT "_IS_ANONYMOUS" THEN "_CONNECTION" END,
		CASE WHEN NOT "_IS_ANONYMOUS" THEN "_IDENTITY_ID" END
	), `)
	// Write the "_KPID" column.
	// If all KPIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the profile's KPID; otherwise,
	// if we are in a situation where a previously split profile is now merged,
	// in this case, create a new random KPID. If the identities are all new,
	// also in this case, create a new random KPID.
	mergeProfiles.WriteString(`COALESCE(
		CASE
			WHEN COUNT(CASE WHEN "_KPID" IS NOT NULL THEN 1 ELSE 0 END) > 0
				THEN MAX("_KPID"::text)::varchar
			ELSE UUID_STRING()
		END,
		UUID_STRING()
	),`)
	// Write the "_updated_at" column.
	mergeProfiles.WriteString(`MAX("_UPDATED_AT")`)
	mergeProfiles.WriteString(` FROM "KRENALIS_IDENTITIES" GROUP BY "_CLUSTER"';` + "\n")

	// If two profiles who were previously one are split, they will end up having
	// the same KPID, which is incorrect. So this query, in that situation,
	// replaces the KPID of both profiles with new random KPIDs.
	mergeProfiles.WriteString(`UPDATE `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` "U"
		SET
			"_KPID" = UUID_STRING()
		WHERE
			"U"."_KPID" IN (
				SELECT
					"U2"."_KPID"
				FROM
					` + quoteIdent(newProfilesName) + ` "U2"
				GROUP BY
					"U2"."_KPID"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_profile }}", sameProfile.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_profiles }}", mergeProfiles.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_profiles_name }}", quoteIdent(newProfilesName))
	query = strings.ReplaceAll(query, "{{ new_profiles_version }}", strconv.Itoa(newProfilesVersion))
	ctxMulti := gosnowflake.WithMultiStatement(ctx, 5) // TODO(Gianluca): is there a better way?
	_, err = db.ExecContext(ctxMulti, query)
	if err != nil {
		return snowflake(err)
	}

	// Call the 'RESOLVE_IDENTITIES' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = db.ExecContext(ctx, "CALL RESOLVE_IDENTITIES()")
	if err != nil {
		return snowflake(err)
	}

	// Committing the successful operation switches the staged view to the new
	// profiles without requiring another DDL statement.
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		return warehouse.setOperationAsCompleted(ctx, tx, opID, nil)
	})
	if err != nil {
		return err
	}

	// Replace the staged view with its final definition, then remove the old
	// profiles table. These are best-effort cleanups: the operation is already
	// successful and the staged view already exposes the new profiles.
	_, err2 := db.ExecContext(ctx, createViewQuery(newProfilesName, profileColumns, true))
	if err2 != nil {
		slog.Warn("cannot finalize identity resolution view", "err", warehouses.NewOperationError(snowflake(err2)))
		return nil
	}
	_, err2 = db.ExecContext(ctx, `DROP TABLE IF EXISTS `+quoteIdent(currentProfilesName))
	if err2 != nil {
		slog.Warn("cannot drop previous identity resolution table", "err", warehouses.NewOperationError(snowflake(err2)))
	}

	return nil
}

// finalizeIdentityResolution returns nil on local success. On failure, it
// attempts to mark the operation as failed and returns its persisted outcome.
func (warehouse *Snowflake) finalizeIdentityResolution(ctx context.Context, conn connection, opID string, operationErr error) error {

	if operationErr == nil {
		return nil
	}
	operationError := warehouses.NewOperationError(operationErr)

	bo := backoff.New(200)
	bo.SetCap(30 * time.Second)
	for bo.Next(ctx) {
		if err := warehouse.setOperationAsCompleted(ctx, conn, opID, operationError); err != nil {
			slog.Error("cannot mark identity resolution operation as failed, retrying",
				"err", warehouses.NewOperationError(err), "operationError", operationError)
			continue
		}
		status, err := warehouse.readOperationStatus(ctx, conn, opID)
		if err != nil {
			return err
		}
		if status == nil {
			return fmt.Errorf("identity resolution operation %s not found", opID)
		}
		if !status.alreadyCompleted {
			return fmt.Errorf("identity resolution operation %s is not completed", opID)
		}
		if status.executionError != nil {
			return status.executionError
		}
		return nil
	}

	return ctx.Err()
}

// createPendingViewQuery returns a Snowflake view definition that exposes the
// new profiles only after opID has been completed successfully.
func createPendingViewQuery(currentProfilesName, newProfilesName, opID string, profileColumns []warehouses.Column) string {

	var completed strings.Builder
	completed.WriteString(`EXISTS (
		SELECT 1 FROM "KRENALIS_SYSTEM_OPERATIONS"
		WHERE "ID" = `)
	quoteString(&completed, opID)
	completed.WriteString(`
			AND "COMPLETED_AT" IS NOT NULL
			AND "ERROR" = ''
	)`)

	var columns strings.Builder
	metaProps := []string{"_KPID", "_UPDATED_AT"}
	for i, property := range metaProps {
		if i > 0 {
			columns.WriteString(",\n")
		}
		columns.WriteString("\t")
		columns.WriteString(quoteIdent(property))
	}
	for _, column := range profileColumns {
		columns.WriteString(",\n\t")
		columns.WriteString(quoteIdent(column.Name))
	}

	var b strings.Builder
	b.WriteString(`CREATE OR REPLACE VIEW "PROFILES" AS SELECT` + "\n")
	b.WriteString(columns.String())
	b.WriteString("\nFROM ")
	b.WriteString(quoteIdent(newProfilesName))
	b.WriteString(" WHERE ")
	b.WriteString(completed.String())
	b.WriteString("\nUNION ALL\nSELECT\n")
	b.WriteString(columns.String())
	b.WriteString("\nFROM ")
	b.WriteString(quoteIdent(currentProfilesName))
	b.WriteString(" WHERE NOT ")
	b.WriteString(completed.String())

	return b.String()
}
