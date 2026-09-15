// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfilesPagination verifies deterministic pagination and schema-aware live reads.
func TestProfilesPagination(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	const (
		pageSize     = 200
		profileCount = 2*pageSize + 1
	)
	storage := krenalistester.NewTempStorage(t)
	defer storage.Remove()
	users := make([]map[string]string, profileCount)
	for i := range users {
		id := strconv.Itoa(i)
		users[i] = map[string]string{"id": id, "email": "profile-" + id + "@example.com"}
	}
	content, err := json.Marshal(users)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(storage.Root(), "users.json"), content, 0644)
	if err != nil {
		t.Fatal(err)
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.SetFileSystemRoot(storage.Root())
	k.Start()
	defer k.Stop()

	k.UpdateIdentityResolutionSettings(false, nil)
	source := k.CreateSourceFileSystem()
	pipeline := k.CreatePipeline(source, "User", krenalistester.PipelineToSet{
		Name:    "Import users from JSON",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "id", Type: types.JSON()},
			{Name: "email", Type: types.JSON()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{"email": "email"},
		},
		UserIDColumn: "id",
		Path:         "users.json",
		Format:       "json",
		FormatSettings: krenalistester.SettingsProperties(map[string]bool{
			"id": true, "email": true,
		}),
	})
	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)
	k.RunIdentityResolutionAndWait()

	var requestSchema types.Type
	k.Call("GET", "/v1/profiles/schema", nil, nil, &requestSchema)
	schemaJSON, err := requestSchema.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	firstPage, expectedTotal, hasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, 0, pageSize)
	if expectedTotal != profileCount {
		t.Fatalf("expected %d profiles, got %d", profileCount, expectedTotal)
	}
	if len(firstPage) != pageSize {
		t.Fatalf("expected %d profiles in the first page, got %d", pageSize, len(firstPage))
	}
	if !hasNext {
		t.Fatal("expected the first profile page to have a continuation")
	}
	var summaryResponse map[string]any
	k.Call("GET", "/v1/profiles?"+url.Values{
		"properties": []string{"email"},
		"first":      []string{"0"},
		"limit":      []string{"1"},
		"schema":     []string{string(schemaJSON)},
	}.Encode(), nil, nil, &summaryResponse)
	if _, ok := summaryResponse["schema"]; ok {
		t.Fatal("unexpected schema in profiles response")
	}
	var countResponse struct {
		Total int `json:"total"`
	}
	k.Call("GET", "/v1/profiles/count", nil, nil, &countResponse)
	if countResponse.Total != expectedTotal {
		t.Fatalf("unexpected count response: total=%d", countResponse.Total)
	}
	email, ok := firstPage[0].Attributes["email"].(string)
	if !ok {
		t.Fatalf("expected a string email, got %T", firstPage[0].Attributes["email"])
	}
	filter, err := json.Marshal(map[string]any{
		"operator": "and",
		"rules": []any{map[string]any{
			"property": "email",
			"operator": "is",
			"values":   []string{email},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	k.Call("GET", "/v1/profiles/count?"+url.Values{
		"filter": []string{string(filter)},
		"schema": []string{string(schemaJSON)},
	}.Encode(), nil, nil, &countResponse)
	if countResponse.Total != 1 {
		t.Fatalf("unexpected filtered count response: total=%d", countResponse.Total)
	}
	profiles := firstPage
	for first := pageSize; first < expectedTotal; first += pageSize {
		page, total, pageHasNext := k.ProfilesWithSchema(
			requestSchema, []string{"email"}, "", true, first, pageSize)
		if total != expectedTotal {
			t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
		}
		expectedPageSize := min(pageSize, expectedTotal-first)
		if len(page) != expectedPageSize {
			t.Fatalf("expected %d profiles at offset %d, got %d", expectedPageSize, first, len(page))
		}
		if pageHasNext != (first+len(page) < expectedTotal) {
			t.Fatalf("unexpected hasNext value %t at offset %d", pageHasNext, first)
		}
		profiles = append(profiles, page...)
	}

	seen := make(map[string]struct{}, len(profiles))
	hasEqualUpdateTimes := false
	for i, profile := range profiles {

		kpid := profile.KPID.String()
		if _, ok := seen[kpid]; ok {
			t.Fatalf("profile %s occurs in more than one page", kpid)
		}
		seen[kpid] = struct{}{}
		if i == 0 {
			continue
		}

		previous := profiles[i-1]
		if previous.UpdatedAt.Before(profile.UpdatedAt) {
			t.Fatalf("profile %s is newer than the preceding profile %s", profile.KPID, previous.KPID)
		}
		if previous.UpdatedAt.Equal(profile.UpdatedAt) {
			hasEqualUpdateTimes = true
			if bytes.Compare(previous.KPID[:], profile.KPID[:]) <= 0 {
				t.Fatalf("profiles %s and %s with the same update time are not ordered by descending KPID",
					previous.KPID, profile.KPID)
			}
		}

	}
	if !hasEqualUpdateTimes {
		t.Fatal("expected profiles with the same update time")
	}

	attributesPath := "/v1/profiles/" + firstPage[0].KPID.String() + "/attributes?" + url.Values{
		"schema": []string{string(schemaJSON)},
	}.Encode()
	var attributesResponse struct {
		Attributes map[string]any `json:"attributes"`
	}
	k.Call("GET", attributesPath, nil, nil, &attributesResponse)
	if attributesResponse.Attributes["email"] != email {
		t.Fatal("attributes do not match the requested profile")
	}

	empty, _, emptyHasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, expectedTotal, pageSize)
	if len(empty) != 0 || emptyHasNext {
		t.Fatalf("unexpected empty range: profiles=%d hasNext=%t", len(empty), emptyHasNext)
	}

	// An unrelated schema difference is irrelevant to a request's dependencies.
	emailProperty, _ := requestSchema.Properties().ByName("email")
	partialSchema := types.Object([]types.Property{emailProperty})
	unrelatedSchema := types.Object([]types.Property{
		emailProperty,
		{Name: "no_longer_present", Type: types.String()},
	})
	unrelatedJSON, err := unrelatedSchema.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{
		"schema": []string{string(unrelatedJSON)}, "properties": []string{"email"}, "limit": []string{"1"},
	}
	k.Call("GET", "/v1/profiles?"+query.Encode(), nil, nil, &summaryResponse)
	k.Call("GET", "/v1/profiles/count?"+url.Values{
		"schema": []string{string(unrelatedJSON)}, "filter": []string{string(filter)},
	}.Encode(), nil, nil, &countResponse)
	if countResponse.Total != 1 {
		t.Fatalf("unexpected filtered count %d", countResponse.Total)
	}

	// Referenced properties must align, even when they occur only in a filter
	// or order. Invalid arguments must be rejected before querying.
	for _, tc := range []struct {
		name   string
		path   string
		query  url.Values
		status int
		code   string
	}{
		{name: "missing schema", path: "/v1/profiles", status: http.StatusBadRequest},
		{
			name:   "malformed schema",
			path:   "/v1/profiles",
			query:  url.Values{"schema": []string{"{"}},
			status: http.StatusBadRequest,
		},
		{
			name: "empty properties",
			path: "/v1/profiles",
			query: url.Values{"schema": []string{string(schemaJSON)},
				"properties": []string{""}},
			status: http.StatusBadRequest,
		},
		{
			name: "duplicate properties",
			path: "/v1/profiles",
			query: url.Values{"schema": []string{string(schemaJSON)},
				"properties": []string{"email,email"}},
			status: http.StatusBadRequest,
		},
		{
			name: "unknown requested property",
			path: "/v1/profiles",
			query: url.Values{"schema": []string{string(schemaJSON)},
				"properties": []string{"missing"}},
			status: http.StatusUnprocessableEntity,
			code:   "PropertyNotExist",
		},
		{
			name:   "omitted properties uses supplied schema",
			path:   "/v1/profiles",
			query:  url.Values{"schema": []string{string(unrelatedJSON)}},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name: "order dependency",
			path: "/v1/profiles",
			query: url.Values{"schema": []string{string(unrelatedJSON)},
				"properties": []string{"email"},
				"order":      []string{"no_longer_present"}},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:   "count needs filter schema",
			path:   "/v1/profiles/count",
			query:  url.Values{"filter": []string{string(filter)}},
			status: http.StatusBadRequest,
		},
		{
			name:   "attributes need schema",
			path:   "/v1/profiles/" + firstPage[0].KPID.String() + "/attributes",
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := k.TryCall("GET", tc.path+"?"+tc.query.Encode(), nil, nil, nil)
			if err != nil {
				statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
				if !ok || statusErr.Response.Code != tc.status {
					t.Fatalf("expected HTTP %d, got %v", tc.status, err)
				}
				if tc.code != "" {
					var response struct {
						Error struct {
							Code string `json:"code"`
						} `json:"error"`
					}
					err = json.Unmarshal([]byte(statusErr.Response.Text), &response)
					if err != nil {
						t.Fatal(err)
					}
					if response.Error.Code != tc.code {
						t.Fatalf("expected code %s, got %s", tc.code, response.Error.Code)
					}
				}
				return
			}
			t.Fatal("expected an error")
		})
	}

	// Reusing a name with a different type must not reinterpret the arguments.
	changedEmail := emailProperty
	changedEmail.Type = types.Int(64)
	changedSchema := types.Object([]types.Property{changedEmail})
	changedJSON, err := changedSchema.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	numericFilter := `{"operator":"and","rules":[{"property":"email","operator":"is","values":["1"]}]}`
	for _, path := range []string{
		"/v1/profiles?",
		"/v1/profiles/count?filter=" + url.QueryEscape(numericFilter) + "&",
		"/v1/profiles/" + firstPage[0].KPID.String() + "/attributes?",
	} {
		err = k.TryCall("GET", path+url.Values{"schema": []string{string(changedJSON)}}.Encode(), nil, nil, nil)
		if err != nil {
			statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
			if !ok || statusErr.Response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected incompatible schema error, got %v", err)
			}
			continue
		}
		t.Fatal("expected incompatible schema error")
	}

	// Compatible alterations and subsequent IR publications keep old arguments usable.
	newProperties := requestSchema.Properties().Slice()
	newProperties = append(newProperties, types.Property{
		Name: "new_property", Type: types.String(), ReadOptional: true,
	})
	k.AlterProfileSchemaAndWait(types.Object(newProperties), nil, nil)
	page, _, _ := k.ProfilesWithSchema(partialSchema, nil, "", true, 0, 1)
	if len(page) != 1 {
		t.Fatalf("expected one profile, got %d", len(page))
	}
	_, hasEmail := page[0].Attributes["email"]
	if len(page[0].Attributes) != 1 || !hasEmail {
		t.Fatal("omitting properties must return only attributes from the supplied schema")
	}
	k.RunIdentityResolutionAndWait()
	k.ProfilesWithSchema(partialSchema, nil, "", true, 0, 1)

	// Update and remove profiles without running Identity Resolution.
	var settings krenalistester.DBSettings
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), &settings)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := krenalistester.ConnectionPool(t.Context(), &settings)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(t.Context(), "UPDATE profiles SET email = $1 WHERE _kpid = $2",
		"changed@example.com", firstPage[0].KPID.String())
	if err != nil {
		t.Fatal(err)
	}
	k.Call("GET", attributesPath, nil, nil, &attributesResponse)
	if attributesResponse.Attributes["email"] != "changed@example.com" {
		t.Fatal("attributes did not reflect a live update")
	}
	// A partial nested object describes both the selected columns and serialization.
	android, _ := requestSchema.Properties().ByName("android")
	androidID, _ := android.Type.Properties().ByName("id")
	android.Type = types.Object([]types.Property{androidID})
	nestedSchema := types.Object([]types.Property{emailProperty, android})
	nestedJSON, err := nestedSchema.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(),
		"UPDATE profiles SET android_id = 'nested-id', android_idfa = 'unrequested' WHERE _kpid = $1",
		firstPage[0].KPID.String())
	if err != nil {
		t.Fatal(err)
	}
	nestedFilter := `{"operator":"and","rules":[{"property":"android.id","operator":"is","values":["nested-id"]}]}`
	var nestedResponse struct {
		Profiles []struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"profiles"`
	}
	k.Call("GET", "/v1/profiles?"+url.Values{
		"schema": []string{string(nestedJSON)}, "properties": []string{"android"}, "filter": []string{nestedFilter},
	}.Encode(), nil, nil, &nestedResponse)
	if len(nestedResponse.Profiles) != 1 {
		t.Fatalf("expected one nested match, got %d", len(nestedResponse.Profiles))
	}
	nested, ok := nestedResponse.Profiles[0].Attributes["android"].(map[string]any)
	if !ok || len(nested) != 1 || nested["id"] != "nested-id" {
		t.Fatalf("unexpected nested attributes: %v", nested)
	}
	k.Call("GET", "/v1/profiles/count?"+url.Values{
		"schema": []string{string(nestedJSON)}, "filter": []string{nestedFilter},
	}.Encode(), nil, nil, &countResponse)
	if countResponse.Total != 1 {
		t.Fatalf("expected one nested count match, got %d", countResponse.Total)
	}

	_, err = pool.Exec(t.Context(), "DELETE FROM profiles WHERE _kpid = $1", firstPage[0].KPID.String())
	if err != nil {
		t.Fatal(err)
	}
	err = k.TryCall("GET", attributesPath, nil, nil, nil)
	if err != nil {
		statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
		if !ok || statusErr.Response.Code != http.StatusNotFound {
			t.Fatalf("expected a local not-found error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected a missing profile error")
	}
	k.Call("GET", "/v1/profiles/count", nil, nil, &countResponse)
	if countResponse.Total != expectedTotal-1 {
		t.Fatalf("expected the live count %d, got %d", expectedTotal-1, countResponse.Total)
	}

}
