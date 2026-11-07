// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
)

// TestUnifiedPagesPostgreSQL checks page/screen identities without a PK.
func TestUnifiedPagesPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles VALUES ('00000000-0000-0000-0000-000000000001'),('00000000-0000-0000-0000-000000000002');
CREATE TABLE krenalis_events (kpid uuid,message_id text,event text,name text,type text NOT NULL,
 timestamp timestamp(3) NOT NULL,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, typ := range []string{"page", "screen"} {
		for _, test := range []struct {
			name, rows string
			conflict   bool
			truth      unified.Truth
			count      int64
		}{
			{"same copies", `('same','Pricing'),('same','Pricing')`, false, unified.Match, 1},
			{"distinct identities", `('a','Pricing'),('b','Pricing')`, false, unified.NoMatch, 2},
			{"unnamed", `('null',NULL),('empty',''),('lower','pricing')`, false, unified.NoMatch, 0},
			{"selection conflict", `('same','Pricing'),('same',NULL)`, true, 0, 0},
		} {
			t.Run(typ+"/"+test.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events
 (kpid,message_id,event,name,type,timestamp,properties)
 SELECT '00000000-0000-0000-0000-000000000001',id,NULL,name,'`+typ+`','2026-01-15','{}'
 FROM (VALUES `+test.rows+`) v(id,name)`)
				root := &unified.Rule{Count: &unified.Count{Scope: unified.Scope{Type: typ, Event: "Pricing",
					Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
					Test: unified.CountTest{Operator: unified.Equal, Value: 1}}}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					if !test.conflict || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected source conflict=%v without partial results, got %+v, %v",
							test.conflict, result, err)
					}
				}
				if err == nil {
					if test.conflict {
						t.Fatal("expected source-contract failure, got success")
					}
					assertUnifiedTruths(t, result, []unified.Truth{test.truth, unified.NoMatch})
				}
				explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at,
					unifiedTestID(1))
				if err != nil {
					if !test.conflict || explained != nil ||
						!strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected explanation conflict=%v without partial data, got %+v, %v",
							test.conflict, explained, err)
					}
					return
				}
				if test.conflict {
					t.Fatal("expected explanation source-contract failure, got success")
				}
				if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != test.truth ||
					explained.Nodes[0].Count == nil || explained.Nodes[0].Count.Certain != test.count ||
					explained.Nodes[0].Count.Possible != test.count {
					t.Fatalf("expected deduplicated count %d and truth %v, got %+v", test.count, test.truth, explained)
				}
			})
		}
	}

}
