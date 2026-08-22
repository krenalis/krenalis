// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *PostgreSQL) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, primarySources map[string]string) (counts *warehouses.IdentityResolutionCounts, err error) {

	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return nil, err
	}

	status, err := warehouse.executeOperation(ctx, opID, identityResolution)
	if err != nil {
		return nil, err
	}
	if status.alreadyCompleted {
		if status.executionError != nil {
			return nil, status.executionError
		}
		if status.identityResolutionCounts == nil {
			return nil, fmt.Errorf("identity resolution result is unavailable for operation %s", opID)
		}
		return status.identityResolutionCounts, nil
	}

	defer func() {
		counts, err = warehouse.finalizeIdentityResolution(ctx, pool, opID, counts, err)
	}()

	// Determine the current version of the "krenalis_profiles" table and create a copy
	// of it with the incremented version.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return nil, err
	}
	publishedProfilesVersion, err := warehouse.publishedProfilesVersion(ctx)
	if err != nil {
		return nil, err
	}
	newProfilesVersion := profilesVersion + 1
	newProfilesName := fmt.Sprintf("krenalis_profiles_%d", newProfilesVersion)

	// Prepare a candidate profiles version without exposing partial Identity
	// Resolution results.
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		removePreviousProfilesTable := false
		if profilesVersion != publishedProfilesVersion {
			// The latest unpublished profiles version may belong to an operation
			// that is still running. Replace it only if that operation failed.
			err = tx.QueryRow(ctx, `SELECT EXISTS (
				SELECT 1
				FROM "krenalis_profile_schema_versions" "v"
				JOIN "krenalis_system_operations" "o" ON "o"."id" = "v"."operation"
				WHERE "v"."version" = $1
					AND "o"."completed_at" IS NOT NULL
					AND "o"."error" <> ''
			)`, profilesVersion).Scan(&removePreviousProfilesTable)
			if err != nil {
				return err
			}
			if !removePreviousProfilesTable {
				return fmt.Errorf(
					"profiles version %d is unpublished but does not belong to a failed operation", profilesVersion)
			}
		}
		// Create the candidate table by copying the schema from the current profiles table.
		_, err := tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE "krenalis_profiles_%d")`, quoteIdent(newProfilesName), profilesVersion))
		if err != nil {
			return fmt.Errorf("cannot create profiles table (with name %s): %s", quoteIdent(newProfilesName), err)
		}
		// Add the identity metric columns if they are not already present.
		_, err = tx.Exec(ctx, `ALTER TABLE `+quoteIdent(newProfilesName)+
			` ADD COLUMN IF NOT EXISTS "_anonymous_count" integer NOT NULL,`+
			` ADD COLUMN IF NOT EXISTS "_recognized_count" integer NOT NULL`)
		if err != nil {
			return err
		}
		// Link the candidate version to this operation so it can be published on
		// success or removed on failure.
		_, err = tx.Exec(ctx, `INSERT INTO "krenalis_profile_schema_versions" ("version", "operation", "timestamp")`+
			` VALUES ($1, $2, $3)`, newProfilesVersion, opID, time.Now().UTC())
		if err != nil {
			return err
		}
		if removePreviousProfilesTable {
			// Drop the failed candidate table after preserving its schema changes.
			_, err = tx.Exec(ctx, `DROP TABLE IF EXISTS "krenalis_profiles_`+strconv.Itoa(profilesVersion)+`"`)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Generate the SQL function that determines if two identities are the same
	// profile.
	var sameProfile strings.Builder
	if len(identifiers) > 0 {
		sameProfile.WriteString("( CASE\n")
		for _, ident := range identifiers {
			id := quoteIdent(ident.Name)
			sameProfile.WriteString(`                WHEN "i1".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(` IS NOT NULL AND "i2".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(` IS NOT NULL THEN "i1".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(`::text = "i2".`)
			sameProfile.WriteString(id)
			sameProfile.WriteString(`::text`)
			sameProfile.WriteByte('\n')
		}
		sameProfile.WriteString("                ELSE false END )")
	} else {
		sameProfile.WriteString("false")
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
		return nil, fmt.Errorf("cannot create aggregate function 'array_cat_agg': %s", err)
	}

	// Generate the SQL queries that merge the identities to obtain the profiles.
	var mergeProfiles strings.Builder
	mergeProfiles.WriteString(`INSERT INTO `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` (`)
	for _, c := range profileColumns {
		mergeProfiles.WriteString(quoteIdent(c.Name))
		mergeProfiles.WriteByte(',')
	}
	mergeProfiles.WriteString(`"_identities", "_anonymous_count", "_recognized_count", "_kpid", "_updated_at"`)
	mergeProfiles.WriteString(") SELECT\n")
	for _, c := range profileColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeProfiles.WriteString(`(array_cat_agg(DISTINCT ` + quoteIdent(c.Name) + ` ORDER BY ` + quoteIdent(c.Name) + `) FILTER ( WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL))`)
		} else {
			mergeProfiles.WriteByte('(')
			if s, ok := primarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeProfiles.WriteString(`ARRAY_AGG(` + quoteIdent(c.Name) + ` ORDER BY "_updated_at" DESC) FILTER (WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL AND "_connection" = `)
				quoteString(&mergeProfiles, s)
				mergeProfiles.WriteString(`) || `)
			}
			// Concatenate the values of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeProfiles.WriteString(`ARRAY_AGG(` + quoteIdent(c.Name) + ` ORDER BY "_updated_at" DESC) FILTER (WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL)`)
			mergeProfiles.WriteString(`)[1]`)
		}
		mergeProfiles.WriteString(" AS ")
		mergeProfiles.WriteString(quoteIdent(c.Name))
		mergeProfiles.WriteByte(',')
	}
	// Write the "_identities" column.
	mergeProfiles.WriteString(`ARRAY_AGG(DISTINCT "_pk"), `)
	mergeProfiles.WriteString(`COUNT(DISTINCT ("_connection", "_identity_id")) FILTER (WHERE "_is_anonymous"), `)
	mergeProfiles.WriteString(`COUNT(DISTINCT ("_connection", "_identity_id")) FILTER (WHERE NOT "_is_anonymous"), `)
	// Write the "_kpid" column.
	// If all KPIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the profile's KPID; otherwise,
	// if we are in a situation where a previously split profile is now merged,
	// in this case, create a new random KPID. If the identities are all new,
	// also in this case, create a new random KPID.
	mergeProfiles.WriteString(`COALESCE(
		CASE
			WHEN COUNT(DISTINCT "_kpid") FILTER ( WHERE "_kpid" IS NOT NULL ) = 1
				THEN MAX("_kpid"::text)::uuid
			ELSE gen_random_uuid()
		END,
		gen_random_uuid()
	),`)
	// Write the "_updated_at" column.
	mergeProfiles.WriteString(`MAX("_updated_at")`)
	mergeProfiles.WriteString(` FROM "krenalis_identities" GROUP BY "_cluster";` + "\n")

	// If two profiles who were previously one are split, they will end up
	// having the same KPID, which is incorrect. So this query, in that
	// situation, replaces the KPID of both profiles with new random KPIDs.
	mergeProfiles.WriteString(`UPDATE `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` "u"
		SET
			"_kpid" = gen_random_uuid()
		WHERE
			"u"."_kpid" IN (
				SELECT
					"u2"."_kpid"
				FROM
					` + quoteIdent(newProfilesName) + ` "u2"
				GROUP BY
					"u2"."_kpid"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_profile }}", sameProfile.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_profiles }}", mergeProfiles.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_profiles_name }}", quoteIdent(newProfilesName))
	query = strings.ReplaceAll(query, "{{ new_profiles_version }}", strconv.Itoa(newProfilesVersion))
	_, err = pool.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	// Call the 'resolve_identities' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = pool.Exec(ctx, "CALL resolve_identities()")
	if err != nil {
		return nil, err
	}

	// Count the profiles and identities produced by this Identity Resolution
	// before publishing the new profiles table.
	query = `SELECT
			COUNT(*) FILTER (WHERE "_recognized_count" = 0),
			COUNT(*) FILTER (WHERE "_recognized_count" > 0),
			COALESCE(SUM("_anonymous_count"), 0),
			COALESCE(SUM("_recognized_count"), 0),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" = 1),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" = 2),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" = 3),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" BETWEEN 4 AND 10),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" BETWEEN 11 AND 20),
			COUNT(*) FILTER (WHERE "_anonymous_count" + "_recognized_count" >= 21)
		FROM ` + quoteIdent(newProfilesName)
	counts = &warehouses.IdentityResolutionCounts{}
	err = pool.QueryRow(ctx, query).Scan(
		&counts.Profiles.Anonymous,
		&counts.Profiles.Recognized,
		&counts.Identities.Anonymous,
		&counts.Identities.Recognized,
		&counts.Composition.One,
		&counts.Composition.Two,
		&counts.Composition.Three,
		&counts.Composition.FourToTen,
		&counts.Composition.ElevenToTwenty,
		&counts.Composition.MoreThanTwenty,
	)
	if err != nil {
		return nil, err
	}
	if err := warehouses.ValidateIdentityResolutionCounts(counts); err != nil {
		return nil, err
	}

	// Serialize the counts as the structured operation result.
	result, err := json.Marshal(struct {
		Counts *warehouses.IdentityResolutionCounts `json:"counts"`
	}{Counts: counts})
	if err != nil {
		return nil, err
	}

	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {

		// Replace the current "profiles" view with a new one using the "CREATE OR REPLACE VIEW"
		// statement since the table "_profiles" that the view refers to has changed its name.
		_, err = tx.Exec(ctx, createViewQuery(newProfilesName, profileColumns, true))
		if err != nil {
			return err
		}

		// Drop the "profiles" table that existed before executing this Identity Resolution.
		_, err = tx.Exec(ctx, `DROP TABLE IF EXISTS "krenalis_profiles_`+strconv.Itoa(publishedProfilesVersion)+`"`)
		if err != nil {
			return err
		}

		return warehouse.setOperationAsCompleted(ctx, tx, opID, result, nil)
	})
	if err != nil {
		return nil, err
	}

	return counts, nil
}

// finalizeIdentityResolution returns the local result on success. On failure,
// it attempts to mark the operation as failed and returns its persisted result
// or execution error.
func (warehouse *PostgreSQL) finalizeIdentityResolution(ctx context.Context, conn connection, opID string, counts *warehouses.IdentityResolutionCounts, operationErr error) (*warehouses.IdentityResolutionCounts, error) {

	if operationErr == nil {
		return counts, nil
	}
	operationError := warehouses.NewOperationError(operationErr)

	bo := backoff.New(200)
	bo.SetCap(30 * time.Second)
	for bo.Next(ctx) {
		if err := warehouse.setOperationAsCompleted(ctx, conn, opID, nil, operationError); err != nil {
			slog.Error("cannot mark identity resolution operation as failed, retrying",
				"err", warehouses.NewOperationError(err), "operationError", operationError)
			continue
		}
		status, err := warehouse.readOperationStatus(ctx, conn, opID, identityResolution)
		if err != nil {
			return nil, err
		}
		if status == nil {
			return nil, fmt.Errorf("identity resolution operation %s not found", opID)
		}
		if !status.alreadyCompleted {
			return nil, fmt.Errorf("identity resolution operation %s is not completed", opID)
		}
		if status.executionError != nil {
			return nil, status.executionError
		}
		if status.identityResolutionCounts == nil {
			return nil, fmt.Errorf("identity resolution result is unavailable for operation %s", opID)
		}
		return status.identityResolutionCounts, nil
	}

	return nil, ctx.Err()
}
