// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

//go:build krenalis_chunkfault

package datastore

import (
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralChunkFaultSnowflake requires the matching test-only driver
// overlay.
func TestBehavioralChunkFaultSnowflake(t *testing.T) {

	if testing.Short() {
		t.Skip("requires explicitly configured Snowflake and the chunk fault overlay")
	}
	dw := newJSONFilterTestWarehouse(t, "Snowflake")
	execBehavioralFixture(t, dw, `CREATE TABLE "PROFILE_DATA" ("_KPID" VARCHAR(36),"PAYLOAD" VARCHAR);
CREATE VIEW "PROFILES" AS SELECT * FROM "PROFILE_DATA"`)
	populateBehavioralChunkProfiles(t, "Snowflake", dw)
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	store.columnByProperty.user = map[string]warehouses.Column{
		"_kpid": {Name: "_kpid", Type: types.UUID()}, "payload": {Name: "payload", Type: types.String()},
	}
	fixture := &behavioralRecordFixture{store: store, warehouse: dw, at: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	ctx, observation := gosnowflake.WithChunkFault(
		gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag))
	result, err := store.QueryProfiles(ctx, nil, behavioral.Bindings{}, fixture.at,
		behavioral.Page{Limit: 1000, Properties: []string{"payload"}})
	if err != nil {
		unavailable, ok := errors.AsType[*WarehouseError](err)
		if !ok || result != nil || !errors.Is(unavailable.Err, gosnowflake.ErrChunkFault) || ctx.Err() != nil {
			t.Fatalf("expected exhausted driver failure with live caller and no result, got %T, result=%t, canceled=%t",
				err, result != nil, ctx.Err() != nil)
		}
	}
	if err == nil {
		t.Fatal("expected injected chunk failure, got success")
	}
	attempts := observation.ExpectedAttempts()
	if observation.Rows.Load() < 1 || observation.Chunks.Load() < 2 || observation.Attempts.Load() != attempts {
		t.Fatalf("expected rows before %d failed attempts across at least two chunks, got rows=%d chunks=%d attempts=%d",
			attempts, observation.Rows.Load(), observation.Chunks.Load(), observation.Attempts.Load())
	}
	t.Logf("Snowflake chunk fault exhausted rows=%d chunks=%d attempts=%d result_nil=true caller_canceled=false",
		observation.Rows.Load(), observation.Chunks.Load(), observation.Attempts.Load())

	assertBehavioralChunkProfiles(t, fixture.query(t, nil, behavioral.Page{Limit: 1000, Properties: []string{"payload"}}))

}
