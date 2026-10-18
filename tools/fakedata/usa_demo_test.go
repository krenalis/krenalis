// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestUSAWorld checks deterministic photos and unchanged USA facts.
func TestUSAWorld(t *testing.T) {

	data, err := LoadUSAWorldData(t.Context(), "../..")
	if err != nil {
		t.Fatalf("expected verified USA demo data and 6250 image files, got %v", err)
	}
	namespace := IdentityNamespace{Name: "usa-layer1", Generation: 1}
	config := SourceInstanceConfig{ID: "demo-test", Version: USASourceVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1}
	world, err := NewUSAWorld(data, namespace, 2026, config)
	if err != nil {
		t.Fatalf("expected USA demo World, got %v", err)
	}
	original, err := newUSACountryWorld(data.country, namespace, 2026)
	if err != nil {
		t.Fatalf("expected original USA economic World, got %v", err)
	}
	if fmt.Sprintf("%x", data.country.socioeconomic.model.digest) !=
		"f7c8d650c6f3963afbb34a95780c38c1293f29f72df505bbe07d5760e1d48a54" {
		t.Fatal("expected unchanged approved ModelID, got drift")
	}
	if fmt.Sprintf("%x", data.catalog.checksum) != "c2c16abc817b1dc1e063b404263b7689f2735990bcdd65c8f63cbc3088f912c0" {
		t.Fatal("expected historical photo catalog identity, got drift")
	}
	for _, test := range []struct {
		index PersonIndex
		first string
		last  string
	}{
		{374054, "Susan", "Dang"},
		{18076, "Keith", "Smith"},
	} {

		records, err := world.Records(test.index)
		if err != nil {
			t.Fatalf("expected natural USA records at %d, got %v", test.index, err)
		}
		for _, record := range records {
			if record.Observation.FirstName == nil || record.Observation.LastName == nil ||
				*record.Observation.FirstName != test.first || *record.Observation.LastName != test.last {
				t.Fatalf("expected %s %s in both stored representations, got %+v", test.first, test.last, record.Observation)
			}
		}

	}
	activities := map[PrimaryActivity]bool{}
	baseline := map[PersonIndex][]USASourceObservation{}
	for index := PersonIndex(1); index <= 2048; index++ {

		person, err := world.adapter.world.person(index)
		if err != nil {
			t.Fatalf("expected photographed USA person, got %v", err)
		}
		previous, err := original.person(index)
		if err != nil {
			t.Fatalf("expected original USA person, got %v", err)
		}
		photoID := person.Person.PhotoID
		person.Person.PhotoID = ""
		if photoID == "" || !reflect.DeepEqual(person, previous) {
			t.Fatalf("expected only PhotoID to change at %d, got altered demographic or economic facts", index)
		}
		records, err := world.Records(index)
		if err != nil {
			t.Fatalf("expected USA source records with photos, got %v", err)
		}
		if len(records) != 2 || records[0].Observation.PhotoID == nil || records[1].Observation.PhotoID == nil ||
			*records[0].Observation.PhotoID != photoID || *records[1].Observation.PhotoID != photoID ||
			records[0].Observation.ID == records[1].Observation.ID ||
			*records[0].AnnualIncomeUSD != previous.AnnualIncomeUSD {
			t.Fatalf("expected two opaque observations with the same PhotoID and unchanged USD at %d, got mismatch", index)
		}
		activities[records[0].PrimaryActivity] = true
		baseline[index] = records

	}
	if len(activities) != 6 {
		t.Fatalf("expected all six primary activities, got %d", len(activities))
	}
	for index := PersonIndex(2048); index > 0; index-- {
		actual, err := world.Records(index)
		if err != nil {
			t.Fatalf("expected reverse traversal, got %v", err)
		}
		if !reflect.DeepEqual(actual, baseline[index]) {
			t.Fatalf("expected deterministic observations at %d, got drift", index)
		}
	}
	asset, image, err := data.Photo(t.Context(), *baseline[1][0].Observation.PhotoID)
	if err != nil {
		t.Fatalf("expected verified existing rendition, got %v", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(image)) != asset.ContentSHA256 || int64(len(image)) != asset.ContentLength {
		t.Fatal("expected original asset integrity, got mismatch")
	}
	changed := *data
	changed.root = t.TempDir()
	path := filepath.Join(changed.root, "tools/fakefacegen/output/faces-v1/derived/256")
	err = os.MkdirAll(path, 0700)
	if err != nil {
		t.Fatalf("expected isolated asset directory, got %v", err)
	}
	image = bytes.Clone(image)
	image[len(image)-1] ^= 1
	err = os.WriteFile(filepath.Join(path, asset.PhotoID+".webp"), image, 0600)
	if err != nil {
		t.Fatalf("expected isolated altered asset, got %v", err)
	}
	_, _, err = changed.Photo(t.Context(), asset.PhotoID)
	if err != nil {
		return
	}

	t.Fatal("expected altered rendition rejection, got image bytes")

}
