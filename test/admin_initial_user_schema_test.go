// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestAdminInitialProfileSchema tests the correctness of the profile schema that is
// initially created when a workspace is created through the Admin.
func TestAdminInitialProfileSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	f, err := os.Open(filepath.Join("..", "admin/src/components/routes/WorkspaceCreate/InitialSchema.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var schema types.Type
	err = dec.Decode(&schema)
	if err != nil {
		t.Fatal(err)
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.SetProfileSchema(schema)
	k.Start()
	defer k.Stop()

	got := k.Workspace().ProfileSchema
	if !types.Equal(schema, got) {
		t.Fatalf("expected profile schema %#v, got %#v", schema.Properties().Slice(), got.Properties().Slice())
	}

	for path, expected := range map[string]types.Type{
		"email":           types.String().WithMaxLength(254).AsEmail(),
		"phone_number":    types.String().AsPhone(),
		"address.country": types.String().AsCountry(types.ISO3166Alpha2),
	} {
		property, err := got.Properties().ByPath(path)
		if err != nil {
			t.Fatalf("expected property %q, got %s", path, err)
		}
		if !types.Equal(expected, property.Type) {
			t.Fatalf("expected type %#v for property %q, got %#v", expected, path, property.Type)
		}
	}

}
