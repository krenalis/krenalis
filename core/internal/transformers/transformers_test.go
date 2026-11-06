// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"reflect"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

// TestMappingMatching checks that the transformer passes the separately
// supplied matching property to the mapping, as in a Mailchimp export.
func TestMappingMatching(t *testing.T) {

	pipeline := &state.Pipeline{
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String()},
			{Name: "status", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email_address", Type: types.String(), CreateRequired: true, UpdateRequired: true},
			{Name: "status", Type: types.String(), CreateRequired: true},
		}),
		Matching:       state.Matching{In: "email", Out: "email_address"},
		Transformation: state.Transformation{Mapping: map[string]string{"status": "status"}},
	}
	transformer, err := New("organization", pipeline, nil)
	if err != nil {
		t.Fatalf("expected transformer construction to succeed, got %v", err)
	}

	records := []Record{
		{Purpose: Create, Attributes: map[string]any{"email": "user@example.com", "status": "subscribed"}},
		{Purpose: Update, Attributes: map[string]any{"email": "user@example.com", "status": "subscribed"}},
	}
	err = transformer.Transform(t.Context(), records)
	if err != nil {
		t.Fatalf("expected transformation to succeed, got %v", err)
	}
	for _, record := range records {
		if record.Err != nil {
			t.Fatalf("expected record transformation to succeed, got %v", record.Err)
		}
		expected := map[string]any{"status": "subscribed"}
		if !reflect.DeepEqual(record.Attributes, expected) {
			t.Fatalf("expected output %#v, got %#v", expected, record.Attributes)
		}
	}

}

func TestSchemaSubset(t *testing.T) {
	base := types.Object([]types.Property{
		{Name: "id", Type: types.Int(64)},
		{Name: "profile", Type: types.Object([]types.Property{
			{Name: "name", Type: types.String()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "city", Type: types.String()},
				{Name: "zip", Type: types.String()},
			})},
		})},
		{Name: "settings", Type: types.Object([]types.Property{
			{Name: "flags", Type: types.Object([]types.Property{
				{Name: "newsletter", Type: types.Boolean()},
				{Name: "alerts", Type: types.Boolean()},
			})},
			{Name: "timezone", Type: types.String()},
		})},
	})

	tests := []struct {
		name      string
		paths     []string
		want      types.Type
		wantValid bool
	}{
		{
			name:  "keeps full branch when ancestor path is requested",
			paths: []string{"settings.flags"},
			want: types.Object([]types.Property{
				{Name: "settings", Type: types.Object([]types.Property{
					{Name: "flags", Type: types.Object([]types.Property{
						{Name: "newsletter", Type: types.Boolean()},
						{Name: "alerts", Type: types.Boolean()},
					})},
				})},
			}),
			wantValid: true,
		},
		{
			name:  "keeps requested leaves and prunes siblings",
			paths: []string{"id", "settings.flags.alerts"},
			want: types.Object([]types.Property{
				{Name: "id", Type: types.Int(64)},
				{Name: "settings", Type: types.Object([]types.Property{
					{Name: "flags", Type: types.Object([]types.Property{
						{Name: "alerts", Type: types.Boolean()},
					})},
				})},
			}),
			wantValid: true,
		},
		{
			name:  "includes parent hierarchy for deep leaf",
			paths: []string{"profile.address.zip"},
			want: types.Object([]types.Property{
				{Name: "profile", Type: types.Object([]types.Property{
					{Name: "address", Type: types.Object([]types.Property{
						{Name: "zip", Type: types.String()},
					})},
				})},
			}),
			wantValid: true,
		},
		{
			name:  "ignores unknown paths while keeping known ones",
			paths: []string{"id", "missing.branch"},
			want: types.Object([]types.Property{
				{Name: "id", Type: types.Int(64)},
			}),
			wantValid: true,
		},
		{
			name:      "no matching paths returns invalid schema",
			paths:     []string{"does.not.exist"},
			wantValid: false,
		},
		{
			name:      "nil paths return invalid schema",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := slices.Clone(tt.paths)
			slices.Sort(paths)

			got := schemaSubset(base, paths)
			if !tt.wantValid {
				if got.Valid() {
					t.Fatalf("expected invalid type, got %v", got)
				}
				return
			}
			if !got.Valid() {
				t.Fatalf("expected valid type, got invalid")
			}
			if !types.Equal(tt.want, got) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestSchemaSubsetPanicsForNonObjectSchema(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for non-object schema")
		}
	}()
	_ = schemaSubset(types.String(), []string{"any"})
}
