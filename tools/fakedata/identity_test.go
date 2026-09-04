// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
)

// TestFixedBase32Golden verifies every fixed Base32 vector and its inverse.
func TestFixedBase32Golden(t *testing.T) {

	tests := []struct {
		value uint64
		token string
	}{
		{0x0000000000000000, "0000000000000"},
		{0x0000000000000001, "0000000000001"},
		{0x0000000000000020, "0000000000010"},
		{0x0000000000000400, "0000000000100"},
		{0x123456789abcdef0, "14d2pf2dbsqqg"},
		{0x8000000000000000, "8000000000000"},
		{0xffffffffffffffff, "fzzzzzzzzzzzz"},
	}

	for _, test := range tests {
		if got := encodeUint64Base32(test.value); got != test.token {
			t.Fatalf("expected %016x to encode as %s, got %s", test.value, test.token, got)
		}
		got, err := decodeUint64Base32(test.token)
		if err != nil {
			t.Fatalf("expected nil error for %s, got %v", test.token, err)
		}
		if got != test.value {
			t.Fatalf("expected %s to decode as %016x, got %016x", test.token, test.value, got)
		}
	}

}

// TestFixedBase32RejectsNoncanonicalTokens verifies exact length, alphabet, case, and leading digit rules.
func TestFixedBase32RejectsNoncanonicalTokens(t *testing.T) {

	invalid := []string{
		"000000000000",
		"00000000000000",
		"000000000000A",
		"000000000000i",
		"000000000000l",
		"000000000000o",
		"000000000000u",
		"g000000000000",
		"z000000000000",
	}
	for _, token := range invalid {
		_, err := decodeUint64Base32(token)
		if err != nil {
			if !errors.Is(err, ErrMalformedSyntheticID) {
				t.Fatalf("expected malformed ID for %q, got %v", token, err)
			}
			continue
		}
		t.Fatalf("expected malformed ID for %q, got nil", token)
	}

}

// TestIdentityKeyGolden verifies domain separation between person and company identity keys.
func TestIdentityKeyGolden(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	tests := []struct {
		kind      string
		digestHex string
		key       uint64
	}{
		{"person", "dc7879a3fbd84ec2ea7ae46ad2c83d6d93dfb3bd1a3b2287da838582ef554ce5", 0xdc7879a3fbd84ec2},
		{"company", "57d443366116152ef87c131700c86e19a708b660647ff60aae444523e5a2680a", 0x57d443366116152e},
	}

	for _, test := range tests {
		digest, key, err := identityKey(namespace, test.kind)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got := hex.EncodeToString(digest[:]); got != test.digestHex {
			t.Fatalf("expected %s digest %s, got %s", test.kind, test.digestHex, got)
		}
		if key != test.key {
			t.Fatalf("expected %s key %016x, got %016x", test.kind, test.key, key)
		}
	}

}

// TestIdentityNamespace verifies the complete namespace grammar.
func TestIdentityNamespace(t *testing.T) {

	valid := []struct {
		name       string
		generation uint32
	}{
		{"a", 1},
		{"krenalis-demo", 1},
		{"a0123456789012345678901234567890", math.MaxUint32},
	}
	for _, test := range valid {
		namespace, err := NewIdentityNamespace(test.name, test.generation)
		if err != nil {
			t.Fatalf("expected valid namespace %q, got %v", test.name, err)
		}
		if namespace.Name != test.name || namespace.Generation != test.generation {
			t.Fatalf("expected %q/%d, got %q/%d", test.name, test.generation, namespace.Name, namespace.Generation)
		}
	}

	invalid := []IdentityNamespace{
		{},
		{Name: "krenalis-demo"},
		{Name: "", Generation: 1},
		{Name: "Krenalis", Generation: 1},
		{Name: "0demo", Generation: 1},
		{Name: "demo_world", Generation: 1},
		{Name: "demo.world", Generation: 1},
		{Name: "é", Generation: 1},
		{Name: "a01234567890123456789012345678901", Generation: 1},
	}
	for _, namespace := range invalid {
		_, err := NewIdentityNamespace(namespace.Name, namespace.Generation)
		if err != nil {
			if !errors.Is(err, ErrInvalidIdentityNamespace) {
				t.Fatalf("expected invalid namespace error for %#v, got %v", namespace, err)
			}
			continue
		}
		t.Fatalf("expected invalid namespace error for %#v, got nil", namespace)
	}

}

// TestMix64Golden verifies every mixer vector and the normative inverse.
func TestMix64Golden(t *testing.T) {

	tests := []struct {
		input  uint64
		output uint64
	}{
		{0x0000000000000000, 0x0000000000000000},
		{0x0000000000000001, 0x5692161d100b05e5},
		{0x000000000000002a, 0xa759ea27d4727622},
		{0x123456789abcdef0, 0x9629f58e8ec5b906},
		{0xffffffffffffffff, 0xb4d055fcf2cbbd7b},
	}

	for _, test := range tests {
		if got := mix64(test.input); got != test.output {
			t.Fatalf("expected mix64(%016x) = %016x, got %016x", test.input, test.output, got)
		}
		if got := unmix64(test.output); got != test.input {
			t.Fatalf("expected unmix64(%016x) = %016x, got %016x", test.output, test.input, got)
		}
	}

	x := uint64(0x4d595df4d0f33173)
	for i := 0; i < 100_000; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		if got := unmix64(mix64(x)); got != x {
			t.Fatalf("expected mixer round-trip for %016x, got %016x", x, got)
		}
	}

}

// TestSyntheticIDErrors verifies parser and argument error categories.
func TestSyntheticIDErrors(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	validID := "person_v1_krenalis-demo_00000001_4r0n8d46pgtwb"
	longVersionID := "person_v" + strings.Repeat("1", 1<<16) + "_krenalis-demo_00000001_4r0n8d46pgtwb"
	tests := []struct {
		name      string
		namespace IdentityNamespace
		id        string
		want      error
	}{
		{"invalid namespace", IdentityNamespace{}, validID, ErrInvalidIdentityNamespace},
		{"wrong kind", namespace, "company_v1_krenalis-demo_00000001_ef434ynxgmwyh", ErrWrongEntityKind},
		{"wrong name", namespace, "person_v1_another_00000001_4r0n8d46pgtwb", ErrWrongIdentityNamespace},
		{"wrong generation", namespace, "person_v1_krenalis-demo_00000002_4pgge52m1np1x", ErrWrongIdentityNamespace},
		{
			"unsupported version",
			namespace,
			"person_v2_krenalis-demo_00000001_4r0n8d46pgtwb",
			ErrUnsupportedSyntheticIDVersion,
		},
		{"unassigned", namespace, "person_v1_krenalis-demo_00000001_3p92rz2h2crce", ErrUnassignedSyntheticID},
		{"missing version", namespace, "person", ErrMalformedSyntheticID},
		{"missing namespace", namespace, "person_v1", ErrMalformedSyntheticID},
		{"missing generation", namespace, "person_v1_krenalis-demo", ErrMalformedSyntheticID},
		{"missing token", namespace, "person_v1_krenalis-demo_00000001", ErrMalformedSyntheticID},
		{"space", namespace, " " + validID, ErrMalformedSyntheticID},
		{"trailing segment", namespace, validID + "_x", ErrMalformedSyntheticID},
		{"many segments", namespace, strings.Repeat("_", 1<<16), ErrMalformedSyntheticID},
		{"uppercase", namespace, "person_v1_krenalis-demo_00000001_4R0n8d46pgtwb", ErrMalformedSyntheticID},
		{"generation zero", namespace, "person_v1_krenalis-demo_00000000_4r0n8d46pgtwb", ErrMalformedSyntheticID},
		{"generation width", namespace, "person_v1_krenalis-demo_1_4r0n8d46pgtwb", ErrMalformedSyntheticID},
		{"invalid version", namespace, "person_current_krenalis-demo_00000001_4r0n8d46pgtwb", ErrMalformedSyntheticID},
		{"invalid version digits", namespace, "person_v1x_krenalis-demo_00000001_4r0n8d46pgtwb", ErrMalformedSyntheticID},
		{"long unsupported version", namespace, longVersionID, ErrUnsupportedSyntheticIDVersion},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			_, err := PersonIndexFromID(test.namespace, test.id)
			if err != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("expected %v, got %v", test.want, err)
				}
				return
			}
			t.Fatalf("expected %v, got nil", test.want)

		})

	}

	_, err := PersonIDForIndex(namespace, 0)
	if err != nil {
		if !errors.Is(err, ErrInvalidIndex) {
			t.Fatalf("expected invalid index, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid index, got nil")
	}
	_, err = PersonIDForIndex(IdentityNamespace{}, 1)
	if err != nil {
		if !errors.Is(err, ErrInvalidIdentityNamespace) {
			t.Fatalf("expected invalid namespace, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid namespace, got nil")
	}
	_, err = entityIDForIndex(namespace, "Person", 1)
	if err != nil {
		if !errors.Is(err, ErrWrongEntityKind) {
			t.Fatalf("expected wrong entity kind, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected wrong entity kind, got nil")
	}
	_, err = entityIndexFromID(namespace, "Person", validID)
	if err != nil {
		if !errors.Is(err, ErrWrongEntityKind) {
			t.Fatalf("expected wrong entity kind, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected wrong entity kind, got nil")
	}

}

// TestSyntheticIDGolden verifies authoritative person and company mappings in both directions.
func TestSyntheticIDGolden(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	tests := []struct {
		index PersonIndex
		id    string
	}{
		{1, "person_v1_krenalis-demo_00000001_1z4yz7emgwe48"},
		{2, "person_v1_krenalis-demo_00000001_3hhcxdgc68vsb"},
		{42, "person_v1_krenalis-demo_00000001_4r0n8d46pgtwb"},
		{1537291, "person_v1_krenalis-demo_00000001_c5m6wxatk5862"},
		{math.MaxUint64, "person_v1_krenalis-demo_00000001_7dzg0z9r9ekm6"},
	}

	for _, test := range tests {
		id, err := PersonIDForIndex(namespace, test.index)
		if err != nil {
			t.Fatalf("expected nil error for index %d, got %v", test.index, err)
		}
		if id != test.id {
			t.Fatalf("expected index %d ID %s, got %s", test.index, test.id, id)
		}
		index, err := PersonIndexFromID(namespace, test.id)
		if err != nil {
			t.Fatalf("expected nil error for %s, got %v", test.id, err)
		}
		if index != test.index {
			t.Fatalf("expected ID %s index %d, got %d", test.id, test.index, index)
		}
	}

	generationTwo := IdentityNamespace{Name: "krenalis-demo", Generation: 2}
	id, err := PersonIDForIndex(generationTwo, 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "person_v1_krenalis-demo_00000002_4pgge52m1np1x"; id != want {
		t.Fatalf("expected %s, got %s", want, id)
	}

	companyID, err := entityIDForIndex(namespace, "company", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "company_v1_krenalis-demo_00000001_ef434ynxgmwyh"; companyID != want {
		t.Fatalf("expected %s, got %s", want, companyID)
	}
	companyIndex, err := entityIndexFromID(namespace, "company", companyID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if companyIndex != 1 {
		t.Fatalf("expected company index 1, got %d", companyIndex)
	}

}

// TestSyntheticIDNamespaceAndKindSeparation verifies separation by namespace and entity kind.
func TestSyntheticIDNamespaceAndKindSeparation(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	person, err := entityIDForIndex(namespace, "person", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	company, err := entityIDForIndex(namespace, "company", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	otherName, err := entityIDForIndex(IdentityNamespace{Name: "other", Generation: 1}, "person", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	otherGeneration, err := entityIDForIndex(IdentityNamespace{Name: "krenalis-demo", Generation: 2}, "person", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if person == company || person == otherName || person == otherGeneration {
		t.Fatalf("expected domain-separated IDs, got %q, %q, %q, and %q", person, company, otherName, otherGeneration)
	}

}

// TestSyntheticIDRoundTrip verifies bijective behavior over broad deterministic samples.
func TestSyntheticIDRoundTrip(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	indices := []PersonIndex{1, 2, 42, 1537291, math.MaxUint64}
	x := uint64(0x243f6a8885a308d3)
	for i := 0; i < 20_000; i++ {
		x = x*2862933555777941757 + 3037000493
		if x != 0 {
			indices = append(indices, PersonIndex(x))
		}
	}

	seen := make(map[string]PersonIndex, len(indices))
	for _, index := range indices {
		id, err := PersonIDForIndex(namespace, index)
		if err != nil {
			t.Fatalf("expected nil error for index %d, got %v", index, err)
		}
		if previous, ok := seen[id]; ok && previous != index {
			t.Fatalf("indices %d and %d produced duplicate ID %s", previous, index, id)
		}
		seen[id] = index
		decoded, err := PersonIndexFromID(namespace, id)
		if err != nil {
			t.Fatalf("expected nil inverse error for index %d, got %v", index, err)
		}
		if decoded != index {
			t.Fatalf("expected index %d, got %d", index, decoded)
		}
	}

}
