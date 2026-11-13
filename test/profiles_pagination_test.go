// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfilesPagination verifies deterministic pagination and schema-aware
// live reads.
func TestProfilesPagination(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	const (
		pageSize     = 500
		profileCount = 2*pageSize + 1
	)
	storage := krenalistester.NewTempStorage(t)
	defer storage.Remove()
	users := make([]map[string]string, profileCount)
	for i := range users {
		id := strconv.Itoa(i)
		users[i] = map[string]string{
			"id": id, "email": "profile-" + id + "@example.com", "first_name": strconv.Itoa(i % 2),
		}
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
			{Name: "first_name", Type: types.JSON()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{"email": "email", "first_name": "first_name"},
		},
		UserIDColumn: "id",
		Path:         "users.json",
		Format:       "json",
		FormatSettings: krenalistester.SettingsProperties(map[string]bool{
			"id": true, "email": true, "first_name": true,
		}),
	})
	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)
	k.RunIdentityResolutionAndWait()

	var profileSchema krenalistester.ProfileSchema
	k.Call("GET", "/v1/profiles/schema", nil, nil, &profileSchema)
	requestSchema := profileSchema.Schema

	firstPage, expectedTotal, hasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, 0, pageSize)
	if expectedTotal != profileCount {
		t.Fatalf("expected %d profiles, got %d", profileCount, expectedTotal)
	}
	if len(firstPage) != pageSize {
		t.Fatalf("expected %d profiles in the first page, got %d", pageSize, len(firstPage))
	}
	if !hasNext {
		t.Fatal("expected the first profile page to have a continuation, got false")
	}
	maximumPage, maximumTotal, maximumHasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, 0, 1000)
	if len(maximumPage) != 1000 || maximumTotal != profileCount || !maximumHasNext {
		t.Fatalf("expected 1000 profiles, total %d and continuation, got %d profiles, total %d and continuation %t",
			profileCount, len(maximumPage), maximumTotal, maximumHasNext)
	}
	exactPage, exactTotal, exactHasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, 1, 1000)
	if len(exactPage) != 1000 || exactTotal != profileCount || exactHasNext {
		t.Fatalf("expected 1000 profiles, total %d and no continuation, got %d profiles, total %d and continuation %t",
			profileCount, len(exactPage), exactTotal, exactHasNext)
	}
	var summaryResponse map[string]any
	k.Call("POST", "/v1/profiles/query", nil, map[string]any{
		"properties": []string{"email"}, "first": 0, "limit": 1, "schema": requestSchema,
	}, &summaryResponse)
	if _, ok := summaryResponse["schema"]; ok {
		t.Fatalf("expected no schema in profiles response, got %v", summaryResponse["schema"])
	}
	var countResponse struct {
		Total int `json:"total"`
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

	// POST requests carry large schemas in the body instead of the URL.
	postProperties := requestSchema.Properties().Slice()
	postProperties[0].Description = strings.Repeat("x", 32<<10)
	postSchema := types.Object(postProperties)
	var postPage struct {
		Profiles []krenalistester.Profile `json:"profiles"`
		Total    int                      `json:"total"`
		HasNext  bool                     `json:"hasNext"`
	}
	k.Call("POST", "/v1/profiles/query", nil, map[string]any{
		"schema": postSchema, "properties": []string{"email"}, "orderDesc": true, "first": 1, "limit": 1,
	}, &postPage)
	if len(postPage.Profiles) != 1 || postPage.Total != expectedTotal || !postPage.HasNext {
		t.Fatalf("expected one profile, total %d and continuation, got %d profiles, total %d and continuation %t",
			expectedTotal, len(postPage.Profiles), postPage.Total, postPage.HasNext)
	}
	if postPage.Profiles[0].KPID != firstPage[1].KPID || len(postPage.Profiles[0].Attributes) != 1 {
		t.Fatalf("expected profile %s with only email, got %v", firstPage[1].KPID, postPage.Profiles[0])
	}
	k.Call("POST", "/v1/profiles/query", nil, map[string]any{"schema": postSchema}, &postPage)
	if len(postPage.Profiles) != 100 {
		t.Fatalf("expected the default limit of 100 profiles, got %d", len(postPage.Profiles))
	}
	for _, body := range []map[string]any{{}, {"schema": postSchema, "filter": json.Value(filter)}} {
		k.Call("POST", "/v1/profiles/count", nil, body, &countResponse)
		want := expectedTotal
		if body["filter"] != nil {
			want = 1
		}
		if countResponse.Total != want {
			t.Fatalf("expected POST count %d, got %d", want, countResponse.Total)
		}
	}
	var postAttributes struct {
		Attributes map[string]any `json:"attributes"`
	}
	k.Call("POST", "/v1/profiles/"+firstPage[0].KPID.String()+"/attributes/query", nil,
		map[string]any{"schema": postSchema}, &postAttributes)
	if postAttributes.Attributes["email"] != email {
		t.Fatalf("expected POST email %q, got %v", email, postAttributes.Attributes["email"])
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
			t.Fatalf("expected hasNext %t at offset %d, got %t", first+len(page) < expectedTotal, first, pageHasNext)
		}
		profiles = append(profiles, page...)
	}

	seen := make(map[string]struct{}, len(profiles))
	hasEqualUpdateTimes := false
	for i, profile := range profiles {

		kpid := profile.KPID.String()
		if _, ok := seen[kpid]; ok {
			t.Fatalf("expected profile %s in one page, got a duplicate at index %d", kpid, i)
		}
		seen[kpid] = struct{}{}
		if i == 0 {
			continue
		}

		previous := profiles[i-1]
		if previous.UpdatedAt.Before(profile.UpdatedAt) {
			t.Fatalf("expected update times in descending order, got %s before %s",
				previous.UpdatedAt, profile.UpdatedAt)
		}
		if previous.UpdatedAt.Equal(profile.UpdatedAt) {
			hasEqualUpdateTimes = true
			if bytes.Compare(previous.KPID[:], profile.KPID[:]) <= 0 {
				t.Fatalf("expected descending KPIDs for equal update times, got %s before %s",
					previous.KPID, profile.KPID)
			}
		}

	}
	if !hasEqualUpdateTimes {
		t.Fatal("expected profiles with the same update time, got none")
	}

	// Profiles with the same first name are ordered by KPID across pages.
	const orderPageSize = 20
	for _, descending := range []bool{false, true} {
		var ordered []krenalistester.Profile
		for first := 0; first < 2*orderPageSize; first += orderPageSize {
			page, _, _ := k.ProfilesWithSchema(
				requestSchema, []string{"first_name"}, "first_name", descending, first, orderPageSize)
			if len(page) != orderPageSize {
				t.Fatalf("expected %d profiles, got %d", orderPageSize, len(page))
			}
			ordered = append(ordered, page...)
		}
		want := "0"
		if descending {
			want = "1"
		}
		for i, profile := range ordered {
			if profile.Attributes["first_name"] != want {
				t.Fatalf("expected first name %q, got %v", want, profile.Attributes["first_name"])
			}
			if i > 0 {
				previous := ordered[i-1].KPID
				comparison := bytes.Compare(previous[:], profile.KPID[:])
				if (!descending && comparison >= 0) || (descending && comparison <= 0) {
					t.Fatalf("expected KPIDs ordered with descending %t, got %s before %s",
						descending, previous, profile.KPID)
				}
			}
		}
	}

	attributesPath := "/v1/profiles/" + firstPage[0].KPID.String() + "/attributes/query"
	attributesBody := map[string]any{"schema": requestSchema}
	var attributesResponse struct {
		Attributes map[string]any `json:"attributes"`
	}
	k.Call("POST", attributesPath, nil, attributesBody, &attributesResponse)
	if attributesResponse.Attributes["email"] != email {
		t.Fatalf("expected email %q, got %v", email, attributesResponse.Attributes["email"])
	}

	empty, _, emptyHasNext := k.ProfilesWithSchema(
		requestSchema, []string{"email"}, "", true, expectedTotal, pageSize)
	if len(empty) != 0 || emptyHasNext {
		t.Fatalf("expected 0 profiles and no continuation, got %d profiles and continuation %t",
			len(empty), emptyHasNext)
	}

	emailProperty, _ := requestSchema.Properties().ByName("email")
	unrelatedSchema := types.Object([]types.Property{
		emailProperty,
		{Name: "no_longer_present", Type: types.String()},
	})
	k.Call("POST", "/v1/profiles/count", nil, map[string]any{"schema": requestSchema}, &countResponse)
	if countResponse.Total != expectedTotal {
		t.Fatalf("expected %d profiles, got %d", expectedTotal, countResponse.Total)
	}

	// Check schema alignment, including unused properties, and invalid arguments.
	for _, tc := range []struct {
		name    string
		method  string
		path    string
		body    any
		status  int
		code    string
		message string
	}{
		{name: "removed GET profiles", path: "/v1/profiles", status: http.StatusNotFound},
		{name: "GET count is not allowed", path: "/v1/profiles/count", status: http.StatusMethodNotAllowed},
		{name: "removed GET attributes", path: "/v1/profiles/" + firstPage[0].KPID.String() + "/attributes", status: http.StatusNotFound},
		{name: "POST missing body", method: "POST", path: "/v1/profiles/query", status: http.StatusBadRequest},
		{name: "POST invalid body", method: "POST", path: "/v1/profiles/count", body: "invalid", status: http.StatusBadRequest},
		{name: "POST missing schema", method: "POST", path: "/v1/profiles/query", body: map[string]any{}, status: http.StatusBadRequest, code: "BadRequest", message: "schema is required"},
		{name: "POST invalid limit", method: "POST", path: "/v1/profiles/query", body: map[string]any{"schema": postSchema, "limit": 0}, status: http.StatusBadRequest},
		{
			name:   "malformed schema",
			method: "POST",
			path:   "/v1/profiles/query",
			body:   map[string]any{"schema": "{"},
			status: http.StatusBadRequest,
			code:   "BadRequest",
		},
		{
			name:   "empty properties",
			method: "POST",
			path:   "/v1/profiles/query",
			body:   map[string]any{"schema": requestSchema, "properties": []string{}},
			status: http.StatusBadRequest,
		},
		{
			name:   "duplicate properties",
			method: "POST",
			path:   "/v1/profiles/query",
			body:   map[string]any{"schema": requestSchema, "properties": []string{"email", "email"}},
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown requested property",
			method: "POST",
			path:   "/v1/profiles/query",
			body:   map[string]any{"schema": requestSchema, "properties": []string{"missing"}},
			status: http.StatusUnprocessableEntity,
			code:   "PropertyNotExist",
		},
		{
			name:   "omitted properties uses supplied schema",
			method: "POST",
			path:   "/v1/profiles/query",
			body:   map[string]any{"schema": unrelatedSchema},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:   "profiles check unused schema",
			method: "POST",
			path:   "/v1/profiles/query",
			body: map[string]any{"schema": unrelatedSchema,
				"properties": []string{"email"}, "filter": json.Value(filter), "order": "email"},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:   "order property not aligned",
			method: "POST",
			path:   "/v1/profiles/query",
			body: map[string]any{"schema": unrelatedSchema,
				"properties": []string{"email"}, "order": "no_longer_present"},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:    "count needs filter schema",
			method:  "POST",
			path:    "/v1/profiles/count",
			body:    map[string]any{"filter": json.Value(filter)},
			status:  http.StatusBadRequest,
			code:    "BadRequest",
			message: "schema is required",
		},
		{
			name:   "count checks unused schema with filter",
			method: "POST",
			path:   "/v1/profiles/count",
			body:   map[string]any{"schema": unrelatedSchema, "filter": json.Value(filter)},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:   "count checks schema without filter",
			method: "POST",
			path:   "/v1/profiles/count",
			body:   map[string]any{"schema": unrelatedSchema},
			status: http.StatusUnprocessableEntity,
			code:   "SchemaNotAligned",
		},
		{
			name:    "attributes need schema",
			method:  "POST",
			path:    attributesPath,
			body:    map[string]any{},
			status:  http.StatusBadRequest,
			code:    "BadRequest",
			message: "schema is required",
		},
		{
			name:   "attributes malformed schema",
			method: "POST",
			path:   attributesPath,
			body:   map[string]any{"schema": "{"},
			status: http.StatusBadRequest,
			code:   "BadRequest",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = "GET"
			}
			err := k.TryCall(method, tc.path, nil, tc.body, nil)
			if err != nil {
				statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
				if !ok || statusErr.Response.Code != tc.status {
					t.Fatalf("expected HTTP %d, got %v", tc.status, err)
				}
				if tc.code != "" {
					var response struct {
						Error struct {
							Code    string `json:"code"`
							Message string `json:"message"`
						} `json:"error"`
					}
					err = json.Unmarshal([]byte(statusErr.Response.Text), &response)
					if err != nil {
						t.Fatal(err)
					}
					if response.Error.Code != tc.code {
						t.Fatalf("expected code %s, got %s", tc.code, response.Error.Code)
					}
					if tc.message != "" && response.Error.Message != tc.message {
						t.Fatalf("expected message %q, got %q", tc.message, response.Error.Message)
					}
				}
				return
			}
			t.Fatal("expected an error, got nil")
		})
	}

	// Reusing a name with a different type must not reinterpret the arguments.
	changedEmail := emailProperty
	changedEmail.Type = types.Int(64)
	changedSchema := types.Object([]types.Property{changedEmail})
	numericFilter := `{"operator":"and","rules":[{"property":"email","operator":"is","values":["1"]}]}`
	for _, tc := range []struct {
		path string
		body any
	}{
		{path: "/v1/profiles/query", body: map[string]any{"schema": changedSchema}},
		{path: "/v1/profiles/count", body: map[string]any{"schema": changedSchema, "filter": json.Value(numericFilter)}},
		{path: attributesPath, body: map[string]any{"schema": changedSchema}},
	} {
		err = k.TryCall("POST", tc.path, nil, tc.body, nil)
		if err != nil {
			statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
			if !ok || statusErr.Response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected incompatible schema error, got %v", err)
			}
			var response struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			err = json.Unmarshal([]byte(statusErr.Response.Text), &response)
			if err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != "SchemaNotAligned" {
				t.Fatalf("expected code SchemaNotAligned, got %q", response.Error.Code)
			}
			continue
		}
		t.Fatal("expected incompatible schema error, got nil")
	}

	// Compatible alterations and subsequent IR publications keep old arguments usable.
	newProperties := requestSchema.Properties().Slice()
	newProperties = append(newProperties, types.Property{
		Name: "new_property", Type: types.String(), ReadOptional: true,
	})
	k.AlterProfileSchemaAndWait(types.Object(newProperties), nil, nil)
	partialSchema := types.Object([]types.Property{emailProperty})
	page, _, _ := k.ProfilesWithSchema(partialSchema, nil, "", true, 0, 1)
	if len(page) != 1 {
		t.Fatalf("expected one profile, got %d", len(page))
	}
	expectedEmail, ok := page[0].Attributes["email"].(string)
	if len(page[0].Attributes) != 1 || !ok {
		t.Fatalf("expected only an email attribute, got %v", page[0].Attributes)
	}
	k.RunIdentityResolutionAndWait()
	page, _, _ = k.ProfilesWithSchema(partialSchema, nil, "", true, 0, 1)
	if len(page) != 1 {
		t.Fatalf("expected one profile after identity resolution, got %d", len(page))
	}
	if len(page[0].Attributes) != 1 || page[0].Attributes["email"] != expectedEmail {
		t.Fatalf("expected only email %q after identity resolution, got %v", expectedEmail, page[0].Attributes)
	}

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
	k.Call("POST", attributesPath, nil, attributesBody, &attributesResponse)
	if attributesResponse.Attributes["email"] != "changed@example.com" {
		t.Fatalf("expected email changed@example.com after update, got %v", attributesResponse.Attributes["email"])
	}
	// A partial nested object describes both the selected columns and serialization.
	android, _ := requestSchema.Properties().ByName("android")
	androidID, _ := android.Type.Properties().ByName("id")
	android.Type = types.Object([]types.Property{androidID})
	nestedSchema := types.Object([]types.Property{emailProperty, android})
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
	k.Call("POST", "/v1/profiles/query", nil, map[string]any{
		"schema": nestedSchema, "properties": []string{"android"}, "filter": json.Value(nestedFilter),
	}, &nestedResponse)
	if len(nestedResponse.Profiles) != 1 {
		t.Fatalf("expected one nested match, got %d", len(nestedResponse.Profiles))
	}
	nested, ok := nestedResponse.Profiles[0].Attributes["android"].(map[string]any)
	if !ok || len(nested) != 1 || nested["id"] != "nested-id" {
		t.Fatalf("expected only android.id nested-id, got %v", nestedResponse.Profiles[0].Attributes["android"])
	}
	k.Call("POST", "/v1/profiles/count", nil, map[string]any{
		"schema": nestedSchema, "filter": json.Value(nestedFilter),
	}, &countResponse)
	if countResponse.Total != 1 {
		t.Fatalf("expected one nested count match, got %d", countResponse.Total)
	}

	_, err = pool.Exec(t.Context(), "DELETE FROM profiles WHERE _kpid = $1", firstPage[0].KPID.String())
	if err != nil {
		t.Fatal(err)
	}
	err = k.TryCall("POST", attributesPath, nil, attributesBody, nil)
	if err != nil {
		statusErr, ok := errors.AsType[*krenalistester.StatusCodeError](err)
		if !ok || statusErr.Response.Code != http.StatusNotFound {
			t.Fatalf("expected a local not-found error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected a missing profile error, got nil")
	}
	k.Call("POST", "/v1/profiles/count", nil, map[string]any{}, &countResponse)
	if countResponse.Total != expectedTotal-1 {
		t.Fatalf("expected the live count %d, got %d", expectedTotal-1, countResponse.Total)
	}

}
