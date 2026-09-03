// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"io"
	"iter"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

type recordFetcherFunc func(context.Context, connectors.Targets, time.Time, string, types.Type) ([]connectors.Record, string, error)

func (recordFetcherFunc) RecordSchema(context.Context, connectors.Targets, connectors.Role) (types.Type, error) {
	panic("unexpected call to RecordSchema")
}

func (f recordFetcherFunc) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	return f(ctx, target, updatedAt, cursor, schema)
}

// TestAppRecordsPaging verifies paging, deduplication, and lazy record processing.
func TestAppRecordsPaging(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "email", Type: types.String()},
	})
	updatedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	newRecord := func(id string) connectors.Record {
		return connectors.Record{
			ID:         id,
			Attributes: map[string]any{"email": id + "@example.com"},
			UpdatedAt:  updatedAt,
		}
	}

	t.Run("deduplicates records", func(t *testing.T) {

		duplicateUser1 := newRecord("user-1")
		duplicateUser1.Attributes["email"] = "duplicate-user-1@example.com"
		duplicateUser2 := newRecord("user-2")
		duplicateUser2.Attributes["email"] = "duplicate-user-2@example.com"
		records := &appRecords{
			schema:    schema,
			appSchema: schema,
			connector: "test",
			inner: recordFetcherFunc(func(_ context.Context, _ connectors.Targets, _ time.Time, cursor string, _ types.Type) ([]connectors.Record, string, error) {
				switch cursor {
				case "":
					return []connectors.Record{newRecord("user-1"), duplicateUser1, newRecord("user-2")}, "second", nil
				case "second":
					return []connectors.Record{duplicateUser2, newRecord("user-3")}, "third", nil
				case "third":
					return []connectors.Record{newRecord("user-1"), newRecord("user-3")}, "", io.EOF
				default:
					t.Fatalf("unexpected cursor %q", cursor)
					return nil, "", nil
				}
			}),
		}
		defer records.Close()

		var got []string
		var user1Email, user2Email any
		for record := range records.All(t.Context()) {
			got = append(got, record.ID)
			switch record.ID {
			case "user-1":
				user1Email = record.Attributes["email"]
			case "user-2":
				user2Email = record.Attributes["email"]
			}
		}
		if err := records.Err(); err != nil {
			t.Fatalf("expected no iterator error, got %s", err)
		}
		if expected := []string{"user-1", "user-2", "user-3"}; !slices.Equal(got, expected) {
			t.Fatalf("expected record IDs %q, got %q", expected, got)
		}
		if user1Email != "user-1@example.com" {
			t.Fatalf("expected the first user-1 record to be retained, got email %v", user1Email)
		}
		if user2Email != "user-2@example.com" {
			t.Fatalf("expected the first user-2 record to be retained, got email %v", user2Email)
		}

	})

	t.Run("yields records returned with EOF", func(t *testing.T) {

		records := &appRecords{
			schema:    schema,
			appSchema: schema,
			connector: "test",
			inner: recordFetcherFunc(func(_ context.Context, _ connectors.Targets, _ time.Time, cursor string, _ types.Type) ([]connectors.Record, string, error) {
				if cursor != "" {
					t.Fatalf("unexpected cursor %q", cursor)
				}
				return []connectors.Record{newRecord("user-1")}, "", io.EOF
			}),
		}
		defer records.Close()

		var got []string
		for record := range records.All(t.Context()) {
			got = append(got, record.ID)
		}
		if err := records.Err(); err != nil {
			t.Fatalf("expected no iterator error, got %s", err)
		}
		if expected := []string{"user-1"}; !slices.Equal(got, expected) {
			t.Fatalf("expected record IDs %q, got %q", expected, got)
		}

	})

	t.Run("does not process records ahead of iteration", func(t *testing.T) {

		// invalidRecord makes appRecords fail if it processes past user-2.
		invalidRecord := newRecord("")
		records := &appRecords{
			schema:    schema,
			appSchema: schema,
			connector: "test",
			inner: recordFetcherFunc(func(_ context.Context, _ connectors.Targets, _ time.Time, cursor string, _ types.Type) ([]connectors.Record, string, error) {
				switch cursor {
				case "":
					return []connectors.Record{newRecord("user-1")}, "second", nil
				case "second":
					return []connectors.Record{newRecord("user-2"), invalidRecord}, "", io.EOF
				default:
					t.Fatalf("unexpected cursor %q", cursor)
					return nil, "", nil
				}
			}),
		}
		defer records.Close()

		var got []string
		for record := range records.All(t.Context()) {
			got = append(got, record.ID)
			if len(got) == 2 {
				break
			}
		}
		if err := records.Err(); err != nil {
			t.Fatalf("expected no iterator error, got %s", err)
		}
		if expected := []string{"user-1", "user-2"}; !slices.Equal(got, expected) {
			t.Fatalf("expected record IDs %q, got %q", expected, got)
		}

	})

}

// TestAppRecordsPreservesConnectorRecordError verifies that an application
// record preserves the connector error and does not read its update time or
// attributes when the record has an error.
func TestAppRecordsPreservesConnectorRecordError(t *testing.T) {

	recordErr := errors.New("record cannot be read")
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.String()},
	})
	records := &appRecords{
		schema:    schema,
		appSchema: schema,
		updatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		connector: "test",
		inner: recordFetcherFunc(func(context.Context, connectors.Targets, time.Time, string, types.Type) ([]connectors.Record, string, error) {
			return []connectors.Record{{
				ID:  "user-1",
				Err: recordErr,
			}}, "", io.EOF
		}),
	}
	defer records.Close()

	var got []Record
	for record := range records.All(t.Context()) {
		got = append(got, record)
	}
	if err := records.Err(); err != nil {
		t.Fatalf("expected no iterator error, got %s", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one record, got %d", len(got))
	}
	record := got[0]
	if record.ID != "user-1" {
		t.Fatalf("expected record ID %q, got %q", "user-1", record.ID)
	}
	if !errors.Is(record.Err, recordErr) {
		t.Fatalf("expected record error %q, got %v", recordErr, record.Err)
	}
	if !record.UpdatedAt.IsZero() {
		t.Fatalf("expected zero update time, got %s", record.UpdatedAt)
	}
	if record.Attributes != nil {
		t.Fatalf("expected nil attributes, got %#v", record.Attributes)
	}
}

// TestValidateEventType verifies validation of event type IDs, ordering groups,
// and delivery endpoints.
func TestValidateEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType *EventType
		err       string
	}{
		{name: "default delivery endpoint", eventType: &EventType{ID: "createContact", OrderingGroup: "contacts"}},
		{name: "explicit delivery endpoint", eventType: &EventType{ID: "createContact", OrderingGroup: "contacts", DeliveryEndpoint: "contacts"}},
		{name: "invalid ID", eventType: &EventType{ID: "create-contact"}, err: `connector test returned an invalid event type ID ("create-contact")`},
		{name: "long ID", eventType: &EventType{ID: strings.Repeat("a", 26)}, err: `connector test returned an invalid event type ID ("aaaaaaaaaaaaaaaaaaaaaaaaaa")`},
		{name: "invalid ordering group", eventType: &EventType{ID: "contact", OrderingGroup: "contact-events"}, err: `connector test returned an invalid ordering group ("contact-events")`},
		{name: "long ordering group", eventType: &EventType{ID: "contact", OrderingGroup: strings.Repeat("a", 26)}, err: `connector test returned an invalid ordering group ("aaaaaaaaaaaaaaaaaaaaaaaaaa")`},
		{name: "invalid delivery endpoint", eventType: &EventType{ID: "contact", DeliveryEndpoint: "contact-events"}, err: `connector test returned an invalid delivery endpoint ("contact-events")`},
		{name: "long delivery endpoint", eventType: &EventType{ID: "contact", DeliveryEndpoint: strings.Repeat("a", 26)}, err: `connector test returned an invalid delivery endpoint ("aaaaaaaaaaaaaaaaaaaaaaaaaa")`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEventType("test", test.eventType)
			if test.err != "" {
				if err == nil {
					t.Fatalf("expected %q, got nil", test.err)
				}
				if err.Error() != test.err {
					t.Fatalf("expected %q, got %q", test.err, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

// TestApplicationEventType verifies that EventType validates event types before
// returning the one with the requested ID.
func TestApplicationEventType(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		expected := &EventType{ID: "createContact", OrderingGroup: "contacts"}
		app := &Application{inner: &testEventSender{eventTypes: []*EventType{
			expected,
			{ID: "updateContact", OrderingGroup: "contacts"},
		}}}

		got, err := app.EventType(context.Background(), expected.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != expected {
			t.Fatalf("expected event type %p, got %p", expected, got)
		}
	})

	t.Run("invalid event type", func(t *testing.T) {
		app := &Application{connector: "test", inner: &testEventSender{eventTypes: []*EventType{
			{ID: "createContact"},
			{ID: "invalid-id"},
		}}}

		_, err := app.EventType(context.Background(), "createContact")
		expected := `connector test returned an invalid event type ID ("invalid-id")`
		if err == nil {
			t.Fatalf("expected %q, got nil", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("missing", func(t *testing.T) {
		app := &Application{inner: &testEventSender{}}

		_, err := app.EventType(context.Background(), "createContact")
		if err != connectors.ErrEventTypeNotExist {
			t.Fatalf("expected %v, got %v", connectors.ErrEventTypeNotExist, err)
		}
	})

	t.Run("repeated", func(t *testing.T) {
		app := &Application{connector: "test", inner: &testEventSender{eventTypes: []*EventType{
			{ID: "createContact"},
			{ID: "createContact"},
		}}}

		_, err := app.EventType(context.Background(), "createContact")
		expected := "connector test returned multiple event types with the same ID (createContact)"
		if err == nil {
			t.Fatalf("expected %q, got nil", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("different delivery endpoints", func(t *testing.T) {
		app := &Application{connector: "test", inner: &testEventSender{eventTypes: []*EventType{
			{ID: "createContact", OrderingGroup: "contacts"},
			{ID: "updateContact", OrderingGroup: "contacts", DeliveryEndpoint: "contacts"},
		}}}

		_, err := app.EventType(context.Background(), "createContact")
		expected := `connector test returned different delivery endpoints for ordering group "contacts"`
		if err == nil {
			t.Fatalf("expected %q, got nil", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err.Error())
		}
	})
}

// TestApplicationEventTypesRejectsDifferentDeliveryEndpoints verifies that
// event types in the same ordering group must resolve to the same delivery
// endpoint.
func TestApplicationEventTypesRejectsDifferentDeliveryEndpoints(t *testing.T) {
	app := &Application{connector: "test", inner: &testEventSender{eventTypes: []*EventType{
		{ID: "createContact", OrderingGroup: "contacts"},
		{ID: "updateContact", OrderingGroup: "contacts", DeliveryEndpoint: "contacts"},
	}}}

	_, err := app.EventTypes(context.Background())
	expected := `connector test returned different delivery endpoints for ordering group "contacts"`
	if err == nil {
		t.Fatalf("expected %q, got nil", expected)
	}
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

// testEventSender provides event types to Application tests.
type testEventSender struct {
	connectors.EventSender
	eventTypes []*EventType
}

// EventTypes returns the configured event types.
func (sender *testEventSender) EventTypes(context.Context) ([]*EventType, error) {
	return sender.eventTypes, nil
}

func Test_sameValue(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "foo", Type: types.String()},
		{Name: "boo", Type: types.Array(types.Boolean())},
	})

	tests := []struct {
		t        types.Type
		v, v2    any
		expected bool
	}{
		{t: types.String(), v: nil, v2: nil, expected: true},
		{t: types.String(), v: nil, v2: 5, expected: false},
		{t: types.Int(32), v: 4, v2: 4, expected: true},
		{t: types.Int(32), v: 4, v2: nil, expected: false},
		{t: types.Float(64), v: 12.9037, v2: 12.9037, expected: true},
		{t: types.JSON(), v: nil, v2: nil, expected: true},
		{t: types.JSON(), v: json.Value(`null`), v2: json.Value(`null`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"b":[1,2]}`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"c":true}`), expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"a", "b"}, expected: true},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"b", "a"}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: map[string]any{}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, false, true}}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, true, true}}, expected: false},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: nil, expected: false},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"a": 5, "b": 0}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 5}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 3}, expected: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := sameValue(test.t, test.v, test.v2)
			if test.expected != got {
				t.Fatalf("expected %t, got %t", test.expected, got)
			}
		})
	}

}

// Test_singleEventIterator_Peek verifies Peek's behavior before, during, and
// after iterating over a single event.
func Test_singleEventIterator_Peek(t *testing.T) {

	tests := []struct {
		name string
		seq  func(*singleEventIterator) iter.Seq[*connectors.Event]
	}{
		{name: "All", seq: (*singleEventIterator).All},
		{name: "SameUser", seq: (*singleEventIterator).SameUser},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := new(connectors.Event)
			events := newSingleEventIterator(event, "test")

			for range 2 {
				got, ok := events.Peek()
				if !ok || got != event {
					t.Fatalf("Peek before iteration: expected event %p and true, got %p and %t", event, got, ok)
				}
			}

			yielded := 0
			for range test.seq(events) {
				yielded++
				if got, ok := events.Peek(); ok || got != nil {
					t.Fatalf("Peek during iteration: expected nil and false, got %p and %t", got, ok)
				}
			}
			if yielded != 1 {
				t.Fatalf("expected one event, got %d", yielded)
			}

			defer func() {
				if recover() == nil {
					t.Fatal("Peek after iteration: expected panic")
				}
			}()
			events.Peek()
		})
	}
}

// Test_singleEventIterator_UsageAfterIteration verifies that methods available
// only during an active iteration panic after the iteration completes.
func Test_singleEventIterator_UsageAfterIteration(t *testing.T) {

	sequences := []struct {
		name string
		seq  func(*singleEventIterator) iter.Seq[*connectors.Event]
	}{
		{name: "All", seq: (*singleEventIterator).All},
		{name: "SameUser", seq: (*singleEventIterator).SameUser},
	}
	methods := []struct {
		name string
		call func(*singleEventIterator)
	}{
		{name: "Discard", call: func(events *singleEventIterator) {
			events.Discard(errors.New("event is invalid"))
		}},
		{name: "Postpone", call: func(events *singleEventIterator) {
			events.Postpone()
		}},
	}

	for _, sequence := range sequences {
		for _, method := range methods {
			t.Run(sequence.name+"/"+method.name, func(t *testing.T) {
				events := newSingleEventIterator(&connectors.Event{}, "test")
				for range sequence.seq(events) {
				}
				defer func() {
					if recover() == nil {
						t.Fatal(method.name + " after iteration: expected panic")
					}
				}()
				method.call(events)
			})
		}
	}

}
