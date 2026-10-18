// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"maps"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSocioeconomicDestinationDocuments checks preservation on partial updates
// and rejection of every canonical socioeconomic property on creates/updates.
func TestSocioeconomicDestinationDocuments(t *testing.T) {

	canonical := map[string]any{
		"education_level": "tertiary", "primary_activity": "employee", "annual_income_eur": uint(0),
		"job_family": "technical-and-scientific", "industry": "information-and-communication", "job_level": "manager",
	}
	document := completeDestinationAttributes()
	document["customer_id"] = "abc123"
	maps.Copy(document, canonical)
	stored, err := types.MarshalValidate(document, sourceSchema)
	if err != nil {
		t.Fatalf("expected generated Source document, got %v", err)
	}
	updated, err := updateDestinationRecordDocument("abc123", stored, map[string]any{
		"first_name": "Bea", "address": map[string]any{"municipality": "Milan"},
	}, sourceSchema)
	if err != nil {
		t.Fatalf("expected writable partial update, got %v", err)
	}
	decoded, err := types.Decode[map[string]any](bytes.NewReader(updated), sourceSchema)
	if err != nil {
		t.Fatalf("expected updated Source document, got %v", err)
	}
	for name, value := range canonical {
		if decoded[name] != value {
			t.Fatalf("expected preserved %s=%v, got %v", name, value, decoded[name])
		}
	}
	if decoded["first_name"] != "Bea" || decoded["address"].(map[string]any)["municipality"] != "Milan" {
		t.Fatalf("expected ordinary writable properties updated, got %v", decoded)
	}
	for name, value := range canonical {

		create := completeDestinationAttributes()
		create[name] = value
		_, err := newDestinationRecordDocument("new123", create)
		if err == nil {
			t.Fatalf("expected Destination create to reject %s, got success", name)
		}
		_, err = updateDestinationRecordDocument("abc123", stored, map[string]any{name: value}, sourceSchema)
		if err != nil {
			continue
		}
		t.Fatalf("expected Destination update to reject %s even with unchanged value, got success", name)

	}
	created, err := newDestinationRecordDocument("new123", completeDestinationAttributes())
	if err != nil {
		t.Fatalf("expected Destination create without socioeconomic fields, got %v", err)
	}
	decoded, err = types.Decode[map[string]any](bytes.NewReader(created), sourceSchema)
	if err != nil {
		t.Fatalf("expected readable Destination-created record, got %v", err)
	}
	for name := range canonical {
		if _, ok := decoded[name]; ok {
			t.Fatalf("expected Destination-created %s absent, got %v", name, decoded[name])
		}
	}
	_, err = updateDestinationRecordDocument("new123", created, map[string]any{"unknown_property": "invalid"}, sourceSchema)
	if err != nil {
		return
	}

	t.Fatal("expected unknown Destination property rejection, got success")

}

// TestSocioeconomicSourceDocuments checks optional fields, exact integers,
// canonical labels and the existing behavior for unknown source properties.
func TestSocioeconomicSourceDocuments(t *testing.T) {

	legacy := completeDestinationAttributes()
	legacy["customer_id"] = "abc123"
	for _, income := range []uint{0, 1, 1<<32 - 1} {

		document := maps.Clone(legacy)
		document["education_level"] = "upper-secondary"
		document["primary_activity"] = "self-employed"
		document["annual_income_eur"] = income
		document["job_family"] = "skilled-trades"
		document["industry"] = "construction"
		encoded, err := types.MarshalValidate(document, sourceSchema)
		if err != nil {
			t.Fatalf("expected valid Source document with Income %d, got %v", income, err)
		}
		decoded, err := types.Decode[map[string]any](bytes.NewReader(encoded), sourceSchema)
		if err != nil {
			t.Fatalf("expected Source document with Income %d to decode, got %v", income, err)
		}
		if decoded["annual_income_eur"] != income {
			t.Fatalf("expected exact unsigned Income %d, got %v", income, decoded["annual_income_eur"])
		}
		if _, ok := decoded["job_level"]; ok {
			t.Fatalf("expected absent self-employed job_level, got %v", decoded["job_level"])
		}

	}
	encoded, err := types.MarshalValidate(legacy, sourceSchema)
	if err != nil {
		t.Fatalf("expected legacy document without socioeconomic fields, got %v", err)
	}
	decoded, err := types.Decode[map[string]any](bytes.NewReader(encoded), sourceSchema)
	if err != nil {
		t.Fatalf("expected readable legacy document, got %v", err)
	}
	for _, name := range []string{"education_level", "primary_activity", "annual_income_eur",
		"job_family", "industry", "job_level"} {
		if _, ok := decoded[name]; ok {
			t.Fatalf("expected legacy %s to remain absent, got %v", name, decoded[name])
		}
	}
	for name, invalid := range map[string][]json.Value{
		"annual_income_eur": {json.Value(`-1`), json.Value(`0.5`), json.Value(`4294967296`),
			json.Value(`"0"`), json.Value(`null`)},
		"education_level":  {json.Value(`"unknown"`), json.Value(`""`), json.Value(`null`)},
		"primary_activity": {json.Value(`"Employee"`), json.Value(`"self_employed"`), json.Value(`null`)},
		"job_family":       {json.Value(`"unknown"`), json.Value(`""`), json.Value(`null`)},
		"industry":         {json.Value(`"unknown"`), json.Value(`""`), json.Value(`null`)},
		"job_level":        {json.Value(`"unknown"`), json.Value(`""`), json.Value(`null`)},
	} {
		for _, value := range invalid {

			fields := map[string]json.Value{}
			err := json.Unmarshal(encoded, &fields)
			if err != nil {
				t.Fatalf("expected legacy JSON fields, got %v", err)
			}
			fields[name] = value
			data, err := json.Marshal(fields)
			if err != nil {
				t.Fatalf("expected invalid-value document JSON, got %v", err)
			}
			_, err = types.Decode[map[string]any](bytes.NewReader(data), sourceSchema)
			if err == nil {
				t.Fatalf("expected %s=%s to be rejected, got success", name, value)
			}

		}
	}
	fields := map[string]json.Value{}
	err = json.Unmarshal(encoded, &fields)
	if err != nil {
		t.Fatalf("expected legacy document JSON, got %v", err)
	}
	fields["unknown_property"] = json.Value(`"invalid"`)
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("expected unknown-property document JSON, got %v", err)
	}
	_, err = types.Decode[map[string]any](bytes.NewReader(data), sourceSchema)
	if err == nil {
		t.Fatal("expected unknown Source property to be rejected, got success")
	}

}
