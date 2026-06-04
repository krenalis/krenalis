// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalBase58IDMap(t *testing.T) {
	got := map[int]string{}
	err := unmarshalBase58IDMap([]byte(`{"100":"4fr7Kp9ZaQ2m","3":"8QaT3mN7KxP5"}`), got)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{
		100: "4fr7Kp9ZaQ2m",
		3:   "8QaT3mN7KxP5",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBase58IDMigrationQueriesIncludePurgedPipelines(t *testing.T) {
	if !strings.Contains(pipelineIDMapQuery, "pipelines_to_purge") {
		t.Fatalf("expected pipeline ID map query to include pipelines_to_purge")
	}
	queries := strings.Join(base58IDMigrationQueries(), "\n")
	for _, want := range []string{
		"CROSS JOIN LATERAL unnest(w.pipelines_to_purge)",
		"FROM unnest(workspaces.pipelines_to_purge)",
		"ALTER TABLE workspaces ALTER COLUMN pipelines_to_purge SET DEFAULT",
	} {
		if !strings.Contains(queries, want) {
			t.Fatalf("expected migration queries to contain %q", want)
		}
	}
}
