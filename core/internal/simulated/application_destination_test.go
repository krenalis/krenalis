// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

func TestDestinationRecordSchema(t *testing.T) {
	properties := destinationSchema.Properties().Slice()
	wantNames := []string{"first_name", "last_name", "email", "gender", "birth_date", "current_age", "phone", "site_url", "address", "photo"}
	if len(properties) != len(wantNames) {
		t.Fatalf("expected %d destination properties, got %d", len(wantNames), len(properties))
	}
	for i, property := range properties {
		if property.Name != wantNames[i] {
			t.Fatalf("expected destination property %d to be %q, got %q", i, wantNames[i], property.Name)
		}
		if property.Nullable || property.UpdateRequired {
			t.Fatalf("expected %s to be non-nullable and update-optional, got nullable=%t updateRequired=%t",
				property.Name, property.Nullable, property.UpdateRequired)
		}
		if property.CreateRequired != (property.Name != "email") {
			t.Fatalf("expected %s CreateRequired=%t, got %t", property.Name, property.Name != "email", property.CreateRequired)
		}
	}
	if _, ok := destinationSchema.Properties().ByName("customer_id"); ok {
		t.Fatal("expected customer_id to be excluded from destination schema")
	}
	address, _ := destinationSchema.Properties().ByName("address")
	if got := address.Type.Properties().Slice(); len(got) != 5 {
		t.Fatalf("expected five address properties, got %d", len(got))
	}
	country, _ := address.Type.Properties().ByName("country")
	if country.Type.Semantic() != types.CountrySemantic || country.Type.CountryFormat() != types.ISO3166Alpha2 {
		t.Fatalf("expected address.country to use ISO alpha-2, got %+v", country.Type)
	}
	photo, _ := destinationSchema.Properties().ByName("photo")
	if photo.Type.Semantic() != types.URLSemantic {
		t.Fatalf("expected photo to use URL semantic, got %+v", photo.Type)
	}
}

func TestDestinationDocuments(t *testing.T) {
	attributes := completeDestinationAttributes()
	data, err := newDestinationRecordDocument("abc123", attributes)
	if err != nil {
		t.Fatalf("expected complete destination document, got %v", err)
	}
	decoded, err := types.Decode[map[string]any](bytes.NewReader(data), sourceSchema)
	if err != nil {
		t.Fatalf("expected destination document to decode as source, got %v", err)
	}
	if decoded["customer_id"] != "abc123" {
		t.Fatalf("expected customer_id %q, got %v", "abc123", decoded["customer_id"])
	}
	if _, ok := decoded["email"]; ok {
		t.Fatal("expected absent email to remain absent")
	}

	updated, err := updateDestinationRecordDocument("abc123", data, map[string]any{
		"current_age": 37,
		"address":     map[string]any{"municipality": "Milan"},
	})
	if err != nil {
		t.Fatalf("expected partial destination update, got %v", err)
	}
	decoded, err = types.Decode[map[string]any](bytes.NewReader(updated), sourceSchema)
	if err != nil {
		t.Fatalf("expected updated destination document to decode as source, got %v", err)
	}
	if decoded["current_age"] != 37 {
		t.Fatalf("expected current_age 37, got %v", decoded["current_age"])
	}
	if !reflect.DeepEqual(decoded["address"].(map[string]any)["street"], "Via Roma 1") ||
		decoded["address"].(map[string]any)["municipality"] != "Milan" {
		t.Fatalf("expected address patch to preserve and replace leaves, got %v", decoded["address"])
	}
	if _, err := updateDestinationRecordDocument("abc123", data, map[string]any{"email": nil}); err == nil {
		t.Fatal("expected explicit email null to be rejected")
	}
	if _, err := newDestinationRecordDocument("abc123", map[string]any{}); err == nil {
		t.Fatal("expected incomplete create to be rejected")
	}
}

func completeDestinationAttributes() map[string]any {
	return map[string]any{
		"first_name":  "Ada",
		"last_name":   "Rossi",
		"gender":      "female",
		"birth_date":  time.Date(1990, time.January, 2, 0, 0, 0, 0, time.UTC),
		"current_age": 36,
		"phone":       "+393331234567",
		"site_url":    "https://example.test/ada",
		"address":     map[string]any{"street": "Via Roma 1", "municipality": "Rome", "state_prov": "RM", "postal_code": "00100", "country": "IT"},
		"photo":       "https://krenalis.test/photos/catalog/256/face-000001.webp",
	}
}
