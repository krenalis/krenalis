// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestIdentityWriterSemantics checks validation against the profile schema before any mutation or queued operation.
func TestIdentityWriterSemantics(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic types.Type
		value    string
		valid    bool
	}{
		{"current country", country, "IT", true},
		{"former country", country, "AN", true},
		{"unknown country", country, "ZZ", false},
		{"reserved country", country, "UK", false},
		{"lowercase country", country, "it", false},
		{"empty country", country, "", false},
		{"short country", country, "I", false},
		{"long country", country, "ITA", false},
		{"non-ASCII country", country, "é", false},
		{"phone at limit", types.String().AsPhone(), "+123456789012345", true},
		{"long phone", types.String().AsPhone(), "+1234567890123456", false},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", true},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", false},
		{"empty phone", types.String().AsPhone(), "", true},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", true},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			schema := types.Object([]types.Property{
				{Name: "value", Type: test.semantic},
			})
			p, _ := schema.Properties().ByName("value")
			// The pipeline uses the profile property type.
			pipelineSchema := types.Object([]types.Property{p})
			err := schemas.CheckAlignment(pipelineSchema, schema, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"batch", "event"} {

				t.Run(mode, func(t *testing.T) {

					queue := make(chan flusherRow[map[string]any], 2)
					identity := Identity{
						ID:          "id",
						AnonymousID: "anonymous",
						Attributes:  map[string]any{"value": test.value},
					}
					flatter := newFlatter(pipelineSchema, profileColumnByProperty(schema))
					var err error
					switch mode {
					case "batch":
						writer := &BatchIdentityWriter{schema: schema, flatter: flatter, identities: queue}
						err = writer.Write(t.Context(), identity)
					case "event":
						writer := &EventIdentityWriter{
							schema:     schema,
							flatter:    flatter,
							aligned:    true,
							pipelines:  map[string]struct{}{"pipeline": {}},
							identities: queue,
						}
						err = writer.Write(t.Context(), identity, nil)
					}
					if err != nil {
						if test.valid {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[*schemas.SemanticValidationError](err); !ok {
							t.Fatalf("expected SemanticValidationError, got %T", err)
						}
						if len(queue) != 0 {
							t.Fatal("invalid identity queued a write or an anonymous-identity purge")
						}
						if !reflect.DeepEqual(identity.Attributes, map[string]any{"value": test.value}) {
							t.Fatal("invalid identity was mutated")
						}
						return
					}
					if !test.valid {
						t.Fatal("invalid identity was accepted")
					}
					if len(queue) == 0 {
						t.Fatal("valid identity was not queued")
					}

				})

			}

		})

	}

}
