// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateEventType verifies event type ID and ordering group validation.
func TestValidateEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType *EventType
		err       string
	}{
		{name: "valid", eventType: &EventType{ID: "createContact", OrderingGroup: "contacts"}},
		{name: "invalid ID", eventType: &EventType{ID: "create-contact"}, err: `connector test returned an invalid event type ID ("create-contact")`},
		{name: "long ID", eventType: &EventType{ID: strings.Repeat("a", 26)}, err: `connector test returned an invalid event type ID ("aaaaaaaaaaaaaaaaaaaaaaaaaa")`},
		{name: "invalid ordering group", eventType: &EventType{ID: "contact", OrderingGroup: "contact-events"}, err: `connector test returned an invalid ordering group ("contact-events")`},
		{name: "long ordering group", eventType: &EventType{ID: "contact", OrderingGroup: strings.Repeat("a", 26)}, err: `connector test returned an invalid ordering group ("aaaaaaaaaaaaaaaaaaaaaaaaaa")`},
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

// TestApplicationEventType verifies that EventType validates only matching
// event types while detecting missing and repeated IDs.
func TestApplicationEventType(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		expected := &EventType{ID: "createContact", OrderingGroup: "contacts"}
		app := &Application{inner: &testEventSender{eventTypes: []*EventType{
			nil,
			{ID: "invalid-id"},
			expected,
		}}}

		got, err := app.EventType(context.Background(), expected.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != expected {
			t.Fatalf("expected event type %p, got %p", expected, got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		app := &Application{connector: "test", inner: &testEventSender{eventTypes: []*EventType{
			{ID: "invalid-id"},
		}}}

		_, err := app.EventType(context.Background(), "invalid-id")
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
		app := &Application{inner: &testEventSender{eventTypes: []*EventType{
			{ID: "createContact"},
			{ID: "createContact"},
		}}}

		_, err := app.EventType(context.Background(), "createContact")
		expected := `event type ID "createContact" is repeated`
		if err == nil {
			t.Fatalf("expected %q, got nil", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected %q, got %q", expected, err.Error())
		}
	})
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
