// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

// TestWorkspaceEventDefinitions checks that callers cannot alter the catalog.
func TestWorkspaceEventDefinitions(t *testing.T) {

	var workspace state.Workspace
	definitions := workspace.EventDefinitions()
	if len(definitions) == 0 {
		t.Fatal("expected example event definitions, got an empty catalog")
	}
	want := slices.Clone(definitions)

	definitions[0].ID = "changed"
	definitions[0].Event = "Changed Event"
	definitions[0].Schemas.Traits = types.Type{}
	definitions[0].Schemas.Properties = types.Type{}
	slices.Reverse(definitions)

	for _, workspace := range []state.Workspace{workspace, {}} {
		got := workspace.EventDefinitions()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected an unchanged catalog, got %#v", got)
		}
	}

}

// TestWorkspaceLookupEventDefinition checks exact event and name matching.
func TestWorkspaceLookupEventDefinition(t *testing.T) {

	var workspace state.Workspace
	definitions := workspace.EventDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("expected two example definitions, got %d", len(definitions))
	}
	track, page := definitions[0], definitions[1]

	for _, test := range []struct {
		name        string
		typ         string
		eventOrName string
		want        state.EventDefinition
	}{
		{"track event", "track", "Example Action", track},
		{"page name", "page", "Example Page", page},
		{"wrong type for event", "page", "Example Action", state.EventDefinition{}},
		{"wrong type for name", "track", "Example Page", state.EventDefinition{}},
		{"unknown type", "unknown", "Example Action", state.EventDefinition{}},
		{"missing type", "", "Example Action", state.EventDefinition{}},
		{"missing event", "track", "", state.EventDefinition{}},
		{"missing name", "page", "", state.EventDefinition{}},
		{"unknown event", "track", "Unknown", state.EventDefinition{}},
		{"unknown name", "page", "Unknown", state.EventDefinition{}},
		{"event ID", "track", "example-action", state.EventDefinition{}},
		{"page ID", "page", "example-page", state.EventDefinition{}},
		{"type case", "Track", "Example Action", state.EventDefinition{}},
		{"event case", "track", "example action", state.EventDefinition{}},
		{"name case", "page", "example page", state.EventDefinition{}},
		{"event whitespace", "track", "Example Action ", state.EventDefinition{}},
		{"name whitespace", "page", " Example Page", state.EventDefinition{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := workspace.LookupEventDefinition(test.typ, test.eventOrName)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %#v, got %#v", test.want, got)
			}
		})
	}

	definition := workspace.LookupEventDefinition("track", "Example Action")
	definition.Event = "Changed Event"
	definition.Schemas.Traits = types.Type{}
	definition.Schemas.Properties = types.Type{}
	got := workspace.EventDefinitions()
	if !reflect.DeepEqual(got, definitions) {
		t.Fatalf("expected an unchanged catalog, got %#v", got)
	}

}
