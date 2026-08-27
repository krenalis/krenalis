// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses_test

import (
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

// profileIdentityGroup describes the identity fixtures that form one profile.
type profileIdentityGroup struct {
	name                          string
	anonymous                     int
	recognized                    int
	duplicateFirstAcrossPipelines bool
}

// TestResolveIdentitiesCounts verifies that all warehouse implementations
// return profile and identity counts for an Identity Resolution and persist
// them for retries of the same operation ID.
func TestResolveIdentitiesCounts(t *testing.T) {
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(platform, func(t *testing.T) {
			dw := newCountIdentitiesWarehouse(t, platform)
			ctx := t.Context()
			profileColumns := []warehouses.Column{
				{Name: "email", Type: types.String(), Nullable: true},
			}
			if err := dw.Initialize(ctx, profileColumns); err != nil {
				t.Fatal(err)
			}

			emptyOp := uuid.NewString()
			emptyCounts := resolveAndAssertCounts(t, dw, emptyOp, profileColumns, warehouses.IdentityResolutionCounts{})

			mergeIdentityResolutionCountIdentities(t, dw, profileColumns, []profileIdentityGroup{
				{name: "anonymous-one", anonymous: 1, duplicateFirstAcrossPipelines: true},
				{name: "anonymous-two", anonymous: 2},
			})
			anonymousOp := uuid.NewString()
			anonymousCounts := resolveAndAssertCounts(t, dw, anonymousOp, profileColumns, warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Anonymous: 2},
				Identities:  warehouses.Counts{Anonymous: 3},
				Composition: warehouses.IdentityResolutionComposition{One: 1, Two: 1},
			})

			// Turn both existing profiles into recognized profiles. They retain their
			// anonymous identities and gain one recognized identity each.
			mergeIdentityResolutionCountIdentities(t, dw, profileColumns, []profileIdentityGroup{
				{name: "anonymous-one", recognized: 1},
				{name: "anonymous-two", recognized: 1},
			})
			recognizedOp := uuid.NewString()
			recognizedCounts := resolveAndAssertCounts(t, dw, recognizedOp, profileColumns, warehouses.IdentityResolutionCounts{
				Profiles:    warehouses.Counts{Recognized: 2},
				Identities:  warehouses.Counts{Anonymous: 3, Recognized: 2},
				Composition: warehouses.IdentityResolutionComposition{Two: 1, Three: 1},
			})

			// Exercise the 4-10 and 11-20 bucket boundaries and the open-ended
			// 21+ bucket.
			mergeIdentityResolutionCountIdentities(t, dw, profileColumns, []profileIdentityGroup{
				{name: "recognized-four", recognized: 4},
				{name: "anonymous-ten", anonymous: 10},
				{name: "mixed-eleven", anonymous: 2, recognized: 9},
				{name: "recognized-twenty", recognized: 20},
				{name: "recognized-twenty-one", recognized: 21},
				{name: "anonymous-twenty-five", anonymous: 25},
			})
			variedOp := uuid.NewString()
			resolveAndAssertCounts(t, dw, variedOp, profileColumns, warehouses.IdentityResolutionCounts{
				Profiles:   warehouses.Counts{Anonymous: 2, Recognized: 6},
				Identities: warehouses.Counts{Anonymous: 40, Recognized: 56},
				Composition: warehouses.IdentityResolutionComposition{
					Two: 1, Three: 1, FourToTen: 2, ElevenToTwenty: 2, MoreThanTwenty: 2,
				},
			})

			// A newly initialized warehouse starts at profile version 0, so the
			// first two operations produce versions 1 and 2. Later operations drop
			// both physical tables, but their complete persisted results remain
			// retryable.
			if _, err := dw.Count(ctx, "krenalis_profiles_1"); err == nil {
				t.Fatal("expected physical profiles table for the empty operation to be unavailable, got no error")
			}
			retryCounts, err := dw.ResolveIdentities(ctx, emptyOp, profileColumns, profileColumns, nil)
			if err != nil {
				t.Fatal(err)
			}
			if retryCounts == nil {
				t.Fatal("expected non-nil retry counts after dropping the physical table, got nil")
			}
			if diff := cmp.Diff(emptyCounts, retryCounts); diff != "" {
				t.Fatalf("unexpected retry counts after dropping the physical table (-want +got):\n%s", diff)
			}
			if _, err := dw.Count(ctx, "krenalis_profiles_2"); err == nil {
				t.Fatal("expected physical profiles table for the anonymous-only operation to be unavailable, got no error")
			}
			retryCounts, err = dw.ResolveIdentities(ctx, anonymousOp, profileColumns, profileColumns, nil)
			if err != nil {
				t.Fatal(err)
			}
			if retryCounts == nil {
				t.Fatal("expected non-nil retry counts after dropping the physical table, got nil")
			}
			if diff := cmp.Diff(anonymousCounts, retryCounts); diff != "" {
				t.Fatalf("unexpected retry counts after dropping the physical table (-want +got):\n%s", diff)
			}

			t.Run("legacy completed operation without result", func(t *testing.T) {
				setOperationResult(t, dw, recognizedOp, nil)
				counts, err := dw.ResolveIdentities(ctx, recognizedOp, profileColumns, profileColumns, nil)
				if err != nil {
					t.Fatal(err)
				}
				if counts == nil {
					t.Fatal("expected zero legacy counts, got nil")
				}
				if *counts != (warehouses.IdentityResolutionCounts{}) {
					t.Fatalf("expected zero legacy counts, got %v", *counts)
				}
			})
			t.Run("completed operation with invalid result", func(t *testing.T) {
				result, err := json.Marshal(struct {
					Counts warehouses.IdentityResolutionCounts `json:"counts"`
				}{Counts: warehouses.IdentityResolutionCounts{
					Profiles: warehouses.Counts{Anonymous: -1},
				}})
				if err != nil {
					t.Fatal(err)
				}
				setOperationResult(t, dw, recognizedOp, result)
				counts, err := dw.ResolveIdentities(ctx, recognizedOp, profileColumns, profileColumns, nil)
				if err != nil {
					if counts != nil {
						t.Fatalf("expected nil counts with an invalid persisted result, got %v", counts)
					}
					return
				}
				t.Fatalf("expected ResolveIdentities to reject an invalid persisted result, got counts %v", counts)
			})
			t.Run("completed operation with oversized result", func(t *testing.T) {
				result, err := json.Marshal(struct {
					Counts  warehouses.IdentityResolutionCounts `json:"counts"`
					Padding string                              `json:"padding"`
				}{Counts: *recognizedCounts, Padding: strings.Repeat("x", 8<<10)})
				if err != nil {
					t.Fatal(err)
				}
				setOperationResult(t, dw, recognizedOp, result)
				counts, err := dw.ResolveIdentities(ctx, recognizedOp, profileColumns, profileColumns, nil)
				if err != nil {
					if !strings.Contains(err.Error(), "exceeds the supported size") {
						t.Fatalf("expected an oversized result error, got %v", err)
					}
					if counts != nil {
						t.Fatalf("expected nil counts with an oversized persisted result, got %v", counts)
					}
					return
				}
				t.Fatalf("expected ResolveIdentities to reject an oversized persisted result, got counts %v", counts)
			})
			t.Run("non-identity-resolution operation has no result", func(t *testing.T) {
				opAlter := uuid.NewString()
				alteredProfileColumns := append([]warehouses.Column{}, profileColumns...)
				alteredProfileColumns = append(alteredProfileColumns, warehouses.Column{
					Name:     "operation_result_test",
					Type:     types.String(),
					Nullable: true,
				})
				err := dw.AlterProfileSchema(ctx, opAlter, alteredProfileColumns, []warehouses.AlterOperation{{
					Operation: warehouses.OperationAddColumn,
					Column:    "operation_result_test",
					Type:      types.String(),
				}})
				if err != nil {
					t.Fatal(err)
				}
				if result := operationResult(t, dw, opAlter); result != nil {
					t.Fatalf("expected AlterProfileSchema to store a NULL result, got %v", result)
				}
			})

			t.Run("failed operation has no result and remains failed", func(t *testing.T) {
				opFailed := uuid.NewString()
				profilesBefore, countErr := dw.Count(ctx, "profiles")
				if countErr != nil {
					t.Fatal(countErr)
				}
				badProfileColumns := append([]warehouses.Column{}, profileColumns...)
				badProfileColumns = append(badProfileColumns, warehouses.Column{
					Name:     "column_not_in_profiles_table",
					Type:     types.String(),
					Nullable: true,
				})

				_, err := dw.ResolveIdentities(ctx, opFailed, profileColumns, badProfileColumns, nil)
				if err != nil {
					if _, ok := errors.AsType[*warehouses.OperationError](err); !ok {
						t.Fatalf("expected ResolveIdentities to return *warehouses.OperationError, got %v", err)
					}
				}
				if err == nil {
					t.Fatalf("expected ResolveIdentities to return *warehouses.OperationError, got %v", err)
				}
				profilesAfter, countErr := dw.Count(ctx, "profiles")
				if countErr != nil {
					t.Fatal(countErr)
				}
				if profilesAfter != profilesBefore {
					t.Fatalf("failed Identity Resolution changed visible profiles from %d to %d", profilesBefore, profilesAfter)
				}

				// Retry with valid arguments to verify that the persisted failure is
				// returned without executing the operation again.
				_, retryErr := dw.ResolveIdentities(ctx, opFailed, profileColumns, profileColumns, nil)
				if retryErr != nil {
					if _, ok := errors.AsType[*warehouses.OperationError](retryErr); !ok {
						t.Fatalf("expected ResolveIdentities retry to return *warehouses.OperationError, got %v", retryErr)
					}
				}
				if retryErr == nil {
					t.Fatalf("expected ResolveIdentities retry to return *warehouses.OperationError, got %v", retryErr)
				}
				if retryErr.Error() != err.Error() {
					t.Fatalf("expected ResolveIdentities retry error %q, got %q", err, retryErr)
				}
				if result := operationResult(t, dw, opFailed); result != nil {
					t.Fatalf("expected failed operation to store a NULL result, got %v", result)
				}
			})

			t.Run("oversized persisted error message", func(t *testing.T) {
				oversizedMessage := strings.Repeat("é", warehouses.MaxOperationErrorBytes/len("é")+1)
				setOperationError(t, dw, variedOp, oversizedMessage)
				counts, err := dw.ResolveIdentities(ctx, variedOp, profileColumns, profileColumns, nil)
				var operationError *warehouses.OperationError
				if err != nil {
					var ok bool
					operationError, ok = errors.AsType[*warehouses.OperationError](err)
					if !ok {
						t.Fatalf("expected ResolveIdentities to return *warehouses.OperationError, got %v", err)
					}
				}
				if err == nil {
					t.Fatalf("expected ResolveIdentities to return *warehouses.OperationError, got %v", err)
				}
				if counts != nil {
					t.Fatalf("expected nil counts for a failed operation, got %v", counts)
				}
				want := "warehouse operation failed with an invalid or oversized error message"
				if operationError.Error() != want {
					t.Fatalf("expected %q, got %q", want, operationError)
				}
			})
		})
	}
}

// assertIdentityResolutionCountInvariants verifies the internal consistency of
// Identity Resolution counts returned by a warehouse.
func assertIdentityResolutionCountInvariants(t *testing.T, counts warehouses.IdentityResolutionCounts) {
	t.Helper()

	if counts.Profiles.Anonymous < 0 || counts.Profiles.Recognized < 0 {
		t.Fatalf("expected non-negative profile counts, got %v", counts.Profiles)
	}
	if counts.Identities.Anonymous < 0 || counts.Identities.Recognized < 0 {
		t.Fatalf("expected non-negative identity counts, got %v", counts.Identities)
	}

	composition := [...]int{
		counts.Composition.One,
		counts.Composition.Two,
		counts.Composition.Three,
		counts.Composition.FourToTen,
		counts.Composition.ElevenToTwenty,
		counts.Composition.MoreThanTwenty,
	}
	compositionTotal := 0
	for _, count := range composition {
		if count < 0 {
			t.Fatalf("expected non-negative composition counts, got %v", counts.Composition)
		}
		compositionTotal += count
	}

	profilesTotal := counts.Profiles.Anonymous + counts.Profiles.Recognized
	if profilesTotal != compositionTotal {
		t.Fatalf("expected profile total %d, got %d", compositionTotal, profilesTotal)
	}
	identitiesTotal := counts.Identities.Anonymous + counts.Identities.Recognized
	if identitiesTotal < profilesTotal {
		t.Fatalf("expected at least %d identities, got %d", profilesTotal, identitiesTotal)
	}
}

// assertPersistedIdentityResolutionResult verifies the persisted Identity
// Resolution result payload.
func assertPersistedIdentityResolutionResult(t *testing.T, dw warehouses.Warehouse, opID string, want warehouses.IdentityResolutionCounts) {
	t.Helper()
	result := operationResult(t, dw, opID)
	if result == nil {
		t.Fatal("expected successful Identity Resolution to store a non-NULL result, got NULL")
	}
	encoded, ok := result.(json.Value)
	if !ok {
		t.Fatalf("expected operation result type json.Value, got %T", result)
	}

	var root map[string]any
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatal(err)
	}
	if len(root) != 1 {
		t.Fatalf("expected operation result root to contain only counts, got %v", root)
	}
	if _, ok := root["counts"]; !ok {
		t.Fatalf("expected operation result root to contain counts, got %v", root)
	}
	for _, field := range []string{"profiles", "identities", "composition", "anonymous", "recognized"} {
		if _, ok := root[field]; ok {
			t.Fatalf("expected operation result root not to contain %q, got %v", field, root)
		}
	}
	countsObject, ok := root["counts"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation result counts to be an object, got %T", root["counts"])
	}
	for _, field := range []string{"profiles", "identities", "composition"} {
		if _, ok := countsObject[field]; !ok {
			t.Fatalf("expected operation result counts to contain %q, got %v", field, countsObject)
		}
	}
	if len(countsObject) != 3 {
		t.Fatalf("expected operation result counts to contain only profiles, identities, and composition, got %v", countsObject)
	}
	compositionObject, ok := countsObject["composition"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation result composition to be an object, got %T", countsObject["composition"])
	}
	for _, field := range []string{"one", "two", "three", "fourToTen", "elevenToTwenty", "moreThanTwenty"} {
		if _, ok := compositionObject[field]; !ok {
			t.Fatalf("expected operation result composition to contain %q, got %v", field, compositionObject)
		}
	}
	if len(compositionObject) != 6 {
		t.Fatalf("expected operation result composition to contain only the six buckets, got %v", compositionObject)
	}

	var persisted struct {
		Counts warehouses.IdentityResolutionCounts `json:"counts"`
	}
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, persisted.Counts); diff != "" {
		t.Fatalf("unexpected persisted Identity Resolution counts (-want +got):\n%s", diff)
	}
}

// mergeIdentityResolutionCountIdentities inserts identity fixtures grouped
// into profiles by email.
func mergeIdentityResolutionCountIdentities(t *testing.T, dw warehouses.Warehouse, profileColumns []warehouses.Column, groups []profileIdentityGroup) {
	t.Helper()
	for _, group := range groups {
		if group.anonymous < 0 || group.recognized < 0 || group.anonymous+group.recognized == 0 {
			t.Fatalf("expected non-negative counts for a non-empty test profile, got %+v", group)
		}
	}

	columns := []warehouses.Column{
		{Name: "_pipeline", Type: types.String()},
		{Name: "_is_anonymous", Type: types.Boolean()},
		{Name: "_identity_id", Type: types.String()},
		{Name: "_connection", Type: types.String()},
		{Name: "_updated_at", Type: types.DateTime()},
		{Name: "_run", Type: types.String(), Nullable: true},
	}
	columns = append(columns, profileColumns...)

	var rows []map[string]any
	for _, group := range groups {
		for _, kind := range []struct {
			anonymous bool
			count     int
			name      string
		}{
			{anonymous: true, count: group.anonymous, name: "anonymous"},
			{count: group.recognized, name: "recognized"},
		} {
			for identityIndex := range kind.count {
				row := map[string]any{
					"_pipeline":     fmt.Sprintf("%s-%s-pipeline-%d", group.name, kind.name, identityIndex),
					"_is_anonymous": kind.anonymous,
					"_identity_id":  fmt.Sprintf("%s-%s-identity-%d", group.name, kind.name, identityIndex),
					"_connection":   fmt.Sprintf("%s-%s-connection-%d", group.name, kind.name, identityIndex),
					"_updated_at":   time.Now().UTC(),
					"_run":          "identity-resolution-counts-test",
					"email":         group.name + "@example.com",
				}
				rows = append(rows, row)
				if group.duplicateFirstAcrossPipelines && identityIndex == 0 {
					duplicate := make(map[string]any, len(row))
					maps.Copy(duplicate, row)
					duplicate["_pipeline"] = row["_pipeline"].(string) + "-duplicate"
					rows = append(rows, duplicate)
				}
			}
		}
	}
	if err := dw.MergeIdentities(t.Context(), columns, rows); err != nil {
		t.Fatal(err)
	}
}

// operationResult returns the structured result stored for an operation.
func operationResult(t *testing.T, dw warehouses.Warehouse, opID string) any {
	t.Helper()
	idColumn := warehouses.Column{Name: "id", Type: types.UUID()}
	resultColumn := warehouses.Column{Name: "result", Type: types.JSON(), Nullable: true}
	rows, _, err := dw.Query(t.Context(), warehouses.RowQuery{
		Columns: []warehouses.Column{idColumn, resultColumn},
		Table:   "krenalis_system_operations",
		Where:   warehouses.NewBaseExpr(idColumn, warehouses.OpIs, opID),
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected one system operation, got none")
	}
	row := make([]any, 2)
	if err := rows.Scan(row...); err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("expected one system operation, got multiple")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return row[1]
}

// resolveAndAssertCounts resolves identities and verifies the returned and
// persisted counts, including the result returned by an immediate retry.
func resolveAndAssertCounts(t *testing.T, dw warehouses.Warehouse, opID string, profileColumns []warehouses.Column, want warehouses.IdentityResolutionCounts) *warehouses.IdentityResolutionCounts {
	t.Helper()
	counts, err := dw.ResolveIdentities(t.Context(), opID, profileColumns, profileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts == nil {
		t.Fatal("expected ResolveIdentities to return non-nil counts, got nil")
	}
	if diff := cmp.Diff(want, *counts); diff != "" {
		t.Fatalf("unexpected ResolveIdentities counts (-want +got):\n%s", diff)
	}
	assertIdentityResolutionCountInvariants(t, *counts)
	assertPersistedIdentityResolutionResult(t, dw, opID, *counts)
	retryCounts, err := dw.ResolveIdentities(t.Context(), opID, profileColumns, profileColumns, nil)
	if err != nil {
		t.Fatal(err)
	}
	if retryCounts == nil {
		t.Fatal("expected non-nil immediate retry counts, got nil")
	}
	if diff := cmp.Diff(counts, retryCounts); diff != "" {
		t.Fatalf("unexpected immediate retry counts (-want +got):\n%s", diff)
	}
	return counts
}

// setOperationError sets an operation's stored error message.
func setOperationError(t *testing.T, dw warehouses.Warehouse, opID, message string) {
	t.Helper()
	table := warehouses.Table{
		Name: "krenalis_system_operations",
		Columns: []warehouses.Column{
			{Name: "id", Type: types.UUID()},
			{Name: "error", Type: types.String()},
		},
		Keys: []string{"id"},
	}
	if err := dw.Merge(t.Context(), table, [][]any{{opID, message}}, nil); err != nil {
		t.Fatal(err)
	}
}

// setOperationResult replaces the structured result of a completed operation.
func setOperationResult(t *testing.T, dw warehouses.Warehouse, opID string, result json.Value) {
	t.Helper()
	table := warehouses.Table{
		Name: "krenalis_system_operations",
		Columns: []warehouses.Column{
			{Name: "id", Type: types.UUID()},
			{Name: "result", Type: types.JSON(), Nullable: true},
		},
		Keys: []string{"id"},
	}

	// Convert a nil json.Value to a nil interface so Merge stores SQL NULL.
	var storedResult any = result
	if result == nil {
		storedResult = nil
	}
	if err := dw.Merge(t.Context(), table, [][]any{{opID, storedResult}}, nil); err != nil {
		t.Fatal(err)
	}
}
