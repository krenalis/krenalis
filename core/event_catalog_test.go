// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestEventCatalogFacade checks bounded discovery and exact lookup without
// requiring a warehouse connection or exposing mutable state.
func TestEventCatalogFacade(t *testing.T) {

	workspace := &Workspace{core: &Core{}, workspace: &state.Workspace{}}
	definitions, err := workspace.EventDefinitions("")
	if err != nil {
		t.Fatalf("expected the catalog, got %v", err)
	}
	if len(definitions) != 102 {
		t.Fatalf("expected 102 entries, got %d", len(definitions))
	}
	definitions[0].ID = "changed"
	for _, query := range []string{"ORDER COMPLETED", " commerce.order_completed ", "Ordine completato"} {
		matches, err := workspace.EventDefinitions(query)
		if err != nil {
			t.Fatalf("expected matching definitions, got %v", err)
		}
		found := false
		for _, match := range matches {
			if match.ID == "commerce.order_completed" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected Order Completed, got %+v", matches)
		}
	}
	definition, err := workspace.EventDefinition("commerce.order_completed")
	if err != nil {
		t.Fatalf("expected definition, got %v", err)
	}
	if definition.Event != "Order Completed" || !definition.Schemas.Properties.IsObject() {
		t.Fatalf("expected technical event and native schema, got %+v", definition)
	}
	definition.Event = "changed"
	again, err := workspace.EventDefinition("commerce.order_completed")
	if err != nil {
		t.Fatalf("expected definition, got %v", err)
	}
	if again.Event != "Order Completed" {
		t.Fatalf("expected unchanged definition, got %+v, %v", again, err)
	}
	for _, id := range []string{"", strings.Repeat("a", 129), "\xff"} {
		_, err := workspace.EventDefinition(id)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected bad request, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid ID error, got nil")
	}
	_, err = workspace.EventDefinition("Order Completed")
	if err != nil {
		failure, ok := errors.AsType[*errors.UnprocessableError](err)
		if !ok || failure.Code != EventDefinitionNotExist {
			t.Fatalf("expected EventDefinitionNotExist, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected unknown ID error, got nil")
	}
	for _, query := range []string{strings.Repeat("a", 257), "\xff"} {
		_, err := workspace.EventDefinitions(query)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected bad request, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid query error, got nil")
	}
	workspace.core.closed.Store(true)
	defer func() {
		if recover() == nil {
			t.Fatal("expected closed-core panic, got no panic")
		}
	}()
	workspace.EventDefinitions("")

}

// TestEventCatalogSchemaTypes checks profile-compatible value types while
// allowing nullable properties and arrays of objects specifically for events.
func TestEventCatalogSchemaTypes(t *testing.T) {
	var workspace state.Workspace
	for _, definition := range workspace.EventDefinitions() {
		for _, schema := range []types.Type{definition.Schemas.Traits, definition.Schemas.Properties} {
			if !schema.Valid() {
				continue
			}
			for path, property := range schema.Properties().WalkAll() {
				if property.Type.IsObject() || property.Type.IsArray() && property.Type.Elem().IsObject() {
					continue
				}
				property.Nullable = false
				err := checkAllowedPropertyProfileSchema(types.Object([]types.Property{property}))
				if err != nil {
					t.Fatalf("expected profile-compatible type for %s.%s, got %v", definition.ID, path, err)
				}
			}
		}
	}
}
