// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfilesCountAPI exercises the actual HTTP/core/datastore/warehouse path.
func TestProfilesCountAPI(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL, NATS and the API server")
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetProfileSchema(types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
		{Name: "country", Type: types.String(), ReadOptional: true},
	}))
	k.Start()
	defer k.Stop()
	k.UpdateIdentityResolutionSettings(false, nil)
	var settings krenalistester.DBSettings
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), &settings)
	if err != nil {
		t.Fatalf("expected warehouse settings, got %v", err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
	if err != nil {
		t.Fatalf("expected isolated warehouse connection, got %v", err)
	}
	defer pool.Close()

	// Publish the coherent Profiles/event fixture directly, with no asynchronous
	// writer or Identity Resolution running during this test.
	_, err = pool.Exec(t.Context(), `INSERT INTO krenalis_profiles_0
 (_kpid,_updated_at,email,country,_anonymous_count,_recognized_count)
 SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,'2026-02-01 12:00:00.123456',
 'profile-'||n||'@example.com',CASE WHEN n IN (1,3) THEN 'IT' ELSE 'FR' END,0,1 FROM generate_series(1,4) n;
INSERT INTO krenalis_events
 (kpid,message_id,event,timestamp,received_at,sent_at,type,connection_id,anonymous_id,traits,properties)
SELECT _kpid,'e'||n,'Purchase','2026-01-15','2026-02-01','2026-02-01','track','fixture','anonymous-'||n,'{}',
 CASE n WHEN 2 THEN '{"currency":"EUR"}' WHEN 3 THEN '{"currency":null}' ELSE '{}' END::jsonb
 FROM (SELECT _kpid,right(_kpid::text,1)::int n FROM profiles) p WHERE n>1;`)
	if err != nil {
		t.Fatalf("expected coherent API fixture, got %v", err)
	}
	const count = `{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents",` +
		`"name":"Purchase","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
		`"duration":{"value":31,"unit":"day"}},"where":{"operator":"and","rules":[` +
		`{"property":"properties.currency","operator":"is","values":["EUR"]}]}},"operator":"is","values":[0]}]}`
	const at = "2026-02-01T00:00:00Z"
	query := url.Values{"filter": {count}, "at": {at}, "order": {"email"}, "limit": {"1"}}
	type evaluation struct {
		At      time.Time `json:"at"`
		Match   int64     `json:"match"`
		NoMatch int64     `json:"noMatch"`
		Unknown int64     `json:"unknown"`
	}
	type response struct {
		Profiles   []krenalistester.Profile `json:"profiles"`
		Schema     types.Type               `json:"schema"`
		Total      int                      `json:"total"`
		Evaluation *evaluation              `json:"evaluation"`
	}
	var page response
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if len(page.Profiles) != 1 || !strings.HasSuffix(page.Profiles[0].KPID.String(), "0001") || page.Total != 2 ||
		page.Evaluation == nil || page.Evaluation.At.Format(time.RFC3339Nano) != at ||
		page.Evaluation.Match != 2 || page.Evaluation.NoMatch != 1 || page.Evaluation.Unknown != 1 {
		t.Fatalf("expected MATCH-only first page and totals 2/1/1 at T, got %+v", page)
	}
	firstEvaluation := *page.Evaluation
	query.Set("first", "1")
	query.Set("properties", "email")
	page = response{}
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if len(page.Profiles) != 1 || !strings.HasSuffix(page.Profiles[0].KPID.String(), "0003") ||
		page.Evaluation == nil || *page.Evaluation != firstEvaluation || page.Schema.Properties().Len() != 1 {
		t.Fatalf("expected projected second page at the same T, got %+v", page)
	}
	query.Set("first", "99")
	page = response{}
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if len(page.Profiles) != 0 || page.Total != 2 || page.Evaluation == nil || *page.Evaluation != firstEvaluation {
		t.Fatalf("expected empty page with complete evaluation totals, got %+v", page)
	}
	query.Del("first")
	query.Del("at")
	page = response{}
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if page.Evaluation == nil || page.Evaluation.At.After(time.Now()) || page.Evaluation.At.Nanosecond()%1000 != 0 {
		t.Fatalf("expected automatic nonfuture microsecond T, got %+v", page)
	}

	// The old response shape and byte encoding remain untouched for Record.
	var raw json.RawMessage
	k.Call("GET", "/v1/profiles?properties=email&order=email&limit=1", nil, nil, &raw)
	schema, err := types.Object([]types.Property{{Name: "email", Type: types.String(), ReadOptional: true}}).MarshalJSON()
	if err != nil {
		t.Fatalf("expected response schema, got %v", err)
	}
	want := `{"profiles":[{"kpid":"00000000-0000-0000-0000-000000000001",` +
		`"updatedAt":"2026-02-01T12:00:00.123456Z","attributes":{"email":"profile-1@example.com"}}],` +
		`"schema":` + string(schema) + `,"total":4}`
	if string(raw) != want {
		t.Fatalf("expected unchanged Record response %s, got %s", want, raw)
	}

	for _, test := range []struct {
		name   string
		filter string
		at     string
		status int
		code   string
	}{
		{"negative", strings.Replace(count, `"values":[0]`, `"values":[-1]`, 1), at, 400, ""},
		{"too precise", count, "2026-02-01T00:00:00.000000001Z", 400, ""},
		{"truncation", count, "2026-02-01T00:00:00.0000000001Z", 400, ""},
		{"future", count, "9999-01-01T00:00:00Z", 400, ""},
		{"empty at", count, "", 400, ""},
		{"at without Count", `{"operator":"and","rules":[{"property":"email","operator":"exists","values":[]}]}`, at, 400, ""},
		{"unknown property", strings.Replace(count, `"properties.currency"`, `"notPresent"`, 1), at, 422, "PropertyNotExist"},
		{"unsupported range", strings.Replace(count, `"values":[0]`, `"values":[9223372036854775808]`, 1), at, 422, "FilterCapabilityNotImplemented"},
		{"long valid literal", strings.Replace(count, `"values":[0]`, `"values":[`+strings.Repeat("9", 10000)+`]`, 1), at, 422, "FilterCapabilityNotImplemented"},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := url.Values{"filter": {test.filter}, "at": {test.at}}
			var body json.RawMessage
			err := k.TryCall("GET", "/v1/profiles?"+q.Encode(), nil, nil, &body)
			if err != nil {
				status, ok := errors.AsType[*krenalistester.StatusCodeError](err)
				if !ok || status.Response.Code != test.status ||
					test.code != "" && !strings.Contains(status.Response.Text, test.code) {
					t.Fatalf("expected HTTP %d / %s, got %v", test.status, test.code, err)
				}
			}
			if err == nil {
				t.Fatalf("expected HTTP %d, got success: %s", test.status, body)
			}
		})
	}

	_, err = pool.Exec(t.Context(), `TRUNCATE krenalis_profiles_0`)
	if err != nil {
		t.Fatalf("expected empty population fixture, got %v", err)
	}
	query.Set("at", at)
	page = response{}
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if len(page.Profiles) != 0 || page.Total != 0 || page.Evaluation == nil ||
		page.Evaluation.Match != 0 || page.Evaluation.NoMatch != 0 || page.Evaluation.Unknown != 0 {
		t.Fatalf("expected empty population with three zero totals, got %+v", page)
	}

	// A source failure propagates as an operation failure; it must not fall
	// back to a Record query or report successful zero counts.
	_, err = pool.Exec(t.Context(), `ALTER TABLE krenalis_events RENAME TO missing_events`)
	if err != nil {
		t.Fatalf("expected failure fixture, got %v", err)
	}
	query.Set("at", at)
	err = k.TryCall("GET", "/v1/profiles?"+query.Encode(), nil, nil, &page)
	if err != nil {
		status, ok := errors.AsType[*krenalistester.StatusCodeError](err)
		if !ok || status.Response.Code != 503 || strings.Contains(status.Response.Text, "missing_events") {
			t.Fatalf("expected unavailable operation without warehouse failure text, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected technical failure, got successful page")
	}

}
