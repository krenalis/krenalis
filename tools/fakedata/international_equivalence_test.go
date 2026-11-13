// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestInternationalDocumentIdentities tests transport invariance, closed
// projections and isolation using full base data and an artificial L2 model.
func TestInternationalDocumentIdentities(t *testing.T) {

	base, err := readItalianBaseFixture(t)
	if err != nil {
		t.Fatalf("expected base conversion, got %v", err)
	}
	canonical, err := base.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected base bytes, got %v", err)
	}
	fingerprints, err := base.Fingerprints()
	if err != nil {
		t.Fatalf("expected six fingerprints, got %v", err)
	}

	digest, err := base.LogicalDigest()
	if err != nil {
		t.Fatalf("expected base digest, got %v", err)
	}
	// Independently reproduced using Python integer JSON and KFD1 framing.
	wantDigest := "64a8f6875f5a4b6c6b84fbc449d0f1c4a5f8586650f057b5c54282fb70ae948b"
	if got := fmt.Sprintf("%x", digest); got != wantDigest {
		t.Fatalf("expected qualified base logical digest, got %s", got)
	}
	for i, want := range []string{
		"97643544aa55574db056156bb9c08950fd1707b46cdccc73eaffb4cf5954234a",
		"5d562860c3428a4ad306bff29923cfc61177f12cc5a5a2c86f68d07bd8f81d05",
		"1d6ef98a75ff0b686556528e953f8551cf2e68890719c83d35fd97d0c22a4e9d",
		"d18bea32cfb5140d6a0f6f6371400867f129ea96d2d104d4a2bce69a65072c85",
		"58bcb1f2d2e3d8451fa6f1321af1f60741603effddd498a8f8e080d88680f134",
		"42a648f63524f459da9c9f00dc3ee2dbf155ef7de8510b07ee517b9ad08fc340",
	} {
		if got := fmt.Sprintf("%x", fingerprints[i]); got != want {
			t.Fatalf("expected fingerprint %s at family %d, got %s", want, i, got)
		}
	}

	altered := reverseInternationalObjects(t, canonical)
	decoded, err := DecodeItalianBaseData(t.Context(), bytes.NewReader(altered))
	if err != nil {
		t.Fatalf("expected reordered document accepted, got %v", err)
	}
	got, err := decoded.LogicalDigest()
	if err != nil {
		t.Fatalf("expected reordered digest, got %v", err)
	}
	gotFingerprints, err := decoded.Fingerprints()
	if err != nil {
		t.Fatalf("expected reordered fingerprints, got %v", err)
	}
	if got != digest || gotFingerprints != fingerprints {
		t.Fatal("expected whitespace/property-order invariant identities, got divergence")
	}
	for _, test := range []struct {
		name   string
		family int
		mutate func(*internationalBase)
	}{
		{"release", -1, func(d *internationalBase) {
			d.Release = "other-release"
		}},
		{"definition", -1, func(d *internationalBase) {
			s := "description"
			d.Genders.Categories[0].Definition = &s
		}},
		{"age", 0, func(d *internationalBase) {
			d.Ages[0].Weight++
		}},
		{"first", 1, func(d *internationalBase) {
			s := "new-atom"
			d.FirstNames[0].Entries[0].ASCIIAtom = &s
		}},
		{"last", 2, func(d *internationalBase) {
			s := "new-atom"
			d.LastNames[0].ASCIIAtom = &s
		}},
		{"place", 3, func(d *internationalBase) {
			d.Geography.Places[0].Population++
		}},
		{"postal", 4, func(d *internationalBase) {
			d.Geography.PostalCodes[d.Geography.Places[0].ID] = []string{"00001"}
		}},
		{"street", 5, func(d *internationalBase) {
			a := &d.StreetNames.Entries[0]
			for i := range d.StreetNames.Entries {
				b := &d.StreetNames.Entries[i]
				if a.Weight != b.Weight {
					a.Weight, b.Weight = b.Weight, a.Weight
					break
				}
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			d, err := DecodeItalianBaseData(t.Context(), bytes.NewReader(canonical))
			if err != nil {
				t.Fatalf("expected fresh base, got %v", err)
			}
			test.mutate(&d.document)
			err = validateItalianBase(&d.document)
			if err != nil {
				t.Fatalf("expected valid identity mutation, got %v", err)
			}
			got, err := d.Fingerprints()
			if err != nil {
				t.Fatalf("expected modified fingerprints, got %v", err)
			}
			for i := range got {
				if (got[i] != fingerprints[i]) != (i == test.family) {
					t.Fatalf("expected only family %d changed, got difference at %d", test.family, i)
				}
			}
			changed, err := d.LogicalDigest()
			if err != nil {
				t.Fatalf("expected modified digest, got %v", err)
			}
			if changed == digest {
				t.Fatal("expected whole-document digest changed, got original")
			}
		})
	}

	model := internationalSocioeconomicFixture(t)
	encoded, err := internationalCanonical(model)
	if err != nil {
		t.Fatalf("expected artificial model JSON, got %v", err)
	}
	first, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("expected artificial model accepted, got %v", err)
	}
	second, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(reverseInternationalObjects(t, encoded)))
	if err != nil {
		t.Fatalf("expected reordered model accepted, got %v", err)
	}
	firstDigest, err := first.LogicalDigest()
	if err != nil {
		t.Fatalf("expected model digest, got %v", err)
	}
	secondDigest, err := second.LogicalDigest()
	if err != nil {
		t.Fatalf("expected reordered digest, got %v", err)
	}
	if firstDigest != secondDigest || !reflect.DeepEqual(first.model, second.model) {
		t.Fatal("expected model identities independent of JSON layout, got divergence")
	}
	second.document.Release = "metadata-only-revision"
	definition := "An explanatory category definition"
	second.document.Dimensions.Education.Categories[0].Definition = &definition
	metadata, err := second.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected metadata variation, got %v", err)
	}
	changed, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(metadata))
	if err != nil {
		t.Fatalf("expected valid metadata variation, got %v", err)
	}
	changedDigest, err := changed.LogicalDigest()
	if err != nil {
		t.Fatalf("expected metadata digest, got %v", err)
	}
	if changedDigest == firstDigest || !reflect.DeepEqual(first.model, changed.model) {
		t.Fatal("expected changed document digest and unchanged frozen model identities, got divergence")
	}

}

// reverseInternationalObjects changes every object's order, escaping and
// whitespace, retaining arrays and exact numeric tokens through RawMessage.
func reverseInternationalObjects(t *testing.T, input []byte) []byte {
	t.Helper()
	trimmed := bytes.TrimSpace(input)
	if trimmed[0] == '{' {
		var fields map[string]json.Value
		err := json.Unmarshal(trimmed, &fields)
		if err != nil {
			t.Fatalf("expected object fixture, got %v", err)
		}
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		slices.Reverse(keys)
		out := []byte("{\n")
		for i, key := range keys {
			if i > 0 {
				out = append(out, ',', '\n')
			}
			out = append(out, fmt.Sprintf("  \"\\u%04x%s\" : ", key[0], key[1:])...)
			out = append(out, reverseInternationalObjects(t, fields[key])...)
		}
		return append(out, '\n', '}')
	}
	if trimmed[0] == '[' {
		var entries []json.Value
		err := json.Unmarshal(trimmed, &entries)
		if err != nil {
			t.Fatalf("expected array fixture, got %v", err)
		}
		out := []byte("[ ")
		for i, entry := range entries {
			if i > 0 {
				out = append(out, ',', ' ')
			}
			out = append(out, reverseInternationalObjects(t, entry)...)
		}
		return append(out, ' ', ']')
	}
	return trimmed
}
