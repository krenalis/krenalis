// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"net/url"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestUSADocuments exercises real candidate documents through the application
// boundary, preserving canonical facts through writable observation changes.
func TestUSADocuments(t *testing.T) {

	data, err := fakedata.LoadUSAWorldData(t.Context(), "../../..")
	if err != nil {
		t.Fatalf("expected verified USA data, got %v", err)
	}
	namespace, err := fakedata.NewIdentityNamespace("usa-application", 1)
	if err != nil {
		t.Fatalf("expected USA namespace, got %v", err)
	}
	world, err := fakedata.NewUSAWorld(data, namespace, 2026, fakedata.SourceInstanceConfig{
		ID: "application", Version: fakedata.USASourceVersion, CoverageNumerator: 1, CoverageDenominator: 1,
		DuplicateNumerator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		t.Fatalf("expected prepared USA World, got %v", err)
	}
	photoURL, err := url.Parse("http://localhost/v1/simulated/usa/photos/123456789ABC/")
	if err != nil {
		t.Fatalf("expected photo URL, got %v", err)
	}
	documents, err := NewUSADocuments(world, photoURL)
	if err != nil {
		t.Fatalf("expected USA application documents, got %v", err)
	}
	activities := map[string]bool{}
	for index := fakedata.PersonIndex(1); index <= 256; index++ {

		record, err := documents.Record(index, 0)
		if err != nil {
			t.Fatalf("expected generated USA application record, got %v", err)
		}
		document, attributes, err := documents.decode(t.Context(), record.ID, record.Data)
		if err != nil {
			t.Fatalf("expected valid stored USA document, got %v", err)
		}
		if document.Source == nil || !bytes.Contains(document.Source, []byte(`"sw-country-aware-usd-m2-candidate-v1"`)) ||
			attributes["country"] != "US" || attributes["currency"] != "USD" || attributes["annual_income_usd"] == nil ||
			attributes["annual_income_eur"] != nil {
			t.Fatalf("expected private M2 candidate and USD-only Source projection, got %v", attributes)
		}
		activities[attributes["primary_activity"].(string)] = true
		patch := map[string]any{"first_name": "Updated", "current_age": 40,
			"address": map[string]any{"municipality": "Updated locality"}}
		updated, err := documents.update(t.Context(), record.ID, record.Data, patch)
		if err != nil {
			t.Fatalf("expected writable USA observation update, got %v", err)
		}
		afterDocument, after, err := documents.decode(t.Context(), record.ID, updated)
		if err != nil {
			t.Fatalf("expected readable updated USA document, got %v", err)
		}
		attributes["first_name"], attributes["current_age"] = "Updated", 40
		attributes["address"].(map[string]any)["municipality"] = "Updated locality"
		if !bytes.Equal(document.Source, afterDocument.Source) || !reflect.DeepEqual(after, attributes) {
			t.Fatalf("expected unchanged candidate facts with updated observations, got %v", after)
		}
		if index != 1 {
			continue
		}
		t.Run("approved public optionality", func(t *testing.T) {

			observation := completeDestinationAttributes()
			observation["address"].(map[string]any)["country"] = "US"
			observation["customer_id"] = "candidate-source"
			delete(observation, "first_name")
			delete(observation, "last_name")
			delete(observation, "phone")
			_, err := types.MarshalValidate(observation, usaSourceSchema)
			if err != nil {
				t.Fatalf("expected M2 Source admission of contact omissions, got %v", err)
			}
			_, err = types.MarshalValidate(observation, usaDemoSourceSchema)
			if err != nil {
				return
			}

			t.Fatal("expected unchanged historical Source schema, got altered optionality")

		})
		t.Run("present zero Income", func(t *testing.T) {

			canonical, err := world.DecodeDocument(t.Context(), document.Source)
			if err != nil {
				t.Fatalf("expected original USD candidate, got %v", err)
			}
			*canonical.AnnualIncomeUSD = 0
			source, err := world.EncodeDocument(canonical)
			if err != nil {
				t.Fatalf("expected representable USD zero, got %v", err)
			}
			stored, err := documents.encode(source, document.Observation)
			if err != nil {
				t.Fatalf("expected persisted USD zero, got %v", err)
			}
			attributes, err := documents.Decode(t.Context(), record.ID, stored)
			if err != nil {
				t.Fatalf("expected readable USD zero, got %v", err)
			}
			income, present := attributes["annual_income_usd"]
			if !present || income != uint(0) {
				t.Fatalf("expected present USD zero, got %v present=%t", income, present)
			}

		})
		for _, field := range []string{"annual_income_usd", "annual_income_eur", "country", "currency", "photo_id",
			"education_level", "primary_activity", "job_family", "industry", "job_level"} {
			t.Run(field, func(t *testing.T) {
				_, err := documents.update(t.Context(), record.ID, record.Data, map[string]any{field: "changed"})
				if err != nil {
					return
				}
				t.Fatalf("expected immutable %s rejection, got admission", field)
			})
		}
		for _, invalid := range []json.Value{
			json.Value(`null`), json.Value(`{}`), append(bytes.Clone(record.Data), record.Data...),
			bytes.Repeat([]byte(" "), 32<<10+1), document.Source,
			bytes.Replace(record.Data, []byte(usaApplicationDocumentSchema), []byte("unknown"), 1),
			bytes.Replace(record.Data, []byte(`"annualIncomeUSD":`), []byte(`"annualIncomeEUR":`), 1),
		} {
			_, err := documents.Decode(t.Context(), record.ID, invalid)
			if err != nil {
				continue
			}
			t.Fatal("expected invalid application document rejection, got admission")
		}
		t.Run("cross-country Destination", func(t *testing.T) {
			_, err := documents.update(t.Context(), record.ID, record.Data,
				map[string]any{"address": map[string]any{"country": "IT"}})
			if err != nil {
				return
			}
			t.Fatal("expected cross-country Destination rejection, got admission")
		})

	}
	if len(activities) != 6 {
		t.Fatalf("expected all six activity branches, got %v", activities)
	}

	attributes := completeDestinationAttributes()
	attributes["address"].(map[string]any)["country"] = "US"
	created, err := documents.create("created-usa", attributes)
	if err != nil {
		t.Fatalf("expected country-aware USA Destination create, got %v", err)
	}
	document, projected, err := documents.decode(t.Context(), "created-usa", created)
	if err != nil {
		t.Fatalf("expected readable USA Destination create, got %v", err)
	}
	if document.Source != nil || projected["currency"] != "USD" || projected["country"] != "US" ||
		projected["annual_income_usd"] != nil || projected["annual_income_eur"] != nil || projected["photo_id"] != nil {
		t.Fatalf("expected created US observation without invented canonical facts, got %v", projected)
	}

}
