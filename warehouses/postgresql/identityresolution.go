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
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *PostgreSQL) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, profilePrimarySources map[string]int) error {
	status, err := warehouse.executeOperation(ctx, opID, identityResolution)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.resolveIdentities(ctx, opID, identifiers, profileColumns, profilePrimarySources)
	bo := backoff.New(200)
	bo.SetCap(time.Second)
	for bo.Next(ctx) {
		err2 := warehouse.setOperationAsCompleted(ctx, opID, err)
		if err2 != nil {
			slog.Error("cannot set identity resolution operation as completed, retrying", "err", err2, "operationError", err)
			continue
		}
		if err != nil {
			return warehouses.NewOperationError(err)
		}
		return nil
	}
	return ctx.Err()
}

func (warehouse *PostgreSQL) resolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, primarySources map[string]int) error {

	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}

	// Determine the current version of the "meergo_profiles" table and create a copy
	// of it with the incremented version.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}
	newProfilesVersion := profilesVersion + 1
	newProfilesName := fmt.Sprintf("meergo_profiles_%d", newProfilesVersion)

	// Create a copy of the current profiles table and set its new version in
	// 'meergo_profile_schema_versions'.
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (LIKE "meergo_profiles_%d")`, quoteIdent(newProfilesName), profilesVersion))
		if err != nil {
			return fmt.Errorf("cannot create profiles table (with name %s): %s", quoteIdent(newProfilesName), err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO "meergo_profile_schema_versions" ("version", "operation", "timestamp")`+
			` VALUES ($1, $2, $3)`, newProfilesVersion, opID, time.Now().UTC())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
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
		return fmt.Errorf("cannot create aggregate function 'array_cat_agg': %s", err)
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
	mergeProfiles.WriteString(`"_identities", "_mpid", "_updated_at"`)
	mergeProfiles.WriteString(") SELECT\n")
	for _, c := range profileColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeProfiles.WriteString(`(array_cat_agg(DISTINCT ` + quoteIdent(c.Name) + ` ORDER BY ` + quoteIdent(c.Name) + `) FILTER ( WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL))`)
		} else {
			mergeProfiles.WriteByte('(')
			if s, ok := primarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeProfiles.WriteString(`ARRAY_AGG(` + quoteIdent(c.Name) + ` ORDER BY "_updated_at" DESC) FILTER (WHERE ` + quoteIdent(c.Name) + ` IS NOT NULL AND "_connection" = ` + strconv.Itoa(s) + `) || `)
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
	// Write the "_mpid" column.
	// If all MPIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the profile's MPID; otherwise,
	// if we are in a situation where a previously split profile is now merged,
	// in this case, create a new random MPID. If the identities are all new,
	// also in this case, create a new random MPID.
	mergeProfiles.WriteString(`COALESCE(
		CASE
			WHEN COUNT(DISTINCT "_mpid") FILTER ( WHERE "_mpid" IS NOT NULL ) = 1
				THEN MAX("_mpid"::text)::uuid
			ELSE gen_random_uuid()
		END,
		gen_random_uuid()
	),`)
	// Write the "_updated_at" column.
	mergeProfiles.WriteString(`MAX("_updated_at")`)
	mergeProfiles.WriteString(` FROM "meergo_identities" GROUP BY "_cluster";` + "\n")

	// If two profiles who were previously one are split, they will end up
	// having the same MPID, which is incorrect. So this query, in that
	// situation, replaces the MPID of both profiles with new random MPIDs.
	mergeProfiles.WriteString(`UPDATE `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` "u"
		SET
			"_mpid" = gen_random_uuid()
		WHERE
			"u"."_mpid" IN (
				SELECT
					"u2"."_mpid"
				FROM
					` + quoteIdent(newProfilesName) + ` "u2"
				GROUP BY
					"u2"."_mpid"
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
		return err
	}

	// Call the 'resolve_identities' stored procedure (which is declared in the
	// "identity_resolution.sql" file).
	_, err = pool.Exec(ctx, "CALL resolve_identities()")
	if err != nil {
		return err
	}

	// Replace the current "profiles" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_profiles" that the view refers
	// to has changed its name.
	_, err = pool.Exec(ctx, createViewQuery(newProfilesName, profileColumns, true))
	if err != nil {
		return err
	}

	// Drop the 'profiles' table that existed before executing this Identity
	// Resolution.
	_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS "meergo_profiles_`+strconv.Itoa(profilesVersion)+`"`)
	if err != nil {
		return err
	}

	return nil
}
