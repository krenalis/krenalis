// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	_ "github.com/krenalis/krenalis/warehouses/postgresql"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/google/go-cmp/cmp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	countIdentitiesConnectionRecognized              = "111111111111"
	countIdentitiesConnectionMixed                   = "222222222222"
	countIdentitiesConnectionMixedPipelines          = "333333333333"
	countIdentitiesConnectionDuplicate               = "444444444444"
	countIdentitiesConnectionSingleDuplicate         = "555555555555"
	countIdentitiesConnectionWithoutProfileDuplicate = "666666666666"
	countIdentitiesConnectionDifferent               = "777777777777"
	countIdentitiesConnectionOther1                  = "888888888888"
	countIdentitiesConnectionOther2                  = "999999999999"
	countIdentitiesConnectionAnonymous               = "AAAAAAAAAAAA"
	countIdentitiesConnectionOversized               = "BBBBBBBBBBBBB"
	countIdentitiesConnectionExcess1                 = "CCCCCCCCCCCC"
	countIdentitiesConnectionExcess2                 = "DDDDDDDDDDDD"

	countIdentitiesPipelineRecognized               = "pipeline-recognized"
	countIdentitiesPipelineMixed                    = "pipeline-mixed"
	countIdentitiesPipelineMixed1                   = "pipeline-mixed-1"
	countIdentitiesPipelineMixed2                   = "pipeline-mixed-2"
	countIdentitiesPipelineDuplicate1               = "pipeline-duplicate-1"
	countIdentitiesPipelineDuplicate2               = "pipeline-duplicate-2"
	countIdentitiesPipelineSingleDuplicate          = "pipeline-single-duplicate"
	countIdentitiesPipelineWithoutProfileDuplicate1 = "pipeline-without-profile-duplicate-1"
	countIdentitiesPipelineWithoutProfileDuplicate2 = "pipeline-without-profile-duplicate-2"
	countIdentitiesPipelineDifferent1               = "pipeline-different-1"
	countIdentitiesPipelineDifferent2               = "pipeline-different-2"
	countIdentitiesPipelineOther1                   = "pipeline-other-1"
	countIdentitiesPipelineOther2                   = "pipeline-other-2"
	countIdentitiesPipelineAnonymous                = "pipeline-anonymous"
	countIdentitiesPipelineExcluded                 = "pipeline-excluded"
	countIdentitiesPipelineEmpty                    = "pipeline-empty"
	countIdentitiesPipelineOversized                = "pipeline-oversized"
	countIdentitiesPipelineExcess                   = "pipeline-excess"
)

// countIdentitiesSettingsLoader provides warehouse settings for CountIdentities
// tests.
type countIdentitiesSettingsLoader struct {
	settings json.Value
}

// Load loads warehouse settings for CountIdentities tests.
func (loader countIdentitiesSettingsLoader) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(loader.settings, dst)
}

// countIdentity describes an identity fixture for CountIdentities tests.
type countIdentity struct {
	connection  string
	pipeline    string
	id          string
	isAnonymous bool
	kpid        any
}

// TestCountIdentitiesNoPipelines verifies that no warehouse connection is
// needed when there are no selected pipelines.
func TestCountIdentitiesNoPipelines(t *testing.T) {
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(platform, func(t *testing.T) {
			dw := warehouses.Registered(platform).New(countIdentitiesSettingsLoader{})
			t.Cleanup(func() {
				if err := dw.Close(); err != nil {
					t.Error(err)
				}
			})

			counts, err := dw.CountIdentities(t.Context(), []string{})
			if err != nil {
				t.Fatal(err)
			}
			want := &warehouses.IdentityCounts{
				Anonymous:      map[string]int{},
				Recognized:     map[string]int{},
				WithoutProfile: map[string]int{},
			}
			if diff := cmp.Diff(want, counts); diff != "" {
				t.Errorf("expected identity counts %v, got %v (-want +got):\n%s", want, counts, diff)
			}
		})
	}
}

// TestCountIdentities verifies that all warehouse implementations apply the
// same counting semantics and reject unsafe results from the requested
// pipelines.
func TestCountIdentities(t *testing.T) {
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(platform, func(t *testing.T) {
			dw := newCountIdentitiesWarehouse(t, platform)
			ctx := t.Context()
			if err := dw.Initialize(ctx, nil); err != nil {
				t.Fatal(err)
			}

			identities := []countIdentity{
				{connection: countIdentitiesConnectionRecognized, pipeline: countIdentitiesPipelineRecognized, id: "recognized-1"},
				{connection: countIdentitiesConnectionRecognized, pipeline: countIdentitiesPipelineRecognized, id: "recognized-2", kpid: "00000000-0000-0000-0000-000000000001"},
				{connection: countIdentitiesConnectionMixed, pipeline: countIdentitiesPipelineMixed, id: "same-id", isAnonymous: true},
				{connection: countIdentitiesConnectionMixed, pipeline: countIdentitiesPipelineMixed, id: "same-id"},
				{connection: countIdentitiesConnectionMixedPipelines, pipeline: countIdentitiesPipelineMixed1, id: "same-id", isAnonymous: true},
				{connection: countIdentitiesConnectionMixedPipelines, pipeline: countIdentitiesPipelineMixed2, id: "same-id"},
				{connection: countIdentitiesConnectionDuplicate, pipeline: countIdentitiesPipelineDuplicate1, id: "duplicated"},
				{connection: countIdentitiesConnectionDuplicate, pipeline: countIdentitiesPipelineDuplicate2, id: "duplicated", kpid: "00000000-0000-0000-0000-000000000002"},
				{connection: countIdentitiesConnectionSingleDuplicate, pipeline: countIdentitiesPipelineSingleDuplicate, id: "assigned-duplicate"},
				{connection: countIdentitiesConnectionSingleDuplicate, pipeline: countIdentitiesPipelineSingleDuplicate, id: "assigned-duplicate", kpid: "00000000-0000-0000-0000-000000000004"},
				{connection: countIdentitiesConnectionSingleDuplicate, pipeline: countIdentitiesPipelineSingleDuplicate, id: "without-profile-duplicate"},
				{connection: countIdentitiesConnectionSingleDuplicate, pipeline: countIdentitiesPipelineSingleDuplicate, id: "without-profile-duplicate"},
				{connection: countIdentitiesConnectionWithoutProfileDuplicate, pipeline: countIdentitiesPipelineWithoutProfileDuplicate1, id: "duplicated"},
				{connection: countIdentitiesConnectionWithoutProfileDuplicate, pipeline: countIdentitiesPipelineWithoutProfileDuplicate2, id: "duplicated"},
				{connection: countIdentitiesConnectionDifferent, pipeline: countIdentitiesPipelineDifferent1, id: "different-1"},
				{connection: countIdentitiesConnectionDifferent, pipeline: countIdentitiesPipelineDifferent2, id: "different-2"},
				{connection: countIdentitiesConnectionOther1, pipeline: countIdentitiesPipelineOther1, id: "same-across-connections"},
				{connection: countIdentitiesConnectionOther2, pipeline: countIdentitiesPipelineOther2, id: "same-across-connections"},
				{connection: countIdentitiesConnectionAnonymous, pipeline: countIdentitiesPipelineAnonymous, id: "anonymous-only", isAnonymous: true, kpid: "00000000-0000-0000-0000-000000000003"},
				{connection: countIdentitiesConnectionRecognized, pipeline: countIdentitiesPipelineExcluded, id: "excluded"},
				{connection: countIdentitiesConnectionOversized, pipeline: countIdentitiesPipelineOversized, id: "oversized"},
				{connection: countIdentitiesConnectionExcess1, pipeline: countIdentitiesPipelineExcess, id: "excess-1"},
				{connection: countIdentitiesConnectionExcess2, pipeline: countIdentitiesPipelineExcess, id: "excess-2"},
			}
			columns := []warehouses.Column{
				{Name: "_pipeline", Type: types.String()},
				{Name: "_is_anonymous", Type: types.Boolean()},
				{Name: "_identity_id", Type: types.String()},
				{Name: "_connection", Type: types.String()},
				{Name: "_kpid", Type: types.UUID(), Nullable: true},
				{Name: "_updated_at", Type: types.DateTime()},
				{Name: "_run", Type: types.String(), Nullable: true},
			}
			rows := make([]map[string]any, len(identities))
			for i, identity := range identities {
				rows[i] = map[string]any{
					"_pipeline":     identity.pipeline,
					"_is_anonymous": identity.isAnonymous,
					"_identity_id":  identity.id,
					"_connection":   identity.connection,
					"_kpid":         identity.kpid,
					"_updated_at":   time.Now().UTC(),
					"_run":          "test-run",
				}
			}
			if err := dw.MergeIdentities(ctx, columns, rows); err != nil {
				t.Fatal(err)
			}

			tests := []struct {
				name      string
				pipelines []string
				want      *warehouses.IdentityCounts
				wantErr   bool
			}{
				{
					name: "no pipelines",
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{},
						WithoutProfile: map[string]int{},
					},
				},
				{
					name:      "single pipeline with recognized identities",
					pipelines: []string{countIdentitiesPipelineRecognized},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{countIdentitiesConnectionRecognized: 2},
						WithoutProfile: map[string]int{countIdentitiesConnectionRecognized: 1},
					},
				},
				{
					name:      "single pipeline with anonymous and recognized identities",
					pipelines: []string{countIdentitiesPipelineMixed},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{countIdentitiesConnectionMixed: 1},
						Recognized:     map[string]int{countIdentitiesConnectionMixed: 1},
						WithoutProfile: map[string]int{countIdentitiesConnectionMixed: 2},
					},
				},
				{
					name:      "same identity ID as anonymous and recognized in multiple pipelines",
					pipelines: []string{countIdentitiesPipelineMixed1, countIdentitiesPipelineMixed2},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{countIdentitiesConnectionMixedPipelines: 1},
						Recognized:     map[string]int{countIdentitiesConnectionMixedPipelines: 1},
						WithoutProfile: map[string]int{countIdentitiesConnectionMixedPipelines: 2},
					},
				},
				{
					name:      "duplicate identity in multiple pipelines is counted once",
					pipelines: []string{countIdentitiesPipelineDuplicate1, countIdentitiesPipelineDuplicate2},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{countIdentitiesConnectionDuplicate: 1},
						WithoutProfile: map[string]int{},
					},
				},
				{
					name:      "physical duplicate rows in a single pipeline are counted separately",
					pipelines: []string{countIdentitiesPipelineSingleDuplicate},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{countIdentitiesConnectionSingleDuplicate: 4},
						WithoutProfile: map[string]int{countIdentitiesConnectionSingleDuplicate: 1},
					},
				},
				{
					name:      "identity without a profile duplicated in multiple pipelines",
					pipelines: []string{countIdentitiesPipelineWithoutProfileDuplicate1, countIdentitiesPipelineWithoutProfileDuplicate2},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{countIdentitiesConnectionWithoutProfileDuplicate: 1},
						WithoutProfile: map[string]int{countIdentitiesConnectionWithoutProfileDuplicate: 1},
					},
				},
				{
					name:      "different identities in multiple pipelines",
					pipelines: []string{countIdentitiesPipelineDifferent1, countIdentitiesPipelineDifferent2},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{countIdentitiesConnectionDifferent: 2},
						WithoutProfile: map[string]int{countIdentitiesConnectionDifferent: 2},
					},
				},
				{
					name:      "same identity ID in different connections",
					pipelines: []string{countIdentitiesPipelineOther1, countIdentitiesPipelineOther2},
					want: &warehouses.IdentityCounts{
						Anonymous: map[string]int{},
						Recognized: map[string]int{
							countIdentitiesConnectionOther1: 1,
							countIdentitiesConnectionOther2: 1,
						},
						WithoutProfile: map[string]int{
							countIdentitiesConnectionOther1: 1,
							countIdentitiesConnectionOther2: 1,
						},
					},
				},
				{
					name:      "pipeline without rows",
					pipelines: []string{countIdentitiesPipelineEmpty},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{},
						Recognized:     map[string]int{},
						WithoutProfile: map[string]int{},
					},
				},
				{
					name:      "sparse maps",
					pipelines: []string{countIdentitiesPipelineAnonymous},
					want: &warehouses.IdentityCounts{
						Anonymous:      map[string]int{countIdentitiesConnectionAnonymous: 1},
						Recognized:     map[string]int{},
						WithoutProfile: map[string]int{},
					},
				},
				{
					name: "multiple connections and excluded pipeline",
					pipelines: []string{
						countIdentitiesPipelineRecognized,
						countIdentitiesPipelineMixed,
						countIdentitiesPipelineMixed1,
						countIdentitiesPipelineMixed2,
						countIdentitiesPipelineDuplicate1,
						countIdentitiesPipelineDuplicate2,
						countIdentitiesPipelineWithoutProfileDuplicate1,
						countIdentitiesPipelineWithoutProfileDuplicate2,
						countIdentitiesPipelineDifferent1,
						countIdentitiesPipelineDifferent2,
						countIdentitiesPipelineOther1,
						countIdentitiesPipelineOther2,
						countIdentitiesPipelineAnonymous,
						countIdentitiesPipelineEmpty,
					},
					want: &warehouses.IdentityCounts{
						Anonymous: map[string]int{
							countIdentitiesConnectionMixed:          1,
							countIdentitiesConnectionMixedPipelines: 1,
							countIdentitiesConnectionAnonymous:      1,
						},
						Recognized: map[string]int{
							countIdentitiesConnectionRecognized:              2,
							countIdentitiesConnectionMixed:                   1,
							countIdentitiesConnectionMixedPipelines:          1,
							countIdentitiesConnectionDuplicate:               1,
							countIdentitiesConnectionDifferent:               2,
							countIdentitiesConnectionOther1:                  1,
							countIdentitiesConnectionOther2:                  1,
							countIdentitiesConnectionWithoutProfileDuplicate: 1,
						},
						WithoutProfile: map[string]int{
							countIdentitiesConnectionRecognized:              1,
							countIdentitiesConnectionMixed:                   2,
							countIdentitiesConnectionMixedPipelines:          2,
							countIdentitiesConnectionWithoutProfileDuplicate: 1,
							countIdentitiesConnectionDifferent:               2,
							countIdentitiesConnectionOther1:                  1,
							countIdentitiesConnectionOther2:                  1,
						},
					},
				},
				{
					name:      "oversized connection identifier",
					pipelines: []string{countIdentitiesPipelineOversized},
					wantErr:   true,
				},
				{
					name:      "more connections than pipelines",
					pipelines: []string{countIdentitiesPipelineExcess},
					wantErr:   true,
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					counts, err := dw.CountIdentities(ctx, test.pipelines)
					if test.wantErr {
						if err == nil {
							t.Fatal("expected CountIdentities to return an error, got nil")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff(test.want, counts); diff != "" {
						t.Errorf("unexpected identity counts (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

// newCountIdentitiesWarehouse creates a warehouse for CountIdentities tests.
func newCountIdentitiesWarehouse(t *testing.T, platform string) warehouses.Warehouse {
	t.Helper()

	var settings json.Value
	switch platform {
	case "PostgreSQL":
		ctx := t.Context()
		container, err := postgres.Run(ctx,
			testimages.PostgreSQL,
			postgres.WithDatabase("krenalis"),
			postgres.WithUsername("krenalis"),
			postgres.WithPassword("krenalis"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := testcontainers.TerminateContainer(container); err != nil {
				t.Error(err)
			}
		})
		host, err := container.Host(ctx)
		if err != nil {
			t.Fatal(err)
		}
		port, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			t.Fatal(err)
		}
		settings, err = json.Marshal(map[string]any{
			"host":     host,
			"port":     port.Num(),
			"username": "krenalis",
			"password": "krenalis",
			"database": "krenalis",
			"schema":   "public",
		})
		if err != nil {
			t.Fatal(err)
		}
	case "Snowflake":
		if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
			t.Skip()
		}
		testEnv, err := snowflaketester.CreateTestEnvironment()
		if err != nil {
			t.Fatalf("cannot create Snowflake test environment: %s", err)
		}
		t.Cleanup(func() {
			if err := testEnv.Teardown(); err != nil {
				t.Logf("cannot teardown Snowflake test environment: %s", err)
			}
		})
		settings = testEnv.Settings().JSON()
	default:
		panic("unsupported warehouse platform " + platform)
	}

	dw := warehouses.Registered(platform).New(countIdentitiesSettingsLoader{settings: settings})
	t.Cleanup(func() {
		if err := dw.Close(); err != nil {
			t.Error(err)
		}
	})
	return dw
}
