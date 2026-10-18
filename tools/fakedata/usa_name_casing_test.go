// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestUSANaturalNameCatalogGeneration checks the adopted loader against the
// historical JSON evidence without admitting a second production binding.
func TestUSANaturalNameCatalogGeneration(t *testing.T) {

	const directory = "../../synthetic-world-usa-l1-us-2026a-BASE-v2-CANDIDATE/"
	pins := map[string]string{
		"assets/base.json":        "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80",
		"assets/first-names.json": "643795bf573521ce6f2902656d85cb11bdc3fcc9a9ef667d22ded9ca5281941e",
		"assets/surnames.json":    "2f44ec588fc1cee43b2a222e36e68ac2222fc16ca8adf0f8c16175d7fe800385",
	}
	read := func(name string, document any) []byte {

		encoded, err := os.ReadFile(directory + name)
		if err != nil {
			t.Fatalf("expected candidate %s, got %v", name, err)
		}
		if pin, ok := pins[name]; ok && fmt.Sprintf("%x", sha256.Sum256(encoded)) != pin {
			t.Fatalf("expected pinned candidate %s, got %x", name, sha256.Sum256(encoded))
		}
		err = decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), document)
		if err != nil {
			t.Fatalf("expected closed candidate %s, got %v", name, err)
		}

		return encoded
	}
	var document usaBaseDocument
	read("assets/base.json", &document)
	var first usaFirstNamesV2Document
	firstBytes := read("assets/first-names.json", &first)
	var last usaSurname50KDocument
	lastBytes := read("assets/surnames.json", &last)
	current := readUSACountryData(t)
	var previous usaBaseDocument
	oldBytes, err := os.ReadFile("../../synthetic-world-usa-l1-us-2026a-BASE-v1-CANDIDATE/assets/base.json")
	if err != nil {
		t.Fatalf("expected historical evidence, got %v", err)
	}
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(oldBytes), &previous)
	if err != nil {
		t.Fatalf("expected historical base JSON, got %v", err)
	}
	t.Run("reject BASE-v1", func(t *testing.T) {
		_, err := decodeUSABaseCandidate(t.Context(), bytes.NewReader(oldBytes), readUSACompletionEvidence(t))
		if err != nil {
			return
		}
		t.Fatal("expected BASE-v1 rejection, got admission")
	})

	base := current.base
	if base.firstNames.identity != sha256.Sum256(firstBytes) || base.lastNames.identity != sha256.Sum256(lastBytes) {
		t.Fatal("expected adopted catalog identities, got different content")
	}

	// Compare every catalog position with the historical evidence.
	if len(first.Groups) != 10 || len(document.FirstNames) != 10 || len(last.Entries) != 50000 ||
		!reflect.DeepEqual(last.Entries, document.LastNames) {
		t.Fatal("expected ten first-name groups and 50000 matching surnames, got different coverage")
	}
	for group := 0; group <= 10; group++ {

		old, entries := previous.LastNames, document.LastNames
		if group < 10 {
			old, entries = previous.FirstNames[group].Entries, document.FirstNames[group].Entries
			if first.Groups[group].Gender != Gender(document.FirstNames[group].Gender) ||
				first.Groups[group].Cohort != document.FirstNames[group].Cohort ||
				!reflect.DeepEqual(first.Groups[group].Entries, entries) {
				t.Fatalf("expected base/catalog first-name agreement at group %d, got mismatch", group)
			}
		}
		if len(entries) != len(old) {
			t.Fatalf("expected %d catalog positions, got %d", len(old), len(entries))
		}
		for i, entry := range entries {

			if strings.ToUpper(entry.Display) != old[i].Display || entry.Display == old[i].Display {
				t.Fatalf("expected naturally cased original name at %d/%d, got %q", group, i, entry.Display)
			}
			entry.Display = old[i].Display
			if !reflect.DeepEqual(entry, old[i]) {
				t.Fatalf("expected unchanged atom and weight at %d/%d, got mismatch", group, i)
			}

		}

	}
	oldRest, newRest := previous, document
	oldRest.FirstNames, newRest.FirstNames = nil, nil
	oldRest.LastNames, newRest.LastNames = nil, nil
	newRest.Release = oldRest.Release
	if !reflect.DeepEqual(oldRest, newRest) || document.Release != "us-2026a-layer1-v2-candidate" {
		t.Fatal("expected unchanged non-nominal base and new release, got mismatch")
	}

	if fmt.Sprintf("%x", base.digest) != "a21b1f2fcc73d7e34bc7845474fe8350f7aa5d6420e1ae4765c7aa51f2108771" {
		t.Fatalf("expected adopted base logical identity, got %x", base.digest)
	}

	var compatibility struct {
		Schema             string   `json:"schema"`
		Country            string   `json:"country"`
		SourceBaseSHA256   string   `json:"sourceBaseSHA256"`
		SourceProfile      string   `json:"sourceProfile"`
		SourceFingerprints []string `json:"sourceFingerprints" max:"6"`
		Contract           string   `json:"contract"`
		Admission          string   `json:"admission"`
	}
	read("sampling-compatibility.json", &compatibility)
	if len(compatibility.SourceFingerprints) != 6 {
		t.Fatalf("expected six sampling dependencies, got %d", len(compatibility.SourceFingerprints))
	}
	for i, encoded := range compatibility.SourceFingerprints {
		fingerprint, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("expected encoded dependency, got %v", err)
		}
		if fmt.Sprintf("%x", fingerprint) != usaBaseSamplingFingerprints[i] {
			t.Fatalf("expected unchanged dependency %d, got %x", i, fingerprint)
		}
	}
	for _, pin := range usaCountrySourcePins() {
		if strings.Contains(pin.Path, "BASE-v1") || strings.Contains(pin.Path, "FIRSTNAMES-v2") ||
			strings.Contains(pin.Path, "SURNAMES-50K-v2") {
			t.Fatalf("expected only current nominal inputs, got %s", pin.Path)
		}
	}
	t.Logf("adopted base logical digest: %x; checked 60000 catalog entries", base.digest)

}
