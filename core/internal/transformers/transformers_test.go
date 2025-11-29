// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"slices"
	"testing"

	"github.com/meergo/meergo/tools/types"
)

func TestSchemaSubset(t *testing.T) {
	base := types.Object([]types.Property{
		{Name: "id", Type: types.Int(64)},
		{Name: "profile", Type: types.Object([]types.Property{
			{Name: "name", Type: types.Text()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "city", Type: types.Text()},
				{Name: "zip", Type: types.Text()},
			})},
		})},
		{Name: "settings", Type: types.Object([]types.Property{
			{Name: "flags", Type: types.Object([]types.Property{
				{Name: "newsletter", Type: types.Boolean()},
				{Name: "alerts", Type: types.Boolean()},
			})},
			{Name: "timezone", Type: types.Text()},
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
						{Name: "zip", Type: types.Text()},
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
	_ = schemaSubset(types.Text(), []string{"any"})
}
