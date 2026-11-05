// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state_test

import (
	"archive/zip"
	"io"
	"os/exec"
	"reflect"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestWorkspaceEventDefinitions checks that callers cannot alter the catalog.
func TestWorkspaceEventDefinitions(t *testing.T) {

	var workspace state.Workspace
	definitions := workspace.EventDefinitions()
	if len(definitions) != 102 {
		t.Fatalf("expected 102 event definitions, got %d", len(definitions))
	}
	want := slices.Clone(definitions)
	ids := map[string]bool{}
	typesCount := map[string]int{}
	for _, definition := range definitions {
		if definition.ID == "" || ids[definition.ID] {
			t.Fatalf("expected a unique nonempty event ID, got %q", definition.ID)
		}
		ids[definition.ID] = true
		typesCount[definition.Type]++
		if definition.DisplayName == "" || definition.Description == "" {
			t.Fatalf("expected event labels and descriptions, got %#v", definition)
		}
		for _, schema := range []types.Type{definition.Schemas.Traits, definition.Schemas.Properties} {
			if schema.Valid() && !schema.IsObject() {
				t.Fatalf("expected an object or unspecified schema for %s, got %s", definition.ID, schema)
			}
		}
		if definition.Schemas.Properties.Valid() &&
			definition.Type != "track" && definition.Type != "page" && definition.Type != "screen" {
			t.Fatalf("expected no properties schema for type %s, got %s", definition.Type, definition.ID)
		}
	}
	wantCounts := map[string]int{"track": 96, "page": 1, "screen": 1, "identify": 1, "group": 1, "alias": 1, "merge": 1}
	if !reflect.DeepEqual(typesCount, wantCounts) {
		t.Fatalf("expected event counts %v, got %v", wantCounts, typesCount)
	}

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

// TestWorkspaceEventDefinitionSchemas checks the native choices for business
// values, typed collections and source contracts with incomplete information.
func TestWorkspaceEventDefinitionSchemas(t *testing.T) {

	var workspace state.Workspace
	for _, test := range []struct {
		name     string
		typ      string
		event    string
		traits   bool
		path     string
		want     types.Type
		nullable bool
	}{
		{"order amount", "track", "Order Completed", false, "total", types.Decimal(18, 4).AsMoney(), false},
		{"product amount", "track", "Order Completed", false, "products.price", types.Decimal(18, 4).AsMoney(), false},
		{"fractional quantity", "track", "Order Completed", false, "products.quantity", types.Decimal(18, 4), false},
		{"event-specific monetary value", "track", "Product Viewed", false, "value", types.Decimal(18, 4).AsMoney(), false},
		{"generic textual value", "track", "Signed In", false, "value", types.String(), false},
		{"rating", "track", "Product Reviewed", false, "rating", types.String(), false},
		{"structured search", "track", "Products Searched", false, "query", types.JSON(), false},
		{"heterogeneous ad type", "track", "Video Playback Started", false, "ad_type", types.JSON(), false},
		{"nullable live duration", "track", "Video Playback Started", false, "total_length", types.Float(64).Real().AsDuration(types.Second), true},
		{"ad duration", "track", "Video Ad Playing", false, "total_length", types.Float(64).Real().AsDuration(types.Second), false},
		{"offset awaiting Measurement Second", "track", "Video Playback Started", false, "position", types.Float(64).Real(), false},
		{"chat duration", "track", "Live Chat Conversation Ended", false, "conversation_duration", types.Float(64).Real().AsDuration(types.Second), false},
		{"logical date", "identify", "", true, "birthday", types.Date(), false},
		{"account creation time", "identify", "", true, "createdAt", types.DateTime(), false},
		{"company identifier", "identify", "", true, "company.id", types.String(), false},
		{"group employees label", "group", "", true, "employees", types.String(), false},
		{"destination-only checkout value", "track", "Checkout Step Completed", false, "value", types.JSON(), false},
		{"destination-only payment value", "track", "Payment Info Entered", false, "value", types.JSON(), false},
		{"proposed type stays unconstrained", "track", "Order Created", false, "total", types.JSON(), false},
		{"unknown type stays unconstrained", "track", "Video Quality Updated", false, "startupTime", types.JSON(), false},
	} {
		t.Run(test.name, func(t *testing.T) {

			definition := workspace.LookupEventDefinition(test.typ, test.event)
			schema := definition.Schemas.Properties
			if test.traits {
				schema = definition.Schemas.Traits
			}
			if !schema.IsObject() {
				t.Fatalf("expected an object schema for %s, got %s", definition.ID, schema)
			}
			var property types.Property
			for path, p := range schema.Properties().WalkAll() {
				if path == test.path {
					property = p
					break
				}
			}

			if !types.Equal(property.Type, test.want) || property.Nullable != test.nullable || !property.ReadOptional {
				t.Fatalf("expected %s nullable=%t optional=true, got %s nullable=%t optional=%t",
					test.want, test.nullable, property.Type, property.Nullable, property.ReadOptional)
			}
			if property.Description == "" {
				t.Fatalf("expected a property description for %s, got an empty description", test.path)
			}

		})
	}

	order := workspace.LookupEventDefinition("track", "Order Completed")
	products, ok := order.Schemas.Properties.Properties().ByName("products")
	if !ok || !products.Type.IsArray() || !products.Type.Elem().IsObject() {
		t.Fatalf("expected an array of product objects, got %s", products.Type)
	}
	signIn := workspace.LookupEventDefinition("track", "Signed In")
	if signIn.Schemas.Traits.Properties().ContainsName("email") {
		t.Fatal("expected no event-specific email contract on Signed In, got an email property")
	}
	group := workspace.LookupEventDefinition("group", "")
	if group.Schemas.Traits.Properties().ContainsName("firstName") {
		t.Fatal("expected group traits rather than person traits, got firstName")
	}

}

// TestWorkspaceEventDefinitionSources compares every runtime definition with
// the authoritative taxonomy, including its technical matcher and description.
func TestWorkspaceEventDefinitionSources(t *testing.T) {

	archive, err := zip.OpenReader("../../../filtering/event-catalog/cdp-event-taxonomy.zip")
	if err != nil {
		t.Fatalf("expected the source taxonomy archive, got %v", err)
	}
	defer archive.Close()
	file, err := archive.Open("cdp-event-catalog.json")
	if err != nil {
		t.Fatalf("expected the source taxonomy JSON, got %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("expected source taxonomy data, got %v", err)
	}
	var source struct {
		Definitions []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			Match       struct {
				AnyOf []struct {
					All []struct {
						Field string `json:"field"`
						Op    string `json:"op"`
						Value string `json:"value"`
					} `json:"all"`
				} `json:"any_of"`
			} `json:"match"`
		} `json:"definitions"`
	}
	err = json.Unmarshal(data, &source)
	if err != nil {
		t.Fatalf("expected a decoded source taxonomy, got %v", err)
	}

	var workspace state.Workspace
	definitions := workspace.EventDefinitions()
	if len(definitions) != len(source.Definitions) {
		t.Fatalf("expected %d definitions from the source taxonomy, got %d", len(source.Definitions), len(definitions))
	}
	for _, definition := range source.Definitions {
		if len(definition.Match.AnyOf) != 1 {
			t.Fatalf("expected one source matcher branch for %s, got %d", definition.ID, len(definition.Match.AnyOf))
		}
		match := map[string]string{}
		for _, condition := range definition.Match.AnyOf[0].All {
			if condition.Op != "eq" {
				t.Fatalf("expected source equality for %s, got %s", definition.ID, condition.Op)
			}
			match[condition.Field] = condition.Value
		}
		eventOrName := match["event"]
		if match["type"] != "track" {
			eventOrName = match["name"]
		}
		got := workspace.LookupEventDefinition(match["type"], eventOrName)
		if got.ID != definition.ID || got.DisplayName != definition.DisplayName || got.Description != definition.Description {
			t.Fatalf("expected source metadata for %s, got different metadata for %s", definition.ID, got.ID)
		}
	}

}

// TestWorkspaceEventCatalogGeneration detects drift from the source generator.
func TestWorkspaceEventCatalogGeneration(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("requires Python 3 for catalog regeneration")
	}
	command := exec.CommandContext(t.Context(), python, "../../../filtering/event-catalog/generate_catalog.py", "--check")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("expected a reproducible compiled catalog, got %v: %s", err, output)
	}
}

// TestWorkspaceLookupEventDefinition checks exact track names and generic
// page and screen matching, without enabling compatibility aliases.
func TestWorkspaceLookupEventDefinition(t *testing.T) {

	var workspace state.Workspace
	definitions := workspace.EventDefinitions()
	byID := map[string]state.EventDefinition{}
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}

	for _, test := range []struct {
		name        string
		typ         string
		eventOrName string
		wantID      string
	}{
		{"track event", "track", "Order Completed", "commerce.order_completed"},
		{"generic page", "page", "Home", "web.page_viewed"},
		{"unnamed page", "page", "", "web.page_viewed"},
		{"generic screen", "screen", "Home", "app.screen_viewed"},
		{"unnamed screen", "screen", "", "app.screen_viewed"},
		{"identify", "identify", "", "identity.identified"},
		{"group", "group", "", "identity.group_associated"},
		{"alias", "alias", "", "identity.alias_recorded"},
		{"historical definition", "merge", "", "legacy.identity_merge"},
		{"track name on page", "page", "Order Completed", "web.page_viewed"},
		{"page name on track", "track", "Home", ""},
		{"unknown type", "unknown", "Order Completed", ""},
		{"missing type", "", "Order Completed", ""},
		{"missing event", "track", "", ""},
		{"unknown event", "track", "Unknown", ""},
		{"semantic ID is not a technical name", "track", "commerce.order_completed", ""},
		{"type case", "Track", "Order Completed", ""},
		{"event case", "track", "order completed", ""},
		{"event whitespace", "track", "Order Completed ", ""},
		{"disabled compatibility alias", "track", "Completed Order", ""},
		{"similar name", "track", "Purchase Completed", ""},
		{"group with a name", "group", "Company", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := workspace.LookupEventDefinition(test.typ, test.eventOrName)
			if got.ID != test.wantID || !reflect.DeepEqual(got, byID[test.wantID]) {
				t.Fatalf("expected definition %q, got %#v", test.wantID, got)
			}
		})
	}

	definition := workspace.LookupEventDefinition("track", "Order Completed")
	definition.Event = "Changed Event"
	definition.Schemas.Traits = types.Type{}
	definition.Schemas.Properties = types.Type{}
	got := workspace.EventDefinitions()
	if !reflect.DeepEqual(got, definitions) {
		t.Fatalf("expected an unchanged catalog, got %#v", got)
	}

}
