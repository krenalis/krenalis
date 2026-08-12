// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package activecampaign

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const (
	testAPIURL   = "https://account.activecampaign.test/api/3"
	testAPIToken = "activecampaign-test-token"
)

func TestApplicationSpec(t *testing.T) {
	app := connectors.RegisteredApplication("activecampaign")
	if app.Label != "ActiveCampaign" {
		t.Fatalf("expected ActiveCampaign label, got %q", app.Label)
	}
	if app.AsSource != nil {
		t.Fatal("expected ActiveCampaign not to declare a source capability")
	}
	if app.AsDestination == nil || app.AsDestination.Targets != connectors.TargetUser || !app.AsDestination.HasSettings {
		t.Fatalf("unexpected destination specification: %#v", app.AsDestination)
	}
	if app.AsDestination.SendingMode != connectors.None {
		t.Fatalf("expected no event sending mode, got %v", app.AsDestination.SendingMode)
	}
	if app.ReflectType() != reflect.TypeFor[*ActiveCampaign]() {
		t.Fatalf("expected registered type %v, got %v", reflect.TypeFor[*ActiveCampaign](), app.ReflectType())
	}
	if len(app.EndpointGroups) != 1 || app.EndpointGroups[0].RateLimit.RequestsPerSecond != 5 || app.EndpointGroups[0].RateLimit.Burst != 5 {
		t.Fatalf("unexpected endpoint groups: %#v", app.EndpointGroups)
	}
}

func TestRecordSchema(t *testing.T) {
	ac := &ActiveCampaign{}
	source, err := ac.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}
	destination, err := ac.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
	if err != nil {
		t.Fatalf("expected destination schema, got %v", err)
	}

	for _, name := range []string{"email", "firstName", "lastName", "phone"} {
		sourceProperty, ok := source.Properties().ByName(name)
		if !ok {
			t.Fatalf("expected source property %q", name)
		}
		destinationProperty, ok := destination.Properties().ByName(name)
		if !ok {
			t.Fatalf("expected destination property %q", name)
		}
		if !types.Equal(sourceProperty.Type, types.String()) || !types.Equal(destinationProperty.Type, types.String()) {
			t.Fatalf("expected %q to be a string", name)
		}
	}
	sourceEmail, _ := source.Properties().ByName("email")
	destinationEmail, _ := destination.Properties().ByName("email")
	if sourceEmail.CreateRequired {
		t.Fatal("expected source email not to be create-required")
	}
	if !destinationEmail.CreateRequired {
		t.Fatal("expected destination email to be create-required")
	}

	if _, err := ac.RecordSchema(t.Context(), connectors.TargetEvent, connectors.Destination); err == nil {
		t.Fatal("expected unsupported target to fail")
	}
	if _, err := ac.RecordSchema(t.Context(), connectors.TargetUser, connectors.Both); err == nil {
		t.Fatal("expected invalid role to fail")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := ac.RecordSchema(canceled, connectors.TargetUser, connectors.Destination); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled schema request, got %v", err)
	}
}

func TestServeUISaveAndLoad(t *testing.T) {
	store := &testSettingsStore{}
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != testAPIURL+"/users/me" {
			t.Fatalf("unexpected validation request %s %s", req.Method, req.URL.String())
		}
		if got := req.Header.Get("Api-Token"); got != testAPIToken {
			t.Fatalf("expected Api-Token header, got %q", got)
		}
		if got := req.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected JSON Accept header, got %q", got)
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	}}
	ac := newTestActiveCampaign(store, client)

	value, err := json.Marshal(innerSettings{
		APIURL:   " https://account.activecampaign.test/ ",
		APIToken: testAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	ui, err := ac.ServeUI(t.Context(), "save", value, connectors.Destination)
	if err != nil {
		t.Fatalf("expected settings save to succeed, got %v", err)
	}
	if ui != nil {
		t.Fatalf("expected save to return no UI, got %#v", ui)
	}
	if store.stores != 1 {
		t.Fatalf("expected one settings store, got %d", store.stores)
	}

	var saved innerSettings
	if err := store.Load(t.Context(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.APIURL != testAPIURL || saved.APIToken != testAPIToken {
		t.Fatalf("unexpected saved settings: %#v", saved)
	}

	ui, err = ac.ServeUI(t.Context(), "load", nil, connectors.Destination)
	if err != nil {
		t.Fatalf("expected settings load to succeed, got %v", err)
	}
	if len(ui.Fields) != 2 || len(ui.Buttons) != 1 {
		t.Fatalf("unexpected settings UI: %#v", ui)
	}
	urlInput, ok := ui.Fields[0].(*connectors.Input)
	if !ok || urlInput.Name != "apiURL" || urlInput.Type != "url" {
		t.Fatalf("unexpected API URL input: %#v", ui.Fields[0])
	}
	tokenInput, ok := ui.Fields[1].(*connectors.Input)
	if !ok || tokenInput.Name != "apiToken" || tokenInput.Type != "password" {
		t.Fatalf("unexpected API token input: %#v", ui.Fields[1])
	}
	var loaded innerSettings
	if err := ui.Settings.Unmarshal(&loaded); err != nil {
		t.Fatalf("expected load settings JSON, got %v", err)
	}
	if loaded != saved {
		t.Fatalf("expected loaded settings %#v, got %#v", saved, loaded)
	}
	if _, err := ac.ServeUI(t.Context(), "unknown", nil, connectors.Destination); !errors.Is(err, connectors.ErrUIEventNotExist) {
		t.Fatalf("expected ErrUIEventNotExist, got %v", err)
	}
}

func TestServeUISettingsStoreFailures(t *testing.T) {
	t.Run("empty load", func(t *testing.T) {
		store := &testSettingsStore{}
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			t.Fatal("expected load not to make an HTTP request")
			return nil, nil
		}}
		ac := newTestActiveCampaign(store, client)
		ui, err := ac.ServeUI(t.Context(), "load", nil, connectors.Destination)
		if err != nil {
			t.Fatalf("expected empty settings to load, got %v", err)
		}
		var settings innerSettings
		if err := ui.Settings.Unmarshal(&settings); err != nil {
			t.Fatalf("expected empty settings JSON, got %v", err)
		}
		if settings != (innerSettings{}) {
			t.Fatalf("expected zero settings, got %#v", settings)
		}
	})

	t.Run("load failure", func(t *testing.T) {
		wantErr := errors.New("load failed")
		ac := newTestActiveCampaign(&testSettingsStore{loadErr: wantErr}, &testHTTPClient{})
		if _, err := ac.ServeUI(t.Context(), "load", nil, connectors.Destination); !errors.Is(err, wantErr) {
			t.Fatalf("expected load error, got %v", err)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		wantErr := errors.New("store failed")
		store := &testSettingsStore{storeErr: wantErr}
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{}`), nil
		}}
		ac := newTestActiveCampaign(store, client)
		value, _ := json.Marshal(innerSettings{APIURL: testAPIURL, APIToken: testAPIToken})
		if _, err := ac.ServeUI(t.Context(), "save", value, connectors.Destination); !errors.Is(err, wantErr) {
			t.Fatalf("expected store error, got %v", err)
		}
		if client.calls != 1 || store.stores != 0 {
			t.Fatalf("expected validation but no successful store, got %d requests and %d stores", client.calls, store.stores)
		}
	})
}

func TestSaveSettingsValidation(t *testing.T) {
	tests := []struct {
		name     string
		settings innerSettings
	}{
		{name: "empty URL", settings: innerSettings{APIToken: testAPIToken}},
		{name: "HTTP URL", settings: innerSettings{APIURL: "http://account.example", APIToken: testAPIToken}},
		{name: "credentials in URL", settings: innerSettings{APIURL: "https://user:pass@account.example", APIToken: testAPIToken}},
		{name: "invalid port", settings: innerSettings{APIURL: "https://account.example:not-a-port", APIToken: testAPIToken}},
		{name: "unexpected path", settings: innerSettings{APIURL: "https://account.example/api/2", APIToken: testAPIToken}},
		{name: "query", settings: innerSettings{APIURL: "https://account.example?x=1", APIToken: testAPIToken}},
		{name: "fragment", settings: innerSettings{APIURL: "https://account.example/#x", APIToken: testAPIToken}},
		{name: "empty token", settings: innerSettings{APIURL: "https://account.example"}},
		{name: "header control", settings: innerSettings{APIURL: "https://account.example", APIToken: "secret\nvalue"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &testSettingsStore{}
			client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
				t.Fatal("expected no HTTP request")
				return nil, nil
			}}
			ac := newTestActiveCampaign(store, client)
			value, err := json.Marshal(test.settings)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ac.ServeUI(t.Context(), "save", value, connectors.Destination)
			var invalid *connectors.InvalidSettingsError
			if !errors.As(err, &invalid) {
				t.Fatalf("expected InvalidSettingsError, got %T: %v", err, err)
			}
			if client.calls != 0 || store.stores != 0 {
				t.Fatalf("expected no request or store, got %d requests and %d stores", client.calls, store.stores)
			}
			if strings.Contains(err.Error(), test.settings.APIToken) && test.settings.APIToken != "" {
				t.Fatal("expected validation error not to contain the API token")
			}
		})
	}
}

func TestSaveSettingsRejectsUnauthorizedWithoutStoring(t *testing.T) {
	store := &testSettingsStore{}
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"message":"token `+testAPIToken+` rejected"}`), nil
	}}
	ac := newTestActiveCampaign(store, client)
	value, _ := json.Marshal(innerSettings{APIURL: testAPIURL, APIToken: testAPIToken})
	_, err := ac.ServeUI(t.Context(), "save", value, connectors.Destination)
	var invalid *connectors.InvalidSettingsError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected InvalidSettingsError, got %T: %v", err, err)
	}
	if store.stores != 0 {
		t.Fatalf("expected settings not to be stored, got %d stores", store.stores)
	}
	if strings.Contains(err.Error(), testAPIToken) {
		t.Fatal("expected API error not to contain the token")
	}
}

func TestUpsertCreate(t *testing.T) {
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusCreated, `{"contact":{"id":"12"}}`), nil
	}}
	ac := newConfiguredTestActiveCampaign(t, client)
	records := newTestRecords([]connectors.Record{{Attributes: map[string]any{
		"email":     "sam@example.com",
		"firstName": "Sam",
		"lastName":  "Example",
		"phone":     "+390000000",
		"ignored":   "not in the connector schema",
	}}})

	if err := ac.Upsert(t.Context(), connectors.TargetUser, records, types.Type{}); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	req := client.onlyRequest(t)
	assertRequest(t, req, http.MethodPost, "/api/3/contact/sync")
	assertJSONBody(t, req.body, map[string]any{"contact": map[string]any{
		"email":     "sam@example.com",
		"firstName": "Sam",
		"lastName":  "Example",
		"phone":     "+390000000",
	}})
	if req.header.Get("Idempotency-Key") != "" {
		t.Fatal("expected no provider-unsupported idempotency header")
	}
}

func TestUpsertUpdate(t *testing.T) {
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"contact":{"id":"42"}}`), nil
	}}
	ac := newConfiguredTestActiveCampaign(t, client)

	err := ac.Upsert(t.Context(), connectors.TargetUser, newTestRecords([]connectors.Record{{
		ID:         "42",
		Attributes: map[string]any{"firstName": "Ada", "phone": ""},
	}}), types.Type{})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	req := client.onlyRequest(t)
	assertRequest(t, req, http.MethodPut, "/api/3/contacts/42")
	assertJSONBody(t, req.body, map[string]any{"contact": map[string]any{"firstName": "Ada", "phone": ""}})
}

func TestUpsertLocalAndProviderErrors(t *testing.T) {
	t.Run("missing create email", func(t *testing.T) {
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			t.Fatal("expected no HTTP request")
			return nil, nil
		}}
		ac := newConfiguredTestActiveCampaign(t, client)
		err := ac.Upsert(t.Context(), connectors.TargetUser, newTestRecords([]connectors.Record{{Attributes: map[string]any{"firstName": "Sam"}}}), types.Type{})
		assertSingleRecordError(t, err)
	})

	t.Run("invalid update id", func(t *testing.T) {
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			t.Fatal("expected no HTTP request")
			return nil, nil
		}}
		ac := newConfiguredTestActiveCampaign(t, client)
		err := ac.Upsert(t.Context(), connectors.TargetUser, newTestRecords([]connectors.Record{{ID: "not-an-id"}}), types.Type{})
		assertSingleRecordError(t, err)
	})

	t.Run("validation response", func(t *testing.T) {
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusBadRequest, `{"message":"sam@example.com `+testAPIToken+`"}`), nil
		}}
		ac := newConfiguredTestActiveCampaign(t, client)
		err := ac.Upsert(t.Context(), connectors.TargetUser, newTestRecords([]connectors.Record{{Attributes: map[string]any{"email": "sam@example.com"}}}), types.Type{})
		assertSingleRecordError(t, err)
		if strings.Contains(err.Error(), "sam@example.com") || strings.Contains(err.Error(), testAPIToken) {
			t.Fatalf("expected private response data not to appear in error: %v", err)
		}
	})

	t.Run("server response", func(t *testing.T) {
		client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusInternalServerError, `{}`), nil
		}}
		ac := newConfiguredTestActiveCampaign(t, client)
		err := ac.Upsert(t.Context(), connectors.TargetUser, newTestRecords([]connectors.Record{{Attributes: map[string]any{"email": "sam@example.com"}}}), types.Type{})
		if err == nil {
			t.Fatal("expected server response to fail")
		}
		if _, ok := err.(connectors.RecordsError); ok {
			t.Fatalf("expected request-level error, got %v", err)
		}
	})
}

func TestRecordsFinalPageAndInclusiveUpdatedAt(t *testing.T) {
	responseBody := contactsResponse(t, []contact{
		{ID: "10", Email: "first@example.com", FirstName: "First", CreatedAt: "2026-08-12T10:00:00+02:00", UpdatedAt: "2026-08-12T10:01:00+02:00"},
		{ID: "11", Email: "second@example.com", LastName: "Second", Phone: "+3901", CreatedAt: "2026-08-12T10:02:00+02:00"},
	}, "2")
	var body *trackedBody
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("limit") != "100" || req.URL.Query().Get("orders[id]") != "ASC" {
			t.Fatalf("unexpected pagination query %q", req.URL.RawQuery)
		}
		if _, ok := req.URL.Query()["filters[updated_after]"]; ok {
			t.Fatal("expected exact updatedAt filtering to be performed locally")
		}
		body = &trackedBody{Reader: strings.NewReader(responseBody)}
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}}
	ac := newConfiguredTestActiveCampaign(t, client)
	updatedAt := time.Date(2026, 8, 12, 8, 1, 0, 0, time.UTC)

	records, cursor, err := ac.Records(t.Context(), connectors.TargetUser, updatedAt, "", types.Type{})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected final-page EOF, got cursor %q and error %v", cursor, err)
	}
	if cursor != "" || len(records) != 2 {
		t.Fatalf("expected two final records and empty cursor, got %d and %q", len(records), cursor)
	}
	if records[0].ID != "10" || !records[0].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected first record: %#v", records[0])
	}
	wantSecondTime := time.Date(2026, 8, 12, 10, 2, 0, 0, time.FixedZone("", 2*60*60))
	if records[1].ID != "11" || !records[1].UpdatedAt.Equal(wantSecondTime) {
		t.Fatalf("expected cdate fallback for second record, got %#v", records[1])
	}
	if records[1].Attributes["lastName"] != "Second" || records[1].Attributes["phone"] != "+3901" {
		t.Fatalf("unexpected second record attributes: %#v", records[1].Attributes)
	}
	if body == nil || !body.closed {
		t.Fatal("expected response body to be closed")
	}
}

func TestRecordsSkipsEmptyFilteredPages(t *testing.T) {
	firstPage := make([]contact, contactsPageLimit)
	for i := range firstPage {
		firstPage[i] = contact{
			ID:        strconv.Itoa(i + 1),
			Email:     "old@example.com",
			CreatedAt: "2026-08-11T09:00:00Z",
			UpdatedAt: "2026-08-11T10:00:00Z",
		}
	}
	firstResponse := contactsResponse(t, firstPage, "101")
	secondResponse := contactsResponse(t, []contact{{
		ID:        "101",
		Email:     "new@example.com",
		CreatedAt: "2026-08-12T09:00:00Z",
		UpdatedAt: "2026-08-12T10:00:00Z",
	}}, "1")
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		switch req.URL.Query().Get("id_greater") {
		case "":
			return jsonResponse(http.StatusOK, firstResponse), nil
		case "100":
			return jsonResponse(http.StatusOK, secondResponse), nil
		default:
			t.Fatalf("unexpected cursor query %q", req.URL.RawQuery)
			return nil, nil
		}
	}}
	ac := newConfiguredTestActiveCampaign(t, client)

	records, cursor, err := ac.Records(
		t.Context(),
		connectors.TargetUser,
		time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		"",
		types.Type{},
	)
	if !errors.Is(err, io.EOF) || cursor != "" {
		t.Fatalf("expected final EOF after internal pagination, got cursor %q and error %v", cursor, err)
	}
	if len(records) != 1 || records[0].ID != "101" {
		t.Fatalf("expected only new contact 101, got %#v", records)
	}
	if client.calls != 2 {
		t.Fatalf("expected two requests, got %d", client.calls)
	}
}

func TestRecordsReturnsOpaqueCursor(t *testing.T) {
	page := make([]contact, contactsPageLimit)
	for i := range page {
		page[i] = contact{
			ID:        strconv.Itoa(i + 201),
			CreatedAt: "2026-08-12T09:00:00Z",
			UpdatedAt: "2026-08-12T10:00:00Z",
		}
	}
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("id_greater") != "200" {
			t.Fatalf("expected id_greater=200, got %q", req.URL.RawQuery)
		}
		return jsonResponse(http.StatusOK, contactsResponse(t, page, "101")), nil
	}}
	ac := newConfiguredTestActiveCampaign(t, client)
	records, cursor, err := ac.Records(t.Context(), connectors.TargetUser, time.Time{}, "200", types.Type{})
	if err != nil {
		t.Fatalf("expected non-final page, got %v", err)
	}
	if len(records) != contactsPageLimit || cursor != "300" {
		t.Fatalf("expected 100 records and cursor 300, got %d and %q", len(records), cursor)
	}
}

func TestRecordsErrors(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
		body   string
	}{
		{name: "invalid cursor", cursor: "not-a-number"},
		{name: "invalid contact id", body: contactsResponse(t, []contact{{ID: "bad", CreatedAt: "2026-08-12T10:00:00Z"}}, "1")},
		{name: "missing timestamp", body: contactsResponse(t, []contact{{ID: "1"}}, "1")},
		{name: "invalid timestamp", body: contactsResponse(t, []contact{{ID: "1", UpdatedAt: "not-a-time"}}, "1")},
		{name: "malformed JSON", body: `{"contacts":[`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, test.body), nil
			}}
			ac := newConfiguredTestActiveCampaign(t, client)
			_, _, err := ac.Records(t.Context(), connectors.TargetUser, time.Time{}, test.cursor, types.Type{})
			if err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("expected records error, got %v", err)
			}
			if test.cursor != "" && client.calls != 0 {
				t.Fatalf("expected invalid cursor not to make a request, got %d", client.calls)
			}
		})
	}

	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"contacts":[],"meta":{"total":"0"}}`), nil
	}}
	ac := newConfiguredTestActiveCampaign(t, client)
	records, cursor, err := ac.Records(t.Context(), connectors.TargetUser, time.Time{}, "", types.Type{})
	if !errors.Is(err, io.EOF) || records != nil || cursor != "" {
		t.Fatalf("expected empty EOF, got records=%#v cursor=%q err=%v", records, cursor, err)
	}
}

func TestDecodeResponseLimit(t *testing.T) {
	var dst any
	err := decodeResponse(strings.NewReader(strings.Repeat(" ", maxResponseBodySize+1)), &dst)
	if err == nil || err.Error() != "ActiveCampaign response is too large" {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}

func contactsResponse(t *testing.T, contacts []contact, total any) string {
	t.Helper()
	payload := map[string]any{
		"contacts": contacts,
		"meta":     map[string]any{"total": total},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("expected fixture marshal to succeed, got %v", err)
	}
	return string(data)
}

func assertSingleRecordError(t *testing.T, err error) {
	t.Helper()
	recordsErr, ok := err.(connectors.RecordsError)
	if !ok {
		t.Fatalf("expected RecordsError, got %T: %v", err, err)
	}
	if len(recordsErr) != 1 || recordsErr[0] == nil {
		t.Fatalf("expected only record index 0 to fail, got %#v", recordsErr)
	}
}

func assertRequest(t *testing.T, req capturedRequest, method, path string) {
	t.Helper()
	if req.method != method || req.path != path {
		t.Fatalf("expected %s %s, got %s %s", method, path, req.method, req.path)
	}
	if got := req.header.Get("Api-Token"); got != testAPIToken {
		t.Fatalf("expected Api-Token header, got %q", got)
	}
	if got := req.header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON Content-Type, got %q", got)
	}
}

func assertJSONBody(t *testing.T, body []byte, want any) {
	t.Helper()
	var got any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("expected JSON body, got %v: %s", err, body)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ = json.Canonicalize(gotJSON)
	wantJSON, _ = json.Canonicalize(wantJSON)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("unexpected JSON body\nwant: %s\n got: %s", wantJSON, gotJSON)
	}
}

type testSettingsStore struct {
	settings json.Value
	loadErr  error
	storeErr error
	stores   int
}

func (store *testSettingsStore) Load(ctx context.Context, dst any) error {
	if store.loadErr != nil {
		return store.loadErr
	}
	if store.settings == nil {
		return nil
	}
	return store.settings.Unmarshal(dst)
}

func (store *testSettingsStore) Store(ctx context.Context, src any) error {
	if store.storeErr != nil {
		return store.storeErr
	}
	value, err := json.Marshal(src)
	if err != nil {
		return err
	}
	store.settings = value
	store.stores++
	return nil
}

type capturedRequest struct {
	method string
	url    string
	path   string
	header http.Header
	body   []byte
}

type testHTTPClient struct {
	calls    int
	requests []capturedRequest
	do       func(*http.Request) (*http.Response, error)
}

func (client *testHTTPClient) Do(req *http.Request) (*http.Response, error) {
	client.calls++
	captured := capturedRequest{
		method: req.Method,
		url:    req.URL.String(),
		path:   req.URL.EscapedPath(),
		header: req.Header.Clone(),
	}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		captured.body = body
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	client.requests = append(client.requests, captured)
	if client.do == nil {
		return nil, errors.New("test HTTP client has no Do function")
	}
	return client.do(req)
}

func (client *testHTTPClient) ClientSecret() (string, error) {
	return "", errors.New("OAuth is not supported")
}

func (client *testHTTPClient) AccessToken(ctx context.Context) (string, error) {
	return "", errors.New("OAuth is not supported")
}

func (client *testHTTPClient) GetBodyBuffer(enc connectors.ContentEncoding) *connectors.BodyBuffer {
	return connectors.GetBodyBuffer(enc, 1024)
}

func (client *testHTTPClient) onlyRequest(t *testing.T) capturedRequest {
	t.Helper()
	if len(client.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(client.requests))
	}
	return client.requests[0]
}

func newTestActiveCampaign(store *testSettingsStore, client *testHTTPClient) *ActiveCampaign {
	return &ActiveCampaign{env: &connectors.ApplicationEnv{Settings: store, HTTPClient: client}}
}

func newConfiguredTestActiveCampaign(t *testing.T, client *testHTTPClient) *ActiveCampaign {
	t.Helper()
	store := &testSettingsStore{}
	if err := store.Store(t.Context(), innerSettings{APIURL: testAPIURL, APIToken: testAPIToken}); err != nil {
		t.Fatal(err)
	}
	store.stores = 0
	return newTestActiveCampaign(store, client)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (body *trackedBody) Close() error {
	body.closed = true
	return nil
}

type testRecords struct {
	records  []connectors.Record
	consumed bool
}

func newTestRecords(records []connectors.Record) *testRecords {
	return &testRecords{records: records}
}

func (records *testRecords) First() connectors.Record {
	if records.consumed || len(records.records) == 0 {
		panic("invalid Records.First call")
	}
	records.consumed = true
	return records.records[0]
}

func (records *testRecords) All() iter.Seq[connectors.Record] {
	panic("unexpected Records.All call")
}

func (records *testRecords) Same() iter.Seq[connectors.Record] {
	panic("unexpected Records.Same call")
}

func (records *testRecords) Discard(error) {
	panic("unexpected Records.Discard call")
}

func (records *testRecords) Peek() (connectors.Record, bool) {
	panic("unexpected Records.Peek call")
}

func (records *testRecords) Postpone() {
	panic("unexpected Records.Postpone call")
}
