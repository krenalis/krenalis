// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

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
