// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package commerceready

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const (
	testAPIKey = "commerce-ready-test-key"
	testAPIURL = "https://shop.example.com/api/v13/"
)

type recordsIterator struct {
	items      []connectors.Record
	firstCalls int
}

// All rejects iteration intended for batch endpoints.
func (it *recordsIterator) All() iter.Seq[connectors.Record] {
	panic("connectors/commerceready: expected Records.First, got All")
}

// Discard rejects discarding during a single-record operation.
func (it *recordsIterator) Discard(err error) {
	panic("connectors/commerceready: expected Records.First, got Discard")
}

// First consumes one record and leaves later records available.
func (it *recordsIterator) First() connectors.Record {
	it.firstCalls++
	record := it.items[0]
	it.items = it.items[1:]
	return record
}

// Peek rejects unnecessary lookahead for a single-record endpoint.
func (it *recordsIterator) Peek() (connectors.Record, bool) {
	panic("connectors/commerceready: expected Records.First, got Peek")
}

// Postpone rejects postponement during a single-record operation.
func (it *recordsIterator) Postpone() {
	panic("connectors/commerceready: expected Records.First, got Postpone")
}

// Same rejects iteration intended for batch endpoints.
func (it *recordsIterator) Same() iter.Seq[connectors.Record] {
	panic("connectors/commerceready: expected Records.First, got Same")
}

type testBody struct {
	io.Reader
	closed bool
}

// Close records response body closure.
func (b *testBody) Close() error {
	b.closed = true
	return nil
}

type testHTTPClient struct {
	connectors.HTTPClient
	do     func(*http.Request) (*http.Response, error)
	calls  int
	bodies []*testBody
}

// Do delegates to a controlled response and tracks response body closure.
func (client *testHTTPClient) Do(req *http.Request) (*http.Response, error) {
	defer req.Body.Close()
	client.calls++
	response, err := client.do(req)
	if err != nil {
		return nil, err
	}
	body := &testBody{Reader: response.Body}
	response.Body = body
	client.bodies = append(client.bodies, body)
	return response, nil
}

// GetBodyBuffer uses the production request body implementation.
func (client *testHTTPClient) GetBodyBuffer(encoding connectors.ContentEncoding) *connectors.BodyBuffer {
	return connectors.GetBodyBuffer(encoding, 1024)
}

type testSettingsStore struct {
	value    json.Value
	loadErr  error
	storeErr error
	stores   int
}

// Load decodes saved settings or returns an injected failure.
func (s *testSettingsStore) Load(ctx context.Context, dst any) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	return s.value.Unmarshal(dst)
}

// Store replaces settings only after successful serialization.
func (s *testSettingsStore) Store(ctx context.Context, src any) error {
	if s.storeErr != nil {
		return s.storeErr
	}
	value, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.value = value
	s.stores++
	return nil
}

// TestFindCopiesCustomers checks that yielded values survive subsequent reads.
func TestFindCopiesCustomers(t *testing.T) {
	body := `{"status":"ok","customers":[` +
		`{"id":1,"code":"A","collection":8,"collections":[1,2],"loginTime":"first",` +
		`"billingAddress":{"companyName":"First"},"shippingAddress":{"name":"First"}},` +
		`{"id":2,"code":"B","collection":9,"collections":[3],"loginTime":"second",` +
		`"billingAddress":{"companyName":"Second"},"shippingAddress":{"name":"Second"}},` +
		`{"id":3}]}`
	c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		return testResponse(200, body), nil
	})
	settings := innerSettings{APIURL: testAPIURL, APIKey: testAPIKey}
	var items []customer
	for item, err := range c.find(t.Context(), settings, map[string]any{}) {
		if err != nil {
			t.Fatalf("expected customers, got %v", err)
		}
		items = append(items, item)
	}
	if len(items) != 3 {
		t.Fatalf("expected three customers, got %d", len(items))
	}
	for i, want := range []struct {
		code       string
		collection int32
		login      string
		company    string
		values     []int32
	}{
		{"A", 8, "first", "First", []int32{1, 2}},
		{"B", 9, "second", "Second", []int32{3}},
	} {
		item := items[i]
		if item.Code != want.code || item.Collection == nil || *item.Collection != want.collection ||
			item.LoginTime == nil || *item.LoginTime != want.login || item.BillingAddress.CompanyName != want.company ||
			item.ShippingAddress == nil || item.ShippingAddress.Name != want.company ||
			!reflect.DeepEqual(item.Collections, want.values) {
			t.Fatalf("expected preserved customer %d, got %#v", i+1, item)
		}
	}
	if !reflect.DeepEqual(items[2], customer{ID: 3}) {
		t.Fatalf("expected reset third customer, got %#v", items[2])
	}
	if client.calls != 1 || len(client.bodies) != 1 || !client.bodies[0].closed {
		t.Fatalf("expected one completed request and closed body, got %#v", client)
	}
}

// TestFindLazyRequestAndEarlyStop checks resource lifetime around iteration.
func TestFindLazyRequestAndEarlyStop(t *testing.T) {
	prefix := `{"status":"ok","customers":[{"id":1,"code":"A"},`
	c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		response := testResponse(200, "")
		response.Body = io.NopCloser(io.MultiReader(
			strings.NewReader(prefix), iotest.ErrReader(errors.New("unread response tail")),
		))
		return response, nil
	})
	settings := innerSettings{APIURL: testAPIURL, APIKey: testAPIKey}
	customers := c.find(t.Context(), settings, map[string]any{})
	if client.calls != 0 {
		t.Fatalf("expected no request before iteration, got %d calls", client.calls)
	}
	count := 0
	for item, err := range customers {
		if err != nil {
			t.Fatalf("expected first customer before reading tail, got %v", err)
		}
		count++
		if item.ID != 1 || item.Code != "A" {
			t.Fatalf("expected first customer, got %#v", item)
		}
		break
	}
	if count != 1 || client.calls != 1 || len(client.bodies) != 1 || !client.bodies[0].closed {
		t.Fatalf("expected one customer and a closed body after break, got %d customers and %#v", count, client)
	}
}

// TestFindTerminalError checks that errors are yielded once and end iteration.
func TestFindTerminalError(t *testing.T) {
	item := `{"id":1,"code":"A"}`
	tests := []struct {
		name      string
		status    int
		body      string
		customers int
		apiError  bool
		message   string
	}{
		{"HTTP error", 403, "private response", 0, false, ""},
		{"invalid customer", 200, `{"status":"ok","customers":[{"code":42}]}`, 0, false, ""},
		{"truncated array", 200, `{"status":"ok","customers":[` + item + `,`, 1, false, ""},
		{"late API error", 200, `{"customers":[` + item + `],"status":"error","error":{"type":"AccessDenied"}}`, 1, true, ""},
		{"API error without customers", 200, `{"status":"error","error":{"type":"AccessDenied"}}`, 0, true, ""},
		{"oversized JSON value", 200, `{"status":"ok","customers":[` + item + `],"padding":"` + strings.Repeat("x", maxResponseBytes) + `"}`, 1, false, "size limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				return testResponse(tt.status, tt.body), nil
			})
			settings := innerSettings{APIURL: testAPIURL, APIKey: testAPIKey}
			count, failures := 0, 0
			for item, err := range c.find(t.Context(), settings, map[string]any{}) {
				if err != nil {
					failures++
					_, isAPIError := errors.AsType[*apiError](err)
					if failures != 1 || isAPIError != tt.apiError || strings.Contains(err.Error(), "private response") {
						t.Fatalf("expected one sanitized API error %t, got %v", tt.apiError, err)
					}
					if tt.message != "" && !strings.Contains(err.Error(), tt.message) {
						t.Fatalf("expected error containing %q, got %v", tt.message, err)
					}
					if !reflect.DeepEqual(item, customer{}) {
						t.Fatalf("expected zero customer with error, got %#v", item)
					}
					continue
				}
				if failures != 0 {
					t.Fatalf("expected no customer after an error, got %#v", item)
				}
				count++
			}
			if count != tt.customers || failures != 1 {
				t.Fatalf("expected %d customers and one error, got %d customers and %d errors", tt.customers, count, failures)
			}
			if client.calls != 1 || len(client.bodies) != 1 || !client.bodies[0].closed {
				t.Fatalf("expected one request and a closed body, got %#v", client)
			}
		})
	}
}

// TestHTTPFailures checks response limits, status handling, and error privacy.
func TestHTTPFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, testAPIKey},
		{"forbidden", 403, "private@example.com"},
		{"not found", 404, "private@example.com"},
		{"redirect", 302, "private@example.com"},
		{"bad request", 400, "private@example.com"},
		{"unavailable", 503, "private@example.com"},
		{"malformed JSON", 200, `{"private@example.com":`},
		{"unknown status", 200, `{"status":"private@example.com"}`},
		{"missing API error", 200, `{"status":"error"}`},
		{"oversized response", 200, strings.Repeat("x", maxResponseBytes+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				return testResponse(tt.status, tt.body), nil
			})
			schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
			_, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
			if err != nil {
				if strings.Contains(err.Error(), testAPIKey) || strings.Contains(err.Error(), "private@example.com") {
					t.Fatalf("expected a private error, got %v", err)
				}
				if client.calls != 1 {
					t.Fatalf("expected one HTTP call, got %d", client.calls)
				}
				return
			}
			t.Fatal("expected an HTTP or response error, got nil")
		})
	}
}

// TestLiveRecords checks schemas and pagination against an opt-in shop.
func TestLiveRecords(t *testing.T) {

	apiURL := os.Getenv("COMMERCE_READY_API_URL")
	apiKey := os.Getenv("COMMERCE_READY_API_KEY")
	timeZone := os.Getenv("COMMERCE_READY_TIME_ZONE")
	if apiURL == "" || apiKey == "" || timeZone == "" {
		t.Skip("set COMMERCE_READY_API_URL, COMMERCE_READY_API_KEY, and COMMERCE_READY_TIME_ZONE for live reads")
	}
	c, err := testconnector.NewApplication[*CommerceReady]("commerceready", innerSettings{
		APIURL: apiURL, APIKey: apiKey, TimeZone: timeZone,
	})
	if err != nil {
		t.Fatalf("expected connector construction, got %v", err)
	}
	schema, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cursor := ""
	previousID := ""
	for range 2 {
		records, next, err := c.Records(ctx, connectors.TargetUser, time.Time{}, cursor, schema)
		isFinalPage := false
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("expected a customer page or EOF, got %v", err)
			}
			isFinalPage = true
		}
		for _, record := range records {
			if record.ID == "" || record.ID == previousID || record.UpdatedAt.IsZero() {
				t.Fatal("expected new customer identity and update time, got invalid metadata")
			}
			previousID = record.ID
		}
		if isFinalPage {
			break
		}
		if next == "" || next == cursor {
			t.Fatal("expected an advancing cursor, got no progress")
		}
		cursor = next
	}

}

// TestRecordSchema checks role requirements and excluded private fields.
func TestRecordSchema(t *testing.T) {
	c, _, _ := newTestConnector(t, nil)
	for _, role := range []connectors.Role{connectors.Source, connectors.Destination} {
		schema, err := c.RecordSchema(t.Context(), connectors.TargetUser, role)
		if err != nil {
			t.Fatalf("expected customer schema, got %v", err)
		}
		for _, name := range []string{
			"updateTime", "password", "encryptedPassword", "cart", "wishList", "fullName",
		} {
			if schema.Properties().ContainsName(name) {
				t.Fatalf("expected %s excluded, got a schema property", name)
			}
		}
		canLogin, ok := schema.Properties().ByName("canLogin")
		if !ok || !canLogin.Type.IsBoolean() || canLogin.Nullable ||
			canLogin.CreateRequired || canLogin.UpdateRequired {
			t.Fatalf("expected optional boolean canLogin for role %v, got %#v", role, canLogin)
		}
		if schema.Properties().ContainsName("billingAddress") {
			t.Fatal("expected flattened billing properties, got billingAddress")
		}
		for _, name := range []string{
			"firstName", "lastName", "personalCode", "companyName", "companyCode", "companyCode2",
			"street1", "street2", "city", "postalCode", "stateProv", "country", "phoneNumber",
			"mobileNumber", "faxNumber", "email",
		} {
			if !schema.Properties().ContainsName(name) {
				t.Fatalf("expected top-level billing field %s for role %v, got no property", name, role)
			}
		}
		firstName, hasFirstName := schema.Properties().ByName("firstName")
		email, hasEmail := schema.Properties().ByName("email")
		isDestination := role == connectors.Destination
		if !hasFirstName || firstName.CreateRequired != isDestination {
			t.Fatalf("expected first name requirements for role %v, got %#v", role, firstName)
		}
		if !hasEmail || email.CreateRequired != isDestination {
			t.Fatalf("expected address email requirements %t, got %#v", isDestination, email)
		}
		for path, property := range schema.Properties().WalkAll() {
			if property.UpdateRequired {
				t.Fatalf("expected optional update field %s for role %v, got UpdateRequired", path, role)
			}
		}
		if schema.Properties().ContainsName("creationTime") != (role == connectors.Source) {
			t.Fatalf("expected source-only creation time, got role %v", role)
		}
	}
}

// TestRecordSchemaBounds exercises provider lengths and numeric limits.
func TestRecordSchemaBounds(t *testing.T) {

	c, _, _ := newTestConnector(t, nil)
	schema, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
	if err != nil {
		t.Fatalf("expected customer schema, got %v", err)
	}
	tests := []struct {
		path  string
		value any
		valid bool
	}{
		{"firstName", strings.Repeat("è", 25), true},
		{"firstName", strings.Repeat("è", 26), false},
		{"firstName", "", false},
		{"email", "a@b", false},
		{"email", "a@b.c", true},
		{"email", "a.b+_-= ?^#@b.c", false},
		{"email", "a.b+_-=?^#@b.c", true},
		{"email", "a@b.c\n", false},
		{"email", strings.Repeat("a", 121), false},
		{"country", "IT", true},
		{"country", "ITA", false},
		{"code", strings.Repeat("a", 32), true},
		{"code", strings.Repeat("a", 33), false},
		{"locale", "it-IT", true},
		{"locale", "i", false},
		{"group", 1, true},
		{"group", 255, true},
		{"group", 256, false},
		{"group", 0, false},
		{"collection", 2147483647, true},
		{"collection", 0, false},
		{"collections", []any{1, 2}, true},
		{"collections", []any{0}, false},
	}
	for _, tt := range tests {
		property, err := schema.Properties().ByPath(tt.path)
		if err != nil {
			t.Fatalf("expected property %s, got %v", tt.path, err)
		}
		raw, err := json.Marshal(tt.value)
		if err != nil {
			t.Fatalf("expected fixture JSON, got %v", err)
		}
		_, err = types.Decode[any](strings.NewReader(string(raw)), property.Type)
		if (err == nil) != tt.valid {
			t.Errorf("expected %s validity %t, got %v", tt.path, tt.valid, err)
		}
	}

	collections, _ := schema.Properties().ByName("collections")
	for _, count := range []int{100, 101} {
		values := make([]int, count)
		for i := range values {
			values[i] = i + 1
		}
		raw, err := json.Marshal(values)
		if err != nil {
			t.Fatalf("expected collections JSON, got %v", err)
		}
		_, err = types.Decode[any](strings.NewReader(string(raw)), collections.Type)
		if (err == nil) != (count == 100) {
			t.Errorf("expected collection count %d validity %t, got %v", count, count == 100, err)
		}
	}

}

// TestRecordsBodySizeLimit checks the limit including trailing whitespace.
func TestRecordsBodySizeLimit(t *testing.T) {
	for _, size := range []int{maxResponseBytes - 1, maxResponseBytes, maxResponseBytes + 1} {
		prefix := `{"status":"ok","customers":[]}`
		body := prefix + strings.Repeat(" ", size-len(prefix))
		c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
			return testResponse(200, body), nil
		})
		schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
		records, cursor, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
		if err != nil {
			if size <= maxResponseBytes {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected EOF for %d bytes, got %v", size, err)
				}
			}
			if size > maxResponseBytes {
				if errors.Is(err, io.EOF) || !strings.Contains(err.Error(), "size limit") {
					t.Fatalf("expected size limit error for %d bytes, got %v", size, err)
				}
			}
		}
		if err == nil {
			t.Fatalf("expected EOF or size limit error for %d bytes, got nil", size)
		}
		if len(records) != 0 || cursor != "" {
			t.Fatalf("expected no records or cursor, got %d records and %q", len(records), cursor)
		}
	}
}

// TestRecordsFilteredPages checks inclusive filtering and empty-page skipping.
func TestRecordsFilteredPages(t *testing.T) {
	for _, offset := range []time.Duration{0, time.Microsecond} {
		t.Run(offset.String(), func(t *testing.T) {

			c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				var payload struct {
					Conditions struct {
						After int `json:"after"`
					} `json:"conditions"`
					Fields []string   `json:"fields"`
					Order  []string   `json:"order"`
					First  json.Value `json:"first"`
					Limit  json.Value `json:"limit"`
				}
				err := json.Decode(req.Body, &payload)
				if err != nil {
					t.Fatalf("expected a find request, got %v", err)
				}
				if payload.First != nil || payload.Limit != nil || !reflect.DeepEqual(payload.Order, []string{"id"}) ||
					!reflect.DeepEqual(payload.Fields, []string{"id", "updateTime", "code"}) {
					t.Fatalf("expected find fields and order without first or limit, got %#v", payload)
				}
				if _, ok := req.Header["Idempotency-Key"]; !ok {
					t.Fatal("expected a replayable read, got no idempotency marker")
				}
				if payload.Conditions.After == 0 {
					customers := make([]map[string]any, 200)
					for i := range customers {
						customers[i] = map[string]any{"id": i + 1, "updateTime": "2026-01-01 00:00:00"}
					}
					body, err := json.Marshal(map[string]any{"status": "ok", "customers": customers})
					if err != nil {
						t.Fatalf("expected page fixture, got %v", err)
					}
					return testResponse(200, string(body)), nil
				}
				if payload.Conditions.After != 200 {
					t.Fatalf("expected after 200, got %d", payload.Conditions.After)
				}
				body := `{"status":"ok","customers":[` +
					`{"id":201,"updateTime":"2026-07-01 12:00:00","code":"NEW"}]}`
				return testResponse(200, body), nil
			})

			minimum := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC).Add(offset)
			schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
			records, cursor, err := c.Records(t.Context(), connectors.TargetUser, minimum, "", schema)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected EOF after filtered pages, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected EOF, got nil")
			}
			expected := 0
			if offset == 0 {
				expected = 1
			}
			if len(records) != expected || cursor != "" || client.calls != 2 {
				t.Fatalf("expected (%d records, empty cursor, two calls), got (%d, %q, %d)",
					expected, len(records), cursor, client.calls)
			}
			if len(records) > 0 && (records[0].ID != "201" || !records[0].UpdatedAt.Equal(minimum)) {
				t.Fatalf("expected real identity and inclusive timestamp, got %#v", records[0])
			}

		})
	}
}

// TestRecordsPagination checks variable page sizes, cursors, and final pages.
func TestRecordsPagination(t *testing.T) {
	tests := []struct {
		name  string
		pages []int
	}{
		{"empty", []int{0}},
		{"short final page", []int{80}},
		{"large final page", []int{230}},
		{"100 then final page", []int{100, 130}},
		{"200 then final page", []int{200, 30}},
		{"multiple pages", []int{100, 100, 30}},
		{"exactly 100", []int{100, 0}},
		{"exactly 200", []int{200, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			page := 0
			after := 0
			c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				var payload struct {
					Conditions struct {
						After int `json:"after"`
					} `json:"conditions"`
					First json.Value `json:"first"`
					Limit json.Value `json:"limit"`
					Order []string   `json:"order"`
				}
				err := json.Decode(req.Body, &payload)
				if err != nil {
					t.Fatalf("expected find JSON, got %v", err)
				}
				if payload.First != nil || payload.Limit != nil || payload.Conditions.After != after ||
					!reflect.DeepEqual(payload.Order, []string{"id"}) {
					t.Fatalf("expected ID order and after %d without first or limit, got %#v", after, payload)
				}
				if page >= len(tt.pages) {
					t.Fatalf("expected %d pages, got an additional request", len(tt.pages))
				}
				customers := make([]map[string]any, tt.pages[page])
				for i := range customers {
					after += 2
					customers[i] = map[string]any{
						"id": after, "updateTime": "2026-01-01 12:00:00", "code": "A",
					}
				}
				page++
				body, err := json.Marshal(map[string]any{"status": "ok", "customers": customers})
				if err != nil {
					t.Fatalf("expected fixture JSON, got %v", err)
				}
				return testResponse(200, string(body)), nil
			})

			cursor := ""
			expectedID := 0
			schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
			for i, size := range tt.pages {
				records, next, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, cursor, schema)
				isFinalPage := i == len(tt.pages)-1
				if err != nil {
					if !errors.Is(err, io.EOF) || !isFinalPage {
						t.Fatalf("expected EOF only on the final page, got %v on page %d", err, i)
					}
				}
				if err == nil && isFinalPage {
					t.Fatal("expected EOF on the final page, got nil")
				}
				if len(records) != size {
					t.Fatalf("expected %d records, got %d", size, len(records))
				}
				for _, record := range records {
					expectedID += 2
					if record.ID != strconv.Itoa(expectedID) {
						t.Fatalf("expected customer ID %d, got %q", expectedID, record.ID)
					}
				}
				expectedCursor := ""
				if !isFinalPage {
					expectedCursor = strconv.Itoa(expectedID)
				}
				if next != expectedCursor {
					t.Fatalf("expected cursor %q, got %q", expectedCursor, next)
				}
				cursor = next
			}
			if client.calls != len(tt.pages) {
				t.Fatalf("expected %d requests, got %d", len(tt.pages), client.calls)
			}

		})
	}
}

// TestRecordsProjection checks nested fields, nulls, and timestamp conversion.
func TestRecordsProjection(t *testing.T) {

	c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		var request struct {
			Fields []string `json:"fields"`
		}
		err := json.Decode(req.Body, &request)
		if err != nil {
			t.Fatalf("expected find request, got %v", err)
		}
		if !reflect.DeepEqual(request.Fields, []string{
			"id", "updateTime", "creationTime", "loginTime", "collection", "collections",
			"shippingAddress", "billingAddress",
		}) {
			t.Fatalf("expected billing fields requested as an object, got %#v", request.Fields)
		}
		body := `{"status":"ok","customers":[{"id":7,"updateTime":"2026-10-01 12:00:00.123456",` +
			`"email":"wrong@example.com",` +
			`"creationTime":"2026-01-01 12:00:00","loginTime":null,"collection":null,` +
			`"collections":[1,2],"shippingAddress":null,"billingAddress":` + string(testBillingJSON()) + `}]}`
		return testResponse(200, body), nil
	})
	schema := types.Object([]types.Property{
		{Name: "creationTime", Type: types.DateTime()},
		{Name: "loginTime", Type: types.DateTime(), Nullable: true},
		{Name: "collection", Type: types.Int(32), Nullable: true},
		{Name: "collections", Type: types.Array(types.Int(32))},
		{
			Name: "shippingAddress", Nullable: true,
			Type: types.Object([]types.Property{{Name: "name", Type: types.String()}}),
		},
		{Name: "email", Type: types.String()},
		{Name: "firstName", Type: types.String()},
	})

	records, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected final projection page, got %v", err)
		}
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if !records[0].UpdatedAt.Equal(time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)) {
		t.Fatalf("expected microsecond update time in Europe/Rome, got %v", records[0].UpdatedAt)
	}
	attributes := records[0].Attributes
	if attributes["loginTime"] != nil || attributes["shippingAddress"] != nil || attributes["collection"] != nil {
		t.Fatalf("expected nullable values, got %#v", attributes)
	}
	created, ok := attributes["creationTime"].(time.Time)
	if !ok || !created.Equal(time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected winter shop timestamp, got %v", attributes["creationTime"])
	}
	if !reflect.DeepEqual(attributes["collections"], []any{1, 2}) {
		t.Fatalf("expected integer collection IDs, got %#v", attributes["collections"])
	}
	if attributes["email"] != "ada@example.com" {
		t.Fatalf("expected flattened email, got %#v", attributes["email"])
	}
	if attributes["firstName"] != "Ada" {
		t.Fatalf("expected flattened first name, got %#v", attributes["firstName"])
	}

}

// TestRecordsReadsCustomersIncrementally checks customer IDs before reading more
// data.
func TestRecordsReadsCustomersIncrementally(t *testing.T) {
	prefix := `{"status":"ok","customers":[{"id":0,"updateTime":"2026-01-01 12:00:00","code":"A"},`
	c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		response := testResponse(200, "")
		response.Body = io.NopCloser(io.MultiReader(
			strings.NewReader(prefix), iotest.ErrReader(errors.New("unread response tail")),
		))
		return response, nil
	})
	schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
	_, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
	if err != nil {
		if !strings.Contains(err.Error(), "customer IDs out of order") {
			t.Fatalf("expected first customer's ID error, got %v", err)
		}
		return
	}
	t.Fatal("expected first customer's ID error, got nil")
}

// TestRecordsRejectsInvalidBillingAddress checks typed address decoding.
func TestRecordsRejectsInvalidBillingAddress(t *testing.T) {
	tests := []struct {
		name   string
		fields string
	}{
		{"array address", `"billingAddress":[]`},
		{"string address", `"billingAddress":"invalid"`},
		{"invalid email", `"billingAddress":{"email":42}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				body := `{"status":"ok","customers":[{"id":1,"updateTime":"2026-01-01 12:00:00",` +
					tt.fields + `}]}`
				return testResponse(200, body), nil
			})
			schema := types.Object([]types.Property{{Name: "email", Type: types.String()}})
			_, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
			if err != nil {
				if errors.Is(err, io.EOF) {
					t.Fatalf("expected billing address error, got %v", err)
				}
				return
			}
			t.Fatal("expected billing address error, got nil")
		})
	}
}

// TestRecordsRejectsInvalidData checks customer identity and response decoding.
func TestRecordsRejectsInvalidData(t *testing.T) {
	tests := []string{
		`{"status":"ok"}`,
		`{"status":"ok","customers":null}`,
		`{"status":"ok","customers":[{"updateTime":"2026-01-01 12:00:00","code":"A"}]}`,
		`{"status":"ok","customers":[{"id":-1,"updateTime":"2026-01-01 12:00:00","code":"A"}]}`,
		`{"status":"ok","customers":[{"id":2147483648,"updateTime":"2026-01-01 12:00:00","code":"A"}]}`,
		`{"status":"ok","customers":[{"id":1,"updateTime":"private@example.com","code":"A"}]}`,
		`{"status":"ok","customers":[{"id":1,"code":"A"}]}`,
		`{"status":"ok","customers":[{"id":1,"updateTime":"2026-01-01 12:00:00","code":42}]}`,
		`{"status":"ok","customers":[{"id":1,"updateTime":"2026-01-01 12:00:00","code":"A"},` +
			`{"id":1,"updateTime":"2026-01-01 12:00:00","code":"A"}]}`,
	}
	for i, body := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				return testResponse(200, body), nil
			})
			schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
			_, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
			if err != nil {
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "private@example.com") {
					t.Fatalf("expected a sanitized data error, got %v", err)
				}
				return
			}
			t.Fatal("expected invalid data error, got nil")
		})
	}
}

// TestRecordsRejectsInvalidInput checks invalid customer cursors.
func TestRecordsRejectsInvalidInput(t *testing.T) {
	c, _, client := newTestConnector(t, nil)
	schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
	for _, cursor := range []string{"0", "-1", "2147483648", "private@example.com"} {
		_, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, cursor, schema)
		if err != nil {
			if strings.Contains(err.Error(), cursor) {
				t.Fatalf("expected a sanitized cursor error, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid cursor error, got nil")
	}
	if client.calls != 0 {
		t.Fatalf("expected no requests for invalid input, got %d", client.calls)
	}
}

// TestRecordsRejectsInvalidTimestamps checks errors from timestamp conversion.
func TestRecordsRejectsInvalidTimestamps(t *testing.T) {
	tests := []struct {
		name     string
		property string
		fields   string
	}{
		{"creation timestamp", "creationTime", `"creationTime":"sensitive-fixture"`},
		{"login timestamp", "loginTime", `"loginTime":"sensitive-fixture"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				body := `{"status":"ok","customers":[{"id":1,"updateTime":"2026-01-01 12:00:00",` +
					tt.fields + `}]}`
				return testResponse(200, body), nil
			})
			source, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
			if err != nil {
				t.Fatalf("expected source schema, got %v", err)
			}
			property, _ := source.Properties().ByName(tt.property)
			schema := types.Object([]types.Property{property})
			records, cursor, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
			if err != nil {
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "sensitive-fixture") {
					t.Fatalf("expected sanitized timestamp conversion error, got %v", err)
				}
				if records != nil || cursor != "" {
					t.Fatalf("expected no partial records, got %#v and %q", records, cursor)
				}
				return
			}
			t.Fatal("expected timestamp conversion error, got nil")
		})
	}
}

// TestRecordsResponseCompletion checks errors after a complete customer.
func TestRecordsResponseCompletion(t *testing.T) {
	item := `{"id":1,"updateTime":"2026-01-01 12:00:00","code":"A"}`
	valid := `{"status":"ok","customers":[` + item + `]}`
	tests := []struct {
		name      string
		body      string
		readError bool
		wantError bool
	}{
		{"status last", `{"customers":[` + item + `],"status":"ok"}`, false, false},
		{"unknown members", `{"metadata":[{},[1,2]],"customers":[` + item + `],"status":"ok"}`, false, false},
		{"late API error", `{"customers":[` + item + `],"status":"error","error":{"type":"AccessDenied"}}`, false, true},
		{"truncated array", `{"status":"ok","customers":[` + item + `,`, false, true},
		{"truncated envelope", strings.TrimSuffix(valid, "}"), false, true},
		{"trailing JSON", valid + `{}`, false, true},
		{"duplicate status", strings.TrimSuffix(valid, "}") + `,"status":"ok"}`, false, true},
		{"body read error", valid, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				response := testResponse(200, tt.body)
				if tt.readError {
					response.Body = io.NopCloser(io.MultiReader(
						strings.NewReader(tt.body), iotest.ErrReader(errors.New("private response data")),
					))
				}
				return response, nil
			})
			schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
			records, cursor, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", schema)
			if err != nil {
				if tt.wantError {
					if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "private response data") {
						t.Fatalf("expected sanitized response error, got %v", err)
					}
					if records != nil || cursor != "" {
						t.Fatalf("expected discarded partial results, got %#v and %q", records, cursor)
					}
					return
				}
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected EOF, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected EOF or response error, got nil")
			}
			if len(records) != 1 || records[0].Attributes["code"] != "A" || cursor != "" {
				t.Fatalf("expected one final customer, got %#v and %q", records, cursor)
			}
		})
	}
}

// TestRecordsReturnsValues leaves attribute constraints to Krenalis.
func TestRecordsReturnsValues(t *testing.T) {
	location, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatalf("expected shop time zone, got %v", err)
	}
	collections := make([]any, 101)
	for i := range collections {
		collections[i] = 1
	}
	tests := []struct {
		name     string
		property string
		fields   string
		expected any
	}{
		{"code length", "code", `"code":"` + strings.Repeat("x", 33) + `"`, strings.Repeat("x", 33)},
		{"unicode name length", "firstName", `"billingAddress":{"firstName":"` + strings.Repeat("è", 26) + `"}`, strings.Repeat("è", 26)},
		{"email length", "email", `"billingAddress":{"email":"` + strings.Repeat("x", 121) + `"}`, strings.Repeat("x", 121)},
		{"missing billing address", "email", `"code":"A"`, ""},
		{"null billing address", "email", `"billingAddress":null`, ""},
		{"missing email", "email", `"billingAddress":{}`, ""},
		{"locale length", "locale", `"locale":"i"`, "i"},
		{"group minimum", "group", `"group":0`, 0},
		{"group maximum", "group", `"group":256`, 256},
		{"collection minimum", "collection", `"collection":0`, 0},
		{"array length", "collections", `"collections":[` + strings.Repeat("1,", 100) + `1]`, collections},
		{"array element", "collections", `"collections":[0]`, []any{0}},
		{"country", "country", `"billingAddress":{"country":"XX"}`, "XX"},
		{"shipping country", "shippingAddress", `"shippingAddress":{"country":"XX"}`, map[string]any{"name": "", "street1": "", "street2": "", "city": "", "postalCode": "", "stateProv": "", "country": "XX"}},
		{"creation year", "creationTime", `"creationTime":"0000-01-01 00:00:00"`, time.Date(0, 1, 1, 0, 0, 0, 0, location)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				body := `{"status":"ok","customers":[{"id":1,"updateTime":"2026-01-01 12:00:00",` +
					tt.fields + `}]}`
				return testResponse(200, body), nil
			})
			source, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
			if err != nil {
				t.Fatalf("expected source schema, got %v", err)
			}
			property, _ := source.Properties().ByName(tt.property)
			records, cursor, err := c.Records(
				t.Context(), connectors.TargetUser, time.Time{}, "", types.Object([]types.Property{property}),
			)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected EOF, got %v", err)
				}
			}
			if err == nil {
				t.Fatal("expected EOF, got nil")
			}
			if len(records) != 1 || cursor != "" {
				t.Fatalf("expected one final record, got %#v and %q", records, cursor)
			}
			value := records[0].Attributes[tt.property]
			if expected, ok := tt.expected.(time.Time); ok {
				got, ok := value.(time.Time)
				if !ok || !got.Equal(expected) {
					t.Fatalf("expected %s = %v, got %v", tt.property, expected, value)
				}
				return
			}
			if !reflect.DeepEqual(value, tt.expected) {
				t.Fatalf("expected %s = %#v, got %#v", tt.property, tt.expected, value)
			}
		})
	}
}

// TestRecordsReusesCustomer checks that omitted and null fields do not retain
// values.
func TestRecordsReusesCustomer(t *testing.T) {
	body := `{"status":"ok","customers":[` +
		`{"id":1,"updateTime":"2026-01-01 12:00:00","code":"A","canLogin":true,` +
		`"collection":8,"collections":[1,2],"loginTime":"2025-12-01 12:00:00",` +
		`"billingAddress":{"email":"a@example.com"},"shippingAddress":{"name":"First",` +
		`"street1":"1 Example St","street2":"","city":"Rome","postalCode":"00100","country":"IT","stateProv":"RM"}},` +
		`{"id":2,"updateTime":"2026-01-01 12:00:00","billingAddress":{"email":"b@example.com"}},` +
		`{"id":3,"updateTime":"2026-01-01 12:00:00","code":null,"canLogin":false,` +
		`"collection":null,"collections":null,"loginTime":null,"shippingAddress":null,` +
		`"billingAddress":{"email":"c@example.com"}}]}`
	c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		return testResponse(200, body), nil
	})
	source, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source schema, got %v", err)
	}
	var properties []types.Property
	for _, name := range []string{
		"code", "canLogin", "collection", "collections", "loginTime", "shippingAddress", "email",
	} {
		property, _ := source.Properties().ByName(name)
		properties = append(properties, property)
	}
	records, _, err := c.Records(t.Context(), connectors.TargetUser, time.Time{}, "", types.Object(properties))
	if err != nil {
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected EOF, got nil")
	}
	if len(records) != 3 {
		t.Fatalf("expected three records, got %d", len(records))
	}
	first := records[0].Attributes
	if first["code"] != "A" || first["canLogin"] != true || first["collection"] != 8 ||
		!reflect.DeepEqual(first["collections"], []any{1, 2}) || first["email"] != "a@example.com" {
		t.Fatalf("expected unchanged first customer values, got %#v", first)
	}
	shipping, ok := first["shippingAddress"].(map[string]any)
	if !ok || shipping["name"] != "First" {
		t.Fatalf("expected unchanged first shipping address, got %#v", first["shippingAddress"])
	}
	login, ok := first["loginTime"].(time.Time)
	if !ok || login.IsZero() {
		t.Fatalf("expected first login timestamp, got %#v", first["loginTime"])
	}
	for i, record := range records[1:] {
		attributes := record.Attributes
		if attributes["code"] != "" || attributes["canLogin"] != false || attributes["collection"] != nil ||
			attributes["loginTime"] != nil || attributes["shippingAddress"] != nil ||
			!reflect.DeepEqual(attributes["collections"], []any{}) || attributes["email"] != string(rune('b'+i))+"@example.com" {
			t.Fatalf("expected reset customer values, got %#v", attributes)
		}
	}
}

// TestRegistrationAndHTTPPolicy checks capabilities and endpoint policies.
func TestRegistrationAndHTTPPolicy(t *testing.T) {

	spec := connectors.RegisteredApplication("commerceready")
	if spec.AsSource.Targets != connectors.TargetUser || spec.AsDestination.Targets != connectors.TargetUser ||
		spec.AsDestination.SendingMode != connectors.None {
		t.Fatalf("expected source and destination users only, got %#v", spec)
	}
	c, _, _ := newTestConnector(t, nil)
	if _, ok := any(c).(connectors.EventSender); ok {
		t.Fatal("expected no EventSender, got an event implementation")
	}

	group := spec.EndpointGroups[0]
	if group.Patterns != nil {
		t.Fatalf("expected HTTP policy for every configured API path, got %#v", group.Patterns)
	}

	response := testResponse(503, "")
	response.Header.Set("X-Wait-Seconds", "12")
	reason, duration := group.RetryPolicy["503"](response, 0)
	if reason != connectors.RateLimited || duration != 12*time.Second {
		t.Fatalf("expected rate limiting for 12 seconds, got %v for %v", reason, duration)
	}

}

// TestRequestAPIURL checks the base URL and complete method name.
func TestRequestAPIURL(t *testing.T) {
	for _, apiURL := range []string{
		"https://shop.example.com/custom/api/v14/",
		"https://shop.example.com/custom/api/v14",
		"https://shop.example.com/custom/api/v14///",
	} {
		for _, operation := range []string{"find", "create", "update"} {
			t.Run(operation, func(t *testing.T) {
				expectedURL := "https://shop.example.com/custom/api/v14/commerce.customers." + operation
				c, store, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
					if req.URL.String() != expectedURL {
						t.Fatalf("expected request URL %s, got %s", expectedURL, req.URL)
					}
					return testResponse(200, `{"status":"ok"}`), nil
				})
				submitted := json.Value(
					`{"apiURL":"` + apiURL + `","apiKey":"` + testAPIKey + `","timeZone":"UTC"}`,
				)
				_, err := c.ServeUI(t.Context(), "save", submitted, connectors.Source)
				if err != nil {
					t.Fatalf("expected settings to save, got %v", err)
				}
				var settings innerSettings
				err = store.Load(t.Context(), &settings)
				if err != nil {
					t.Fatalf("expected stored settings, got %v", err)
				}
				response, err := c.request(t.Context(), settings, operation, map[string]any{})
				if err != nil {
					t.Fatalf("expected API request, got %v", err)
				}
				response.Body.Close()
				if client.calls != 1 {
					t.Fatalf("expected one request, got %d", client.calls)
				}
			})
		}
	}
}

// TestRequestCancellation checks propagation to the HTTP request context.
func TestRequestCancellation(t *testing.T) {
	c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	schema := types.Object([]types.Property{{Name: "code", Type: types.String()}})
	_, _, err := c.Records(ctx, connectors.TargetUser, time.Time{}, "", schema)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
		return
	}
	t.Fatal("expected cancellation, got nil")
}

// TestSettingsLifecycle checks hidden keys, replacements, and role changes.
func TestSettingsLifecycle(t *testing.T) {

	c, store, client := newTestConnector(t, nil)
	ui, err := c.ServeUI(t.Context(), "load", nil, connectors.Source)
	if err != nil {
		t.Fatalf("expected settings UI, got %v", err)
	}
	if strings.Contains(string(ui.Settings), testAPIKey) {
		t.Fatal("expected the saved key hidden, got plaintext in UI settings")
	}
	if _, ok := ui.Settings.Get([]string{"apiKey"}); ok {
		t.Fatal("expected an absent key input value, got apiKey in UI settings")
	}
	urlInput, ok := ui.Fields[0].(*connectors.Input)
	if !ok || urlInput.Name != "apiURL" || urlInput.Label != "API URL" || urlInput.Placeholder != testAPIURL {
		t.Fatalf("expected an API URL input, got %#v", ui.Fields[0])
	}
	keyInput, ok := ui.Fields[1].(*connectors.Input)
	if !ok || keyInput.Type != "password" {
		t.Fatalf("expected a password input, got %#v", ui.Fields[1])
	}
	timeZoneSelect, ok := ui.Fields[2].(*connectors.Select)
	if !ok || timeZoneSelect.Name != "timeZone" {
		t.Fatalf("expected a time zone selector, got %#v", ui.Fields[2])
	}
	timeZones := map[string]bool{}
	for _, option := range timeZoneSelect.Options {
		name, ok := option.Value.(string)
		if !ok || option.Text != name || name == "" || name == "Local" {
			t.Fatalf("expected an explicit IANA time zone option, got %#v", option)
		}
		_, err := time.LoadLocation(name)
		if err != nil {
			t.Fatalf("expected time zone %q supported by Go, got %v", name, err)
		}
		timeZones[name] = true
	}
	for _, name := range []string{"UTC", "Europe/Rome", "America/New_York", "Asia/Tokyo"} {
		if !timeZones[name] {
			t.Fatalf("expected time zone %q in selector, got no option", name)
		}
	}
	_, err = c.ServeUI(t.Context(), "save", ui.Settings, connectors.Destination)
	if err != nil {
		t.Fatalf("expected untouched settings to preserve the key across roles, got %v", err)
	}
	var saved innerSettings
	err = store.value.Unmarshal(&saved)
	if err != nil {
		t.Fatalf("expected stored settings, got %v", err)
	}
	if saved.APIKey != testAPIKey || saved.TimeZone != "Europe/Rome" {
		t.Fatalf("expected the existing key and time zone, got %#v", saved)
	}

	_, err = c.ServeUI(t.Context(), "save", json.Value(
		`{"apiURL":"https://shop.example.com/custom/api/v14",`+
			`"apiKey":"replacement-key","timeZone":"UTC"}`,
	), connectors.Source)
	if err != nil {
		t.Fatalf("expected key replacement, got %v", err)
	}
	err = store.value.Unmarshal(&saved)
	if err != nil {
		t.Fatalf("expected replaced settings, got %v", err)
	}
	if saved.APIKey != "replacement-key" || saved.TimeZone != "UTC" ||
		saved.APIURL != "https://shop.example.com/custom/api/v14/" {
		t.Fatalf("expected replaced key, API URL and UTC, got %#v", saved)
	}
	if client.calls != 0 {
		t.Fatalf("expected local settings validation, got %d HTTP calls", client.calls)
	}

}

// TestSettingsNewConnection checks empty settings and the first saved key.
func TestSettingsNewConnection(t *testing.T) {

	c, store, _ := newTestConnector(t, nil)
	store.value = json.Value(`{}`)
	ui, err := c.ServeUI(t.Context(), "load", nil, connectors.Source)
	if err != nil {
		t.Fatalf("expected empty settings UI, got %v", err)
	}
	if len(ui.Fields) != 3 || len(ui.Buttons) != 1 || ui.Buttons[0].Event != "save" {
		t.Fatalf("expected three settings and a save button, got %#v", ui)
	}

	_, err = c.ServeUI(t.Context(), "save", json.Value(
		`{"apiURL":"https://shop.example.com/api/v13/","timeZone":"Europe/Rome"}`,
	), connectors.Source)
	if err != nil {
		if _, ok := errors.AsType[*connectors.InvalidSettingsError](err); !ok {
			t.Fatalf("expected a missing key settings error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected a missing key error, got nil")
	}

	_, err = c.ServeUI(t.Context(), "save", json.Value(
		`{"apiURL":"https://shop.example.com/api/v13/","apiKey":"new-key","timeZone":"Europe/Rome"}`,
	), connectors.Source)
	if err != nil {
		t.Fatalf("expected new settings to save, got %v", err)
	}
	if store.stores != 1 {
		t.Fatalf("expected one successful save, got %d", store.stores)
	}

}

// TestSettingsRejectInvalidInput checks rejection without persistence or leaks.
func TestSettingsRejectInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"malformed JSON", `{`},
		{"array", `[]`},
		{"null", `null`},
		{"wrong URL type", `{"apiURL":4,"apiKey":"key","timeZone":"UTC"}`},
		{"missing URL", `{"apiKey":"key","timeZone":"UTC"}`},
		{"insecure URL", `{"apiURL":"http://shop.example.com/api/v13/","apiKey":"key","timeZone":"UTC"}`},
		{"URL credentials", `{"apiURL":"https://private@example.com/api/v13/","apiKey":"key","timeZone":"UTC"}`},
		{"invalid escape", `{"apiURL":"https://shop.example.com/%zz/","apiKey":"key","timeZone":"UTC"}`},
		{"oversized URL", `{"apiURL":"https://shop.example.com/` + strings.Repeat("x", 2048) + `/","apiKey":"key","timeZone":"UTC"}`},
		{"URL query", `{"apiURL":"https://shop.example.com/api/v13/?key=private","apiKey":"key","timeZone":"UTC"}`},
		{"URL fragment", `{"apiURL":"https://shop.example.com/api/v13/#private","apiKey":"key","timeZone":"UTC"}`},
		{"empty URL fragment", `{"apiURL":"https://shop.example.com/api/v13/#","apiKey":"key","timeZone":"UTC"}`},
		{"explicit key clearing", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"","timeZone":"UTC"}`},
		{"null key", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":null,"timeZone":"UTC"}`},
		{"numeric key", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":12,"timeZone":"UTC"}`},
		{"key with newline", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"private\nkey","timeZone":"UTC"}`},
		{"key with spaces", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"private key","timeZone":"UTC"}`},
		{"oversized key", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"` + strings.Repeat("x", maxAPIKeyBytes+1) + `","timeZone":"UTC"}`},
		{"missing time zone", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"key"}`},
		{"host-local time zone", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"key","timeZone":"Local"}`},
		{"unknown time zone", `{"apiURL":"https://shop.example.com/api/v13/","apiKey":"key","timeZone":"Unknown/Place"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store, _ := newTestConnector(t, nil)
			before := string(store.value)
			_, err := c.ServeUI(t.Context(), "save", json.Value(tt.value), connectors.Destination)
			if err != nil {
				if _, ok := errors.AsType[*connectors.InvalidSettingsError](err); !ok {
					t.Fatalf("expected InvalidSettingsError, got %T", err)
				}
				if store.stores != 0 || string(store.value) != before {
					t.Fatal("expected rejected settings unchanged, got a stored value")
				}
				if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), testAPIKey) {
					t.Fatalf("expected sanitized settings error, got %v", err)
				}
				return
			}
			t.Fatal("expected invalid settings, got nil")
		})
	}
}

// TestSettingsStoreFailures checks store errors and unsupported UI events.
func TestSettingsStoreFailures(t *testing.T) {

	c, store, _ := newTestConnector(t, nil)
	want := errors.New("settings store unavailable")
	store.loadErr = want
	_, err := c.ServeUI(t.Context(), "load", nil, connectors.Source)
	if err != nil {
		if !errors.Is(err, want) {
			t.Fatalf("expected load failure, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected load failure, got nil")
	}

	store.loadErr = nil
	store.storeErr = want
	_, err = c.ServeUI(t.Context(), "save", json.Value(
		`{"apiURL":"https://shop.example.com/api/v13/","apiKey":"key","timeZone":"UTC"}`,
	), connectors.Source)
	if err != nil {
		if !errors.Is(err, want) {
			t.Fatalf("expected storage failure, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected storage failure, got nil")
	}

	_, err = c.ServeUI(t.Context(), "unknown", nil, connectors.Source)
	if err != nil {
		if !errors.Is(err, connectors.ErrUIEventNotExist) {
			t.Fatalf("expected unknown UI event, got %v", err)
		}
		return
	}
	t.Fatal("expected unknown UI event, got nil")

}

// TestUpsert checks create/update payloads and single-record consumption.
func TestUpsert(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		canLogin bool
		billing  bool
	}{
		{"create with login disabled", "", false, true},
		{"update with login disabled", "123", false, false},
		{"update with login enabled", "123", true, false},
		{"update billing address", "123", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			attributes := map[string]any{"code": "NEW", "shippingAddress": nil, "canLogin": tt.canLogin}
			expectedCustomer := map[string]any{"code": "NEW", "shippingAddress": nil, "canLogin": tt.canLogin}
			expectedFields := 3
			if tt.billing {
				var billing map[string]any
				err := testBillingJSON().Unmarshal(&billing)
				if err != nil {
					t.Fatalf("expected a billing address fixture, got %v", err)
				}
				for name, value := range billing {
					attributes[name] = value
				}
				expectedCustomer["billingAddress"] = billing
				expectedFields += len(billing)
			}
			c, _, _ := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				operation := "create"
				if tt.id != "" {
					operation = "update"
				}
				if req.URL.String() != testAPIURL+"commerce.customers."+operation {
					t.Fatalf("expected %s endpoint, got %s", operation, req.URL.Path)
				}
				if _, ok := req.Header["Idempotency-Key"]; ok {
					t.Fatal("expected non-replayed writes, got an idempotency marker")
				}
				var payload map[string]json.Value
				err := json.Decode(req.Body, &payload)
				if err != nil {
					t.Fatalf("expected customer payload, got %v", err)
				}
				_, hasID := payload["id"]
				if hasID != (tt.id != "") || (hasID && string(payload["id"]) != "123") {
					t.Fatalf("expected numeric ID only for update, got %#v", payload)
				}
				var customer map[string]any
				err = payload["customer"].Unmarshal(&customer)
				if err != nil {
					t.Fatalf("expected customer JSON, got %v", err)
				}
				if !reflect.DeepEqual(customer, expectedCustomer) {
					t.Fatalf("expected mapped fields and explicit null, got %#v", customer)
				}
				return testResponse(200, `{"status":"ok","id":123}`), nil
			})

			iterator := &recordsIterator{items: []connectors.Record{
				{ID: tt.id, Attributes: attributes},
				{ID: "456", Attributes: map[string]any{"code": "LATER"}},
			}}
			schema, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
			if err != nil {
				t.Fatalf("expected customer schema, got %v", err)
			}
			err = c.Upsert(t.Context(), connectors.TargetUser, iterator, schema)
			if err != nil {
				t.Fatalf("expected successful upsert, got %v", err)
			}
			if iterator.firstCalls != 1 || len(iterator.items) != 1 || iterator.items[0].ID != "456" {
				t.Fatalf("expected exactly one consumed record, got %#v", iterator)
			}
			if len(attributes) != expectedFields || attributes["code"] != "NEW" {
				t.Fatalf("expected unchanged attributes, got %#v", attributes)
			}

		})
	}
}

// TestUpsertErrors checks record failures separately from request failures.
func TestUpsertErrors(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		itemError  bool
		shouldCall bool
	}{
		{"negative ID", "-1", "", true, false},
		{"overflowing ID", "2147483648", "", true, false},
		{"malformed ID", "private@example.com", "", true, false},
		{"zero ID", "0", "", true, false},
		{"duplicate", "", `{"status":"error","error":{"type":"AlreadyExists","description":"private@example.com"}}`, true, true},
		{"validation", "1", `{"status":"error","error":{"type":"InvalidValue","field":"private@example.com","description":"secret"}}`, true, true},
		{"missing resource", "1", `{"status":"error","error":{"type":"NotFound"}}`, true, true},
		{"permissions", "1", `{"status":"error","error":{"type":"AccessDenied"}}`, false, true},
		{"unknown API error", "1", `{"status":"error","error":{"type":"private@example.com"}}`, false, true},
		{"missing created ID", "", `{"status":"ok"}`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, client := newTestConnector(t, func(req *http.Request) (*http.Response, error) {
				return testResponse(200, tt.body), nil
			})
			iterator := &recordsIterator{items: []connectors.Record{{ID: tt.id, Attributes: map[string]any{}}}}
			schema, err := c.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
			if err != nil {
				t.Fatalf("expected customer schema, got %v", err)
			}
			err = c.Upsert(t.Context(), connectors.TargetUser, iterator, schema)
			if err != nil {
				itemErrors, ok := errors.AsType[connectors.RecordsError](err)
				if ok != tt.itemError || (ok && (len(itemErrors) != 1 || itemErrors[0] == nil)) {
					t.Fatalf("expected item error %t at index zero, got %v", tt.itemError, err)
				}
				if strings.Contains(err.Error(), "private@example.com") || strings.Contains(err.Error(), "secret") {
					t.Fatalf("expected a private error, got %v", err)
				}
				if (client.calls == 1) != tt.shouldCall {
					t.Fatalf("expected HTTP call %t, got %d calls", tt.shouldCall, client.calls)
				}
				return
			}
			t.Fatal("expected upsert failure, got nil")
		})
	}
}

// newTestConnector builds a fake-backed connector and checks body closure.
func newTestConnector(t *testing.T, do func(*http.Request) (*http.Response, error)) (*CommerceReady, *testSettingsStore, *testHTTPClient) {
	t.Helper()
	value, err := json.Marshal(innerSettings{
		APIURL: testAPIURL, APIKey: testAPIKey, TimeZone: "Europe/Rome",
	})
	if err != nil {
		t.Fatalf("expected test settings, got %v", err)
	}
	store := &testSettingsStore{value: value}
	client := &testHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Host != "shop.example.com" || req.URL.RawQuery != "" {
			t.Fatalf("expected POST to the configured API without query parameters, got %s %s", req.Method, req.URL)
		}
		if req.Header.Get("X-Secret-Key") != testAPIKey || req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected secret-key authentication and JSON, got headers %#v", req.Header)
		}
		if do == nil {
			t.Fatal("expected no HTTP request, got a request")
		}
		return do(req)
	}}
	t.Cleanup(func() {
		for _, body := range client.bodies {
			if !body.closed {
				t.Error("expected closed response body, got an open body")
			}
		}
	})
	c, err := New(&connectors.ApplicationEnv{Settings: store, HTTPClient: client})
	if err != nil {
		t.Fatalf("expected connector construction, got %v", err)
	}
	return c, store, client
}

func testBillingJSON() json.Value {
	return json.Value(`{"firstName":"Ada","lastName":"Lovelace","personalCode":"","companyName":"",` +
		`"companyCode":"","companyCode2":"","street1":"1 Example St","street2":"","city":"Rome",` +
		`"postalCode":"00100","stateProv":"RM","country":"IT","phoneNumber":"",` +
		`"mobileNumber":"","faxNumber":"","email":"ada@example.com"}`)
}

func testResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
