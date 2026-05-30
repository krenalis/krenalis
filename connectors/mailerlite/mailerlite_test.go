// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mailerlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const testAPIKey = "mailerlite-test-token"

// TestSaveSettingsUsesNewAPIAuth checks that settings validation uses the new API.
func TestSaveSettingsUsesNewAPIAuth(t *testing.T) {

	httpClient := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/fields" {
				t.Fatalf("expected path /api/fields, got %q", r.URL.Path)
			}
			if r.URL.Query().Get("limit") != "1" {
				t.Fatalf("expected limit=1, got raw query %q", r.URL.RawQuery)
			}
			if auth := r.Header.Get("Authorization"); auth != "Bearer "+testAPIKey {
				t.Fatalf("expected bearer Authorization header, got %q", auth)
			}
			if version := r.Header.Get("X-Version"); version != apiVersion {
				t.Fatalf("expected X-Version %q, got %q", apiVersion, version)
			}
			if accept := r.Header.Get("Accept"); accept != "application/json" {
				t.Fatalf("expected Accept application/json, got %q", accept)
			}
			return jsonResponse(http.StatusOK, `{"data":[]}`), nil
		},
	}

	oldAPIBaseURL := apiBaseURL
	apiBaseURL = "https://connect.mailerlite.test/api"
	defer func() { apiBaseURL = oldAPIBaseURL }()

	ml := &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   &testSettingsStore{},
			HTTPClient: httpClient,
		},
	}

	settings, err := json.Marshal(innerSettings{APIKey: testAPIKey})
	if err != nil {
		t.Fatalf("expected settings marshal to succeed, got %v", err)
	}

	ui, err := ml.ServeUI(t.Context(), "save", settings, connectors.Destination)
	if err != nil {
		t.Fatalf("expected save settings to succeed, got %v", err)
	}
	if ui != nil {
		t.Fatalf("expected save to return nil UI, got %#v", ui)
	}
	if httpClient.calls != 1 {
		t.Fatalf("expected one HTTP request, got %d", httpClient.calls)
	}

}

// TestUpsertCreateRequest checks the create request body.
func TestUpsertCreateRequest(t *testing.T) {

	client := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusCreated, `{"data":{"id":"188181874438309840"}}`), nil
		},
	}
	ml := newTestMailerLite(t, client)
	schema := upsertTestSchema()
	ts := time.Date(2026, 5, 25, 10, 11, 12, 0, time.UTC)

	err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
		Attributes: map[string]any{
			"email":         "sam@example.com",
			"status":        "active",
			"subscribed_at": ts,
			"resubscribe":   true,
			"groups":        []any{"123", "456"},
			"fields": map[string]any{
				"name":  "Sam",
				"score": "001",
			},
		},
	}}), schema)
	if err != nil {
		t.Fatalf("expected create upsert to succeed, got %v", err)
	}

	req := client.onlyRequest(t)
	assertMailerLiteRequest(t, req, http.MethodPost, "/api/subscribers")
	assertJSONBody(t, req, map[string]any{
		"email":         "sam@example.com",
		"status":        "active",
		"subscribed_at": "2026-05-25 10:11:12",
		"resubscribe":   true,
		"groups":        []any{"123", "456"},
		"fields": map[string]any{
			"name":  "Sam",
			"score": "001",
		},
	})

}

// TestUpsertCreateRequiresCreatedStatus checks the create status code.
func TestUpsertCreateRequiresCreatedStatus(t *testing.T) {

	client := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"data":{"id":"188181874438309840"}}`), nil
		},
	}
	ml := newTestMailerLite(t, client)

	err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
		Attributes: map[string]any{"email": "sam@example.com"},
	}}), upsertTestSchema())
	if err == nil {
		t.Fatal("expected create upsert with 200 response to fail")
	}
	apiErr, ok := err.(*apiError)
	if !ok {
		t.Fatalf("expected apiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusOK {
		t.Fatalf("expected apiError status 200, got %d", apiErr.StatusCode)
	}

}

// TestUpsertUpdateRequest checks the update request body.
func TestUpsertUpdateRequest(t *testing.T) {

	client := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"data":{"id":"abc/123"}}`), nil
		},
	}
	ml := newTestMailerLite(t, client)
	schema := upsertTestSchema()

	err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
		ID: "abc/123",
		Attributes: map[string]any{
			"email":  "sam@example.com",
			"status": "active",
			"fields": map[string]any{
				"name": "Sam",
			},
		},
	}}), schema)
	if err != nil {
		t.Fatalf("expected update upsert to succeed, got %v", err)
	}

	req := client.onlyRequest(t)
	assertMailerLiteRequest(t, req, http.MethodPut, "/api/subscribers/abc%2F123")
	body := assertJSONBody(t, req, map[string]any{
		"status": "active",
		"fields": map[string]any{
			"name": "Sam",
		},
	})
	if _, ok := body["id"]; ok {
		t.Fatalf("expected update body not to contain id, got %s", string(req.body))
	}
	if _, ok := body["email"]; ok {
		t.Fatalf("expected update body not to contain email, got %s", string(req.body))
	}

}

// TestUpsertCreateRequestIncludesEmailAndFields checks the create body.
func TestUpsertCreateRequestIncludesEmailAndFields(t *testing.T) {

	client := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusCreated, `{"data":{"id":"188181874438309840"}}`), nil
		},
	}
	ml := newTestMailerLite(t, client)

	err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
		Attributes: map[string]any{
			"email": "sam@example.com",
			"fields": map[string]any{
				"name": "Sam",
			},
		},
	}}), upsertTestSchema())
	if err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	req := client.onlyRequest(t)
	assertMailerLiteRequest(t, req, http.MethodPost, "/api/subscribers")
	assertJSONBody(t, req, map[string]any{
		"email": "sam@example.com",
		"fields": map[string]any{
			"name": "Sam",
		},
	})

}

// TestUpsertCreateWithoutEmail checks create validation.
func TestUpsertCreateWithoutEmail(t *testing.T) {

	client := &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			t.Fatal("expected no HTTP request")
			return nil, nil
		},
	}
	ml := newTestMailerLite(t, client)

	err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
		Attributes: map[string]any{"status": "active"},
	}}), upsertTestSchema())
	if _, ok := err.(connectors.RecordsError); !ok {
		t.Fatalf("expected RecordsError, got %T: %v", err, err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no HTTP calls, got %d", client.calls)
	}

}

// TestServeUIUsesMultilineAPIKeyInput checks the settings input widget.
func TestServeUIUsesMultilineAPIKeyInput(t *testing.T) {

	ml := &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   &testSettingsStore{},
			HTTPClient: &testHTTPClient{},
		},
	}

	ui, err := ml.ServeUI(t.Context(), "load", nil, connectors.Destination)
	if err != nil {
		t.Fatalf("expected load settings UI to succeed, got %v", err)
	}
	if len(ui.Fields) != 1 {
		t.Fatalf("expected one settings field, got %d", len(ui.Fields))
	}
	input, ok := ui.Fields[0].(*connectors.Input)
	if !ok {
		t.Fatalf("expected API token field to be an Input, got %T", ui.Fields[0])
	}
	if input.Rows <= 1 {
		t.Fatalf("expected API token field to be multiline, got rows=%d", input.Rows)
	}
	if input.MinLength != legacyAPIKeyLength+1 {
		t.Fatalf("expected API token minimum length to reject legacy keys, got %d", input.MinLength)
	}
	if input.Type == "password" {
		t.Fatal("expected multiline API token field not to use password input type")
	}

}

// TestRecordSchemaEmailAndGroups checks the email and groups schema.
func TestRecordSchemaEmailAndGroups(t *testing.T) {

	ml := &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   &testSettingsStore{},
			HTTPClient: fieldsHTTPClient(t, nil),
		},
	}

	sourceSchema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}
	destinationSchema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
	if err != nil {
		t.Fatalf("expected destination schema, got %v", err)
	}

	sourceEmail, err := sourceSchema.Properties().ByPath("email")
	if err != nil {
		t.Fatalf("expected source email property, got %v", err)
	}
	destinationEmail, err := destinationSchema.Properties().ByPath("email")
	if err != nil {
		t.Fatalf("expected destination email property, got %v", err)
	}
	if !types.Equal(sourceEmail.Type, destinationEmail.Type) {
		t.Fatalf("expected source and destination email types to match, got %s and %s", sourceEmail.Type, destinationEmail.Type)
	}
	if sourceEmail.Nullable != destinationEmail.Nullable {
		t.Fatalf("expected source and destination email nullability to match, got %t and %t", sourceEmail.Nullable, destinationEmail.Nullable)
	}
	if _, err := sourceSchema.Properties().ByPath("groups"); err != nil {
		t.Fatalf("expected source schema to include groups, got %v", err)
	}
	if _, err := destinationSchema.Properties().ByPath("groups"); err != nil {
		t.Fatalf("expected destination schema to include groups, got %v", err)
	}
	if _, err := destinationSchema.Properties().ByPath("id"); err == nil {
		t.Fatal("expected destination schema not to include id")
	}
	if _, err := sourceSchema.Properties().ByPath("id"); err != nil {
		t.Fatalf("expected source schema to include id, got %v", err)
	}
	if got := sourceSchema.Properties().Slice()[0].Name; got != "id" {
		t.Fatalf("expected id to be the first source property, got %q", got)
	}

}

// TestRecordSchemaDynamicFields checks custom field mapping.
func TestRecordSchemaDynamicFields(t *testing.T) {

	fields := json.Value(`[
		{"id":"1","name":"Text Field","key":"text_field","type":"text"},
		{"id":"2","name":"Number Field","key":"number_field","type":"number"},
		{"id":"3","name":"Date Field","key":"date_field","type":"date"},
		{"id":"4","name":"Invalid Key","key":"invalid-key","type":"text"},
		{"id":"5","name":"Unknown Field","key":"unknown_field","type":"boolean"}
	]`)
	ml := &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   &testSettingsStore{},
			HTTPClient: fieldsHTTPClient(t, fields),
		},
	}

	for _, role := range []connectors.Role{connectors.Source, connectors.Destination} {
		schema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, role)
		if err != nil {
			t.Fatalf("expected %s schema, got %v", role, err)
		}

		assertPropertyType(t, schema, "fields.text_field", types.String())
		assertPropertyType(t, schema, "fields.number_field", types.Decimal(numberPrecision, numberScale))
		assertPropertyType(t, schema, "fields.date_field", types.Date())
		assertNoNestedField(t, schema, "invalid-key")
		assertNoNestedField(t, schema, "unknown_field")
	}

}

// TestRecordSchemaNoCustomFields checks the empty custom field case.
func TestRecordSchemaNoCustomFields(t *testing.T) {

	ml := &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   &testSettingsStore{},
			HTTPClient: fieldsHTTPClient(t, nil),
		},
	}

	for _, role := range []connectors.Role{connectors.Source, connectors.Destination} {
		schema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, role)
		if err != nil {
			t.Fatalf("expected %s schema, got %v", role, err)
		}
		assertNoProperty(t, schema, "fields")
	}

}

// TestRecordsParsing checks subscriber parsing.
func TestRecordsParsing(t *testing.T) {

	client := mailerLiteFixtureClient(t, map[string]string{
		"/api/fields":      fieldsFixture(),
		"/api/subscribers": subscribersPageFixture("188181874438309840", "sam@example.com", "2026-05-22 15:37:00", ""),
	})
	ml := newTestMailerLite(t, client)

	schema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}
	records, cursor, err := ml.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
	if err != io.EOF {
		t.Fatalf("expected EOF with final page, got cursor %q and error %v", cursor, err)
	}
	if cursor != "" {
		t.Fatalf("expected empty cursor, got %q", cursor)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}

	record := records[0]
	if record.ID != "188181874438309840" {
		t.Fatalf("unexpected record ID %q", record.ID)
	}
	if want := time.Date(2026, 5, 22, 15, 37, 0, 0, time.UTC); !record.UpdatedAt.Equal(want) {
		t.Fatalf("expected UpdatedAt %s, got %s", want, record.UpdatedAt)
	}
	if record.Err != nil {
		t.Fatalf("expected no record error, got %v", record.Err)
	}
	if got := record.Attributes["email"]; got != "sam@example.com" {
		t.Fatalf("expected email sam@example.com, got %#v", got)
	}
	groups, ok := record.Attributes["groups"].([]string)
	if !ok {
		t.Fatalf("expected groups slice, got %T %#v", record.Attributes["groups"], record.Attributes["groups"])
	}
	if len(groups) != 2 || groups[0] != "123" || groups[1] != "456" {
		t.Fatalf("expected groups [123 456], got %#v", groups)
	}
	if got := record.Attributes["sent"]; got != int32(7) {
		t.Fatalf("expected sent int32(7), got %T %#v", got, got)
	}
	if got := record.Attributes["opens_count"]; got != int32(3) {
		t.Fatalf("expected opens_count int32(3), got %T %#v", got, got)
	}
	if got := record.Attributes["clicks_count"]; got != int32(2) {
		t.Fatalf("expected clicks_count int32(2), got %T %#v", got, got)
	}
	fields, ok := record.Attributes["fields"].(map[string]any)
	if !ok {
		t.Fatalf("expected fields map, got %T", record.Attributes["fields"])
	}
	score, ok := fields["score"].(decimal.Decimal)
	if !ok {
		t.Fatalf("expected score decimal, got %T %#v", fields["score"], fields["score"])
	}
	if fields["name"] != "Sam" || score.Cmp(decimal.MustInt(1)) != 0 || fields["birthday"] != "1990-01-02" {
		t.Fatalf("unexpected fields %#v", fields)
	}

}

// TestRecordsParsingDropsUnrepresentableNumberField checks unrepresentable numbers.
func TestRecordsParsingDropsUnrepresentableNumberField(t *testing.T) {

	client := mailerLiteFixtureClient(t, map[string]string{
		"/api/fields":      fieldsFixture(),
		"/api/subscribers": subscribersPageFixtureWithScore("188181874438309840", "sam@example.com", "2026-05-22 15:37:00", "", "0.00000000000000000000000000000000000001"),
	})
	ml := newTestMailerLite(t, client)

	schema, err := ml.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}
	records, _, err := ml.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
	if err != io.EOF {
		t.Fatalf("expected EOF with final page, got %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if records[0].Err != nil {
		t.Fatalf("expected record to remain valid, got %v", records[0].Err)
	}
	fields, ok := records[0].Attributes["fields"].(map[string]any)
	if !ok {
		t.Fatalf("expected fields map, got %T", records[0].Attributes["fields"])
	}
	if fields["score"] != nil {
		t.Fatalf("expected unrepresentable score to be nil, got %T %#v", fields["score"], fields["score"])
	}

}

// TestParseNumber checks MailerLite number parsing.
func TestParseNumber(t *testing.T) {

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "nil", value: nil, want: nil},
		{name: "empty", value: "", want: nil},
		{name: "leading zeros", value: "001", want: decimal.MustInt(1)},
		{name: "zero fraction", value: "000.12", want: decimal.MustParse("0.12")},
		{name: "negative leading zeros", value: "-001.20", want: decimal.MustParse("-1.20")},
		{name: "positive exponent", value: "001e3", want: decimal.MustInt(1000)},
		{name: "negative exponent", value: "001E-2", want: decimal.MustParse("0.01")},
		{name: "not representable", value: "0.00000000000000000000000000000000000001", want: nil},
		{name: "not string", value: float64(1), want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseNumber(test.value)
			want, ok := test.want.(decimal.Decimal)
			if !ok {
				if got != nil {
					t.Fatalf("expected nil, got %T %#v", got, got)
				}
				return
			}
			gotDecimal, ok := got.(decimal.Decimal)
			if !ok {
				t.Fatalf("expected decimal, got %T %#v", got, got)
			}
			if gotDecimal.Cmp(want) != 0 {
				t.Fatalf("expected %s, got %s", want, gotDecimal)
			}
		})
	}

}

// TestRecordsPaginationAndUpdatedAt checks pagination and updatedAt filtering.
func TestRecordsPaginationAndUpdatedAt(t *testing.T) {

	client := mailerLiteFixtureClient(t, map[string]string{
		"/api/subscribers":                  subscribersPageFixture("old", "old@example.com", "2026-05-22 10:00:00", "next-page"),
		"/api/subscribers?cursor=next-page": subscribersPageFixture("new", "new@example.com", "2026-05-22 12:00:00", ""),
	})
	ml := newTestMailerLite(t, client)
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.String(), Description: "Email address, used only when creating a subscriber"},
		{Name: "updated_at", Type: types.DateTime(), Description: "Last update timestamp"},
	})

	records, cursor, err := ml.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
	if err != nil {
		t.Fatalf("expected first page without error, got %v", err)
	}
	if cursor != "next-page" {
		t.Fatalf("expected next cursor next-page, got %q", cursor)
	}
	if len(records) != 1 || records[0].ID != "old" {
		t.Fatalf("expected first page old record, got %#v", records)
	}

	records, cursor, err = ml.Records(t.Context(), connectors.TargetUser, time.Time{}, cursor, schema)
	if err != io.EOF {
		t.Fatalf("expected EOF on second page, got cursor %q and error %v", cursor, err)
	}
	if cursor != "" {
		t.Fatalf("expected empty final cursor, got %q", cursor)
	}
	if len(records) != 1 || records[0].ID != "new" {
		t.Fatalf("expected second page new record, got %#v", records)
	}

	client.requests = nil
	client.calls = 0
	updatedAt := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	records, cursor, err = ml.Records(t.Context(), connectors.TargetUser, updatedAt, "", schema)
	if err != nil {
		t.Fatalf("expected first page to return next cursor without error, got cursor %q and error %v", cursor, err)
	}
	if cursor != "next-page" {
		t.Fatalf("expected next cursor after updatedAt filtering, got %q", cursor)
	}
	if len(records) != 0 {
		t.Fatalf("expected no records after updatedAt filtering, got %#v", records)
	}
	if client.calls != 1 {
		t.Fatalf("expected one HTTP call when first page is filtered out, got %d", client.calls)
	}

}

// TestUpsertErrorMapping checks API error mapping.
func TestUpsertErrorMapping(t *testing.T) {

	tests := []struct {
		name         string
		statusCode   int
		recordsError bool
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, recordsError: true},
		{name: "not found", statusCode: http.StatusNotFound, recordsError: true},
		{name: "unprocessable entity", statusCode: http.StatusUnprocessableEntity, recordsError: true},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "too many requests", statusCode: http.StatusTooManyRequests},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &testHTTPClient{
				do: func(r *http.Request) (*http.Response, error) {
					return jsonResponse(test.statusCode, `{"message":"MailerLite test error","errors":{"email":["invalid"]}}`), nil
				},
			}
			ml := newTestMailerLite(t, client)

			err := ml.Upsert(t.Context(), connectors.TargetUser, newRecordsIterator([]connectors.Record{{
				Attributes: map[string]any{"email": "sam@example.com"},
			}}), upsertTestSchema())
			if err == nil {
				t.Fatal("expected upsert error, got nil")
			}
			recordsErr, ok := err.(connectors.RecordsError)
			if test.recordsError {
				if !ok {
					t.Fatalf("expected RecordsError, got %T: %v", err, err)
				}
				if recordsErr[0] == nil {
					t.Fatalf("expected RecordsError index 0, got %#v", recordsErr)
				}
				return
			}
			if ok {
				t.Fatalf("expected global error, got RecordsError: %v", err)
			}
		})
	}

}

// TestSaveSettingsValidation checks settings validation failures.
func TestSaveSettingsValidation(t *testing.T) {

	tests := []struct {
		name               string
		apiKey             string
		statusCode         int
		wantInvalidSetting bool
		wantHTTPCalls      int
	}{
		{name: "empty", apiKey: "", wantInvalidSetting: true},
		{name: "space", apiKey: "abc def", wantInvalidSetting: true},
		{name: "control char", apiKey: "abc\ndef", wantInvalidSetting: true},
		{name: "max length", apiKey: strings.Repeat("a", settingsAPIKeyMaxSize), statusCode: http.StatusOK, wantHTTPCalls: 1},
		{name: "too long", apiKey: strings.Repeat("a", settingsAPIKeyMaxSize+1), wantInvalidSetting: true},
		{name: "api unauthorized", apiKey: "valid-shape", statusCode: http.StatusUnauthorized, wantInvalidSetting: true, wantHTTPCalls: 1},
		{name: "api forbidden", apiKey: "valid-shape", statusCode: http.StatusForbidden, wantInvalidSetting: true, wantHTTPCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &testHTTPClient{
				do: func(r *http.Request) (*http.Response, error) {
					assertMailerLiteRequestWithAPIKey(t, capturedRequest{
						method: r.Method,
						path:   r.URL.EscapedPath(),
						header: r.Header,
					}, http.MethodGet, "/api/fields", test.apiKey)
					return jsonResponse(test.statusCode, `{"message":"MailerLite test error"}`), nil
				},
			}
			ml := newTestMailerLite(t, client)

			settings, err := json.Marshal(innerSettings{APIKey: test.apiKey})
			if err != nil {
				t.Fatalf("expected settings marshal to succeed, got %v", err)
			}
			_, err = ml.ServeUI(t.Context(), "save", settings, connectors.Destination)
			if test.wantInvalidSetting {
				if _, ok := err.(*connectors.InvalidSettingsError); !ok {
					t.Fatalf("expected InvalidSettingsError, got %T: %v", err, err)
				}
			} else if err != nil {
				t.Fatalf("expected save settings to succeed, got %v", err)
			}
			if client.calls != test.wantHTTPCalls {
				t.Fatalf("expected %d HTTP calls, got %d", test.wantHTTPCalls, client.calls)
			}
		})
	}

}

// TestLiveMailerLiteAPIKey checks the live API key against MailerLite.
func TestLiveMailerLiteAPIKey(t *testing.T) {

	apiKey := os.Getenv("KRENALIS_TEST_MAILERLITE_API_KEY")
	if apiKey == "" {
		t.Skip("the KRENALIS_TEST_MAILERLITE_API_KEY environment variable is not present")
	}
	if strings.ContainsFunc(apiKey, func(r rune) bool { return r <= ' ' || r == 0x7f }) {
		t.Fatal("expected KRENALIS_TEST_MAILERLITE_API_KEY not to contain spaces or control characters")
	}

	ml, err := testconnector.NewApplication[*MailerLite]("mailerlite", innerSettings{})
	if err != nil {
		t.Fatalf("expected NewApplication to succeed, got %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	settings, err := json.Marshal(innerSettings{APIKey: apiKey})
	if err != nil {
		t.Fatalf("expected settings marshal to succeed, got %v", err)
	}
	if _, err := ml.ServeUI(ctx, "save", settings, connectors.Destination); err != nil {
		t.Fatalf("expected MailerLite API token validation to succeed, got %v", err)
	}

	schema, err := ml.RecordSchema(ctx, connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected MailerLite fields to be readable, got %v", err)
	}
	if _, err := schema.Properties().ByPathSlice([]string{"email"}); err != nil {
		t.Fatalf("expected source schema to include email, got %v", err)
	}

	records, _, err := ml.Records(ctx, connectors.TargetUser, time.Time{}, "", schema)
	if err == io.EOF {
		if os.Getenv("KRENALIS_TEST_MAILERLITE_REQUIRE_SUBSCRIBERS") == "1" {
			t.Fatal("expected at least one MailerLite subscriber because KRENALIS_TEST_MAILERLITE_REQUIRE_SUBSCRIBERS=1")
		}
		t.Skip("the configured MailerLite account has no subscribers")
	}
	if err != nil {
		t.Fatalf("expected MailerLite subscribers to be readable, got %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected at least one MailerLite subscriber")
	}
	record := records[0]
	if record.ID == "" {
		t.Fatal("expected first MailerLite subscriber to have a non-empty Record.ID")
	}
	if record.UpdatedAt.IsZero() {
		t.Fatal("expected first MailerLite subscriber to have a non-zero UpdatedAt")
	}
	if record.Err != nil {
		t.Fatalf("expected first MailerLite subscriber to be valid, got %v", record.Err)
	}
	if record.Attributes["email"] == "" {
		t.Fatalf("expected first MailerLite subscriber to include email, got %#v", record.Attributes["email"])
	}

}

func fieldsHTTPClient(t *testing.T, fields json.Value) *testHTTPClient {
	t.Helper()
	if fields == nil {
		fields = json.Value(`[]`)
	}
	return &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/fields" {
				t.Fatalf("expected path /api/fields, got %q", r.URL.Path)
			}
			return jsonResponse(http.StatusOK, `{"data":`+string(fields)+`}`), nil
		},
	}
}

type testSettingsStore struct {
	settings json.Value
}

func (store *testSettingsStore) Load(ctx context.Context, dst any) error {
	if store.settings == nil {
		return nil
	}
	return store.settings.Unmarshal(dst)
}

func (store *testSettingsStore) Store(ctx context.Context, src any) error {
	settings, err := json.Marshal(src)
	if err != nil {
		return err
	}
	store.settings = settings
	return nil
}

type testHTTPClient struct {
	calls    int
	requests []capturedRequest
	do       func(req *http.Request) (*http.Response, error)
}

func (client *testHTTPClient) Do(req *http.Request) (*http.Response, error) {
	client.calls++
	captured := capturedRequest{
		method: req.Method,
		url:    req.URL.String(),
		path:   req.URL.EscapedPath(),
		query:  req.URL.Query(),
		header: req.Header.Clone(),
	}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		captured.body = body
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	client.requests = append(client.requests, captured)
	if client.do == nil {
		return nil, errors.New("test HTTP client has no Do function")
	}
	return client.do(req)
}

func (client *testHTTPClient) ClientSecret() (string, error) {
	return "", errors.New("test HTTP client does not support OAuth")
}

func (client *testHTTPClient) AccessToken(ctx context.Context) (string, error) {
	return "", errors.New("test HTTP client does not support OAuth")
}

func (client *testHTTPClient) GetBodyBuffer(enc connectors.ContentEncoding) *connectors.BodyBuffer {
	return connectors.GetBodyBuffer(enc, 1024)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type capturedRequest struct {
	method string
	url    string
	path   string
	query  map[string][]string
	header http.Header
	body   []byte
}

func (client *testHTTPClient) onlyRequest(t *testing.T) capturedRequest {
	t.Helper()
	if len(client.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(client.requests))
	}
	return client.requests[0]
}

func newTestMailerLite(t *testing.T, httpClient *testHTTPClient) *MailerLite {
	t.Helper()
	store := &testSettingsStore{}
	if err := store.Store(t.Context(), innerSettings{APIKey: testAPIKey}); err != nil {
		t.Fatalf("expected settings store to succeed, got %v", err)
	}
	oldAPIBaseURL := apiBaseURL
	apiBaseURL = "https://connect.mailerlite.test/api"
	t.Cleanup(func() { apiBaseURL = oldAPIBaseURL })
	return &MailerLite{
		env: &connectors.ApplicationEnv{
			Settings:   store,
			HTTPClient: httpClient,
		},
	}
}

func assertMailerLiteRequest(t *testing.T, req capturedRequest, method, path string) {
	t.Helper()
	assertMailerLiteRequestWithAPIKey(t, req, method, path, testAPIKey)
}

func assertMailerLiteRequestWithAPIKey(t *testing.T, req capturedRequest, method, path, apiKey string) {
	t.Helper()
	if req.method != method {
		t.Fatalf("expected method %s, got %s", method, req.method)
	}
	if req.path != path {
		t.Fatalf("expected path %s, got %s", path, req.path)
	}
	if auth := req.header.Get("Authorization"); auth != "Bearer "+apiKey {
		t.Fatalf("expected bearer Authorization header, got %q", auth)
	}
	if version := req.header.Get("X-Version"); version != apiVersion {
		t.Fatalf("expected X-Version %q, got %q", apiVersion, version)
	}
}

func assertJSONBody(t *testing.T, req capturedRequest, expected map[string]any) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(req.body, &got); err != nil {
		t.Fatalf("expected valid JSON body, got %v; body=%s", err, string(req.body))
	}
	if !jsonEqual(got, expected) {
		t.Fatalf("unexpected JSON body:\nexpected %#v\ngot      %#v\nraw      %s", expected, got, string(req.body))
	}
	return got
}

func assertPropertyType(t *testing.T, schema types.Type, path string, typ types.Type) {
	t.Helper()
	property, err := schema.Properties().ByPath(path)
	if err != nil {
		t.Fatalf("expected property %q, got %v", path, err)
	}
	if !types.Equal(property.Type, typ) {
		t.Fatalf("expected property %q to have type %s, got %s", path, typ, property.Type)
	}
}

func assertNoProperty(t *testing.T, schema types.Type, path string) {
	t.Helper()
	if _, err := schema.Properties().ByPath(path); err == nil {
		t.Fatalf("expected property %q not to exist", path)
	}
}

func assertNoNestedField(t *testing.T, schema types.Type, name string) {
	t.Helper()
	fields, err := schema.Properties().ByPath("fields")
	if err != nil {
		t.Fatalf("expected fields property, got %v", err)
	}
	for _, property := range fields.Type.Properties().All() {
		if property.Name == name {
			t.Fatalf("expected nested field %q not to exist", name)
		}
	}
}

func jsonEqual(a, b any) bool {
	aa, err := json.Marshal(a)
	if err != nil {
		return false
	}
	aa, err = json.Canonicalize(aa)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	bb, err = json.Canonicalize(bb)
	if err != nil {
		return false
	}
	return bytes.Equal(aa, bb)
}

func upsertTestSchema() types.Type {
	return types.Object([]types.Property{
		{Name: "email", Type: types.String(), CreateRequired: true, Description: "Email address, used only when creating a subscriber"},
		{Name: "status", Type: subscriberStatusType(), Description: "Subscriber status"},
		{Name: "subscribed_at", Type: types.DateTime(), Nullable: true, Description: "Subscription timestamp"},
		{Name: "resubscribe", Type: types.Boolean(), Description: "Resubscribe"},
		{Name: "groups", Type: types.Array(types.String()), Description: "Group IDs"},
		{Name: "fields", Type: types.Object([]types.Property{
			{Name: "name", Type: types.String(), Nullable: true, Description: "Name"},
			{Name: "score", Type: types.String().WithMaxLength(1024), Nullable: true, Description: "Score"},
		}), Description: "Subscriber custom fields"},
	})
}

type recordsIterator struct {
	records  []connectors.Record
	consumed bool
	current  int
}

func newRecordsIterator(records []connectors.Record) connectors.Records {
	return &recordsIterator{records: records}
}

func (it *recordsIterator) All() iter.Seq[connectors.Record] {
	if it.consumed {
		panic("Records.All called after records were consumed")
	}
	it.consumed = true
	return func(yield func(connectors.Record) bool) {
		for i, record := range it.records {
			it.current = i
			if !yield(record) {
				return
			}
		}
	}
}

func (it *recordsIterator) Discard(err error) {
	if err == nil {
		panic("Records.Discard called with nil error")
	}
	if !it.consumed {
		panic("Records.Discard called outside iteration")
	}
}

func (it *recordsIterator) First() connectors.Record {
	if it.consumed {
		panic("Records.First called after records were consumed")
	}
	it.consumed = true
	if len(it.records) == 0 {
		panic("Records.First called with no records")
	}
	return it.records[0]
}

func (it *recordsIterator) Peek() (connectors.Record, bool) {
	if !it.consumed {
		if len(it.records) == 0 {
			return connectors.Record{}, false
		}
		return it.records[0], true
	}
	next := it.current + 1
	if next >= len(it.records) {
		return connectors.Record{}, false
	}
	return it.records[next], true
}

func (it *recordsIterator) Postpone() {
	if !it.consumed {
		panic("Records.Postpone called outside iteration")
	}
}

func (it *recordsIterator) Same() iter.Seq[connectors.Record] {
	return it.All()
}

func mailerLiteFixtureClient(t *testing.T, fixtures map[string]string) *testHTTPClient {
	t.Helper()
	return &testHTTPClient{
		do: func(r *http.Request) (*http.Response, error) {
			key := r.URL.EscapedPath()
			if cursor := r.URL.Query().Get("cursor"); cursor != "" {
				key += "?cursor=" + cursor
			}
			body, ok := fixtures[key]
			if !ok {
				t.Fatalf("unexpected MailerLite test request %s %s", r.Method, r.URL.String())
			}
			return jsonResponse(http.StatusOK, body), nil
		},
	}
}

func fieldsFixture() string {
	return `{
		"data": [
			{"id":"1","name":"Name","key":"name","type":"text"},
			{"id":"2","name":"Score","key":"score","type":"number"},
			{"id":"3","name":"Birthday","key":"birthday","type":"date"}
		],
		"meta": {"current_page":1,"last_page":1}
	}`
}

func subscribersPageFixture(id, email, updatedAt, metaNextCursor string) string {
	return subscribersPageFixtureWithScore(id, email, updatedAt, metaNextCursor, "001")
}

func subscribersPageFixtureWithScore(id, email, updatedAt, metaNextCursor, score string) string {
	meta := `{}`
	if metaNextCursor != "" {
		meta = `{"next_cursor":` + strconv.Quote(metaNextCursor) + `}`
	}
	return `{
		"data": [{
			"id": ` + strconv.Quote(id) + `,
			"email": ` + strconv.Quote(email) + `,
			"status": "active",
			"source": "manual",
			"sent": 7,
			"opens_count": 3,
			"clicks_count": 2,
			"open_rate": 42.5,
			"click_rate": 28.5,
			"ip_address": null,
			"groups": [{"id":"123"}, {"id":"456"}],
			"subscribed_at": "2026-05-22 15:37:00",
			"unsubscribed_at": null,
			"created_at": "2026-05-22 15:37:00",
			"updated_at": ` + strconv.Quote(updatedAt) + `,
			"fields": {
				"name": "Sam",
				"score": ` + strconv.Quote(score) + `,
				"birthday": "1990-01-02"
			},
			"opted_in_at": null,
			"optin_ip": null
		}],
		"meta": ` + meta + `
	}`
}
