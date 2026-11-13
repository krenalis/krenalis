// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/warehouses"
)

func assertUnifiedChunkProfiles(t *testing.T, result *unified.Result) {

	t.Helper()
	if result == nil || result.Match != 512 || result.NoMatch != 0 || result.Unknown != 0 || len(result.Profiles) != 512 {
		t.Fatal("expected 512 matching profiles at the exact byte budget, got incomplete result")
	}

	bytesRead := 0
	for i, profile := range result.Profiles {
		var expected strings.Builder
		for k := 1; k <= 32; k++ {
			fmt.Fprintf(&expected, "%x", sha256.Sum256([]byte(fmt.Sprintf("%d/%d", i+1, k))))
		}
		payload, ok := profile.Attributes["payload"].(string)
		if !ok || profile.ID != unifiedTestID(i+1) || profile.Truth != unified.Match || payload != expected.String() {
			t.Fatalf("expected profile %d with exact digest payload, got invalid row", i+1)
		}
		bytesRead += len(payload)
	}
	if bytesRead != warehouses.MaxProfilePageBytes {
		t.Fatalf("expected %d payload bytes, got %d", warehouses.MaxProfilePageBytes, bytesRead)
	}

}

func populateUnifiedChunkProfiles(t *testing.T, platform string, dw warehouses.Warehouse) {
	t.Helper()
	switch platform {
	case "PostgreSQL":
		execUnifiedFixture(t, dw, `TRUNCATE TABLE profile_data;
INSERT INTO profile_data SELECT '00000000-0000-0000-0000-'||lpad(p::text,12,'0'),
 string_agg(encode(sha256((p::text||'/'||k::text)::bytea),'hex'),'' ORDER BY k)
 FROM generate_series(1,512) p CROSS JOIN generate_series(1,32) k GROUP BY p`)
	case "Snowflake":
		execUnifiedFixture(t, dw, `TRUNCATE TABLE "PROFILE_DATA";
INSERT INTO "PROFILE_DATA" SELECT '00000000-0000-0000-0000-'||LPAD(TO_VARCHAR(p.n),12,'0'),
 LISTAGG(LOWER(SHA2(TO_VARCHAR(p.n)||'/'||TO_VARCHAR(k.n),256)),'') WITHIN GROUP (ORDER BY k.n)
 FROM (SELECT ROW_NUMBER() OVER(ORDER BY SEQ4()) AS n FROM TABLE(GENERATOR(ROWCOUNT=>512))) p
 CROSS JOIN (SELECT ROW_NUMBER() OVER(ORDER BY SEQ4()) AS n FROM TABLE(GENERATOR(ROWCOUNT=>32))) k GROUP BY p.n`)
	default:
		t.Fatalf("expected supported warehouse, got %s", platform)
	}
}
