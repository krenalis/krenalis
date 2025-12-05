// Copyright 2025 Open2b. All rights reserved.
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

	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/snowflakedb/gosnowflake"
)

//go:embed identity_resolution.sql
var identityResolutionQueries string

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, primarySources map[string]int) error {
	status, err := warehouse.executeOperation(ctx, opID, identityResolution)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.resolveIdentities(ctx, opID, identifiers, profileColumns, primarySources)
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

func (warehouse *Snowflake) resolveIdentities(ctx context.Context, opID string, identifiers, profileColumns []warehouses.Column, profilePrimarySources map[string]int) error {

	// Determine the current version of the "meergo_profiles" table and create a copy
	// of it with the incremented version.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}
	newProfilesVersion := profilesVersion + 1
	newProfilesName := fmt.Sprintf("MEERGO_PROFILES_%d", newProfilesVersion)

	// Create a copy of the current profiles table and set its new version in
	// '_MEERGO_PROFILE_SCHEMA_VERSIONS'.
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		likeTable := fmt.Sprintf(`MEERGO_PROFILES_%d`, profilesVersion)
		_, err = tx.Exec(fmt.Sprintf(`CREATE TABLE %s LIKE %s`, quoteIdent(newProfilesName), quoteIdent(likeTable)))
		if err != nil {
			return fmt.Errorf("cannot create profiles table (with name %s) like table %s: %s", quoteIdent(newProfilesName), quoteIdent(likeTable), err)
		}
		_, err = tx.Exec(`INSERT INTO "_MEERGO_PROFILE_SCHEMA_VERSIONS" ("VERSION", "OPERATION", "TIMESTAMP")`+
			` VALUES (?, ?, ?)`, newProfilesVersion, opID, time.Now().UTC())
		if err != nil {
			return snowflake(err)
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
	mergeProfiles.WriteString(`"_IDENTITIES", "_MPID", "_LAST_CHANGE_TIME"`)
	mergeProfiles.WriteString(") SELECT\n")
	for _, c := range profileColumns {
		if c.Type.Kind() == types.ArrayKind {
			mergeProfiles.WriteString(`CASE WHEN ARRAY_AGG(` + quoteIdent(c.Name) +
				`) = [] THEN NULL ELSE ARRAY_SORT(ARRAY_DISTINCT(ARRAY_FLATTEN(ARRAY_AGG(` + quoteIdent(c.Name) + `)))) END`)
		} else {
			mergeProfiles.WriteString(`(ARRAY_CAT(`)
			if s, ok := profilePrimarySources[c.Name]; ok {
				// In the case of primary sources, list these values first,
				// sorted by last change time, excluding those that are NULL.
				mergeProfiles.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL AND "_CONNECTION" = %d THEN %s END) WITHIN GROUP (ORDER BY "_LAST_CHANGE_TIME" DESC)`, quoteIdent(c.Name), s, quoteIdent(c.Name)))
			} else {
				mergeProfiles.WriteString("ARRAY_CONSTRUCT()")
			}
			mergeProfiles.WriteString(", ")
			// Concatenate the values of all identities for which the value is
			// not NULL, sorted by last change time.
			mergeProfiles.WriteString(fmt.Sprintf(`ARRAY_AGG(CASE WHEN %s IS NOT NULL THEN %s END) WITHIN GROUP (ORDER BY "_LAST_CHANGE_TIME" DESC)`, quoteIdent(c.Name), quoteIdent(c.Name)))
			mergeProfiles.WriteString(`))[0]`)
		}
		mergeProfiles.WriteString(" AS ")
		mergeProfiles.WriteString(quoteIdent(c.Name))
		mergeProfiles.WriteByte(',')
	}
	// Write the "_identities" column.
	mergeProfiles.WriteString(`ARRAY_AGG(DISTINCT "_PK"), `)
	// Write the "_mpid" column.
	// If all MPIDs are the same - ignoring the NULL ones, which refer to new
	// identities - then take the common value as the profile's MPID; otherwise,
	// if we are in a situation where a previously split profile is now merged,
	// in this case, create a new random MPID. If the identities are all new,
	// also in this case, create a new random MPID.
	mergeProfiles.WriteString(`COALESCE(
		CASE
			WHEN COUNT(CASE WHEN "_mpid" IS NOT NULL THEN 1 ELSE 0 END) > 0
				THEN MAX("_mpid"::text)::varchar
			ELSE UUID_STRING()
		END,
		UUID_STRING()
	),`)
	// Write the "_last_change_time" column.
	mergeProfiles.WriteString(`MAX("_LAST_CHANGE_TIME")`)
	mergeProfiles.WriteString(` FROM "_IDENTITIES" GROUP BY "_CLUSTER"';` + "\n")

	// If two profiles who were previously one are split, they will end up having
	// the same MPID, which is incorrect. So this query, in that situation,
	// replaces the MPID of both profiles with new random MPIDs.
	mergeProfiles.WriteString(`UPDATE `)
	mergeProfiles.WriteString(quoteIdent(newProfilesName))
	mergeProfiles.WriteString(` "U"
		SET
			"_MPID" = UUID_STRING()
		WHERE
			"U"."_MPID" IN (
				SELECT
					"U2"."_MPID"
				FROM
					` + quoteIdent(newProfilesName) + ` "U2"
				GROUP BY
					"U2"."_MPID"
				HAVING
					COUNT(*) > 1
	)`)

	// Replace the placeholders in the Identity Resolution queries and run them.
	query := strings.Replace(identityResolutionQueries, "{{ same_profile }}", sameProfile.String(), 1)
	query = strings.Replace(query, "{{ merge_identities_in_profiles }}", mergeProfiles.String(), 1)
	query = strings.ReplaceAll(query, "{{ new_profiles_name }}", quoteIdent(newProfilesName))
	query = strings.ReplaceAll(query, "{{ new_profiles_version }}", strconv.Itoa(newProfilesVersion))
	ctxMulti, err := gosnowflake.WithMultiStatement(ctx, 5) // TODO(Gianluca): is there a better way?
	if err != nil {
		return snowflake(err)
	}
	db := warehouse.openDB()
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

	// Replace the current "profiles" view with a new one using the "CREATE OR
	// REPLACE VIEW" statement since the table "_profiles" that the view refers to
	// has changed its name.
	_, err = db.ExecContext(ctx, createViewQuery(newProfilesName, profileColumns, true))
	if err != nil {
		return snowflake(err)
	}

	// Drop the 'profiles' table that existed before executing this Identity
	// Resolution.
	_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS "MEERGO_PROFILES_`+strconv.Itoa(profilesVersion)+`"`)
	if err != nil {
		return snowflake(err)
	}

	return nil
}
