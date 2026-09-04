// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"errors"
	"testing"
)

// FuzzFixedBase32RoundTrip verifies Base32 round trips for arbitrary uint64 values.
func FuzzFixedBase32RoundTrip(f *testing.F) {

	for _, value := range []uint64{0, 1, 32, 1024, 0x123456789abcdef0, 1 << 63, ^uint64(0)} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value uint64) {

		token := encodeUint64Base32(value)
		decoded, err := decodeUint64Base32(token)
		if err != nil {
			t.Fatalf("expected nil error for %016x encoded as %q, got %v", value, token, err)
		}
		if decoded != value {
			t.Fatalf("expected %016x, got %016x", value, decoded)
		}

	})

}

// FuzzPersonIDParser verifies arbitrary parser inputs and canonical re-encoding of valid IDs.
func FuzzPersonIDParser(f *testing.F) {

	valid := "person_v1_krenalis-demo_00000001_4r0n8d46pgtwb"
	for _, id := range []string{
		valid,
		"",
		"person_v2_krenalis-demo_00000001_4r0n8d46pgtwb",
		"company_v1_krenalis-demo_00000001_ef434ynxgmwyh",
		"person_v1_krenalis-demo_00000001_3p92rz2h2crce",
		" person_v1_krenalis-demo_00000001_4r0n8d46pgtwb",
	} {
		f.Add(id)
	}

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	f.Fuzz(func(t *testing.T, id string) {

		index, err := PersonIndexFromID(namespace, id)
		if err != nil {
			known := errors.Is(err, ErrMalformedSyntheticID) ||
				errors.Is(err, ErrUnsupportedSyntheticIDVersion) ||
				errors.Is(err, ErrWrongEntityKind) ||
				errors.Is(err, ErrWrongIdentityNamespace) ||
				errors.Is(err, ErrUnassignedSyntheticID)
			if !known {
				t.Fatalf("unexpected error category for %q: %v", id, err)
			}
			return
		}

		encoded, err := PersonIDForIndex(namespace, index)
		if err != nil {
			t.Fatalf("expected nil forward error for parsed index %d, got %v", index, err)
		}
		if encoded != id {
			t.Fatalf("expected canonical re-encoding %q, got %q", id, encoded)
		}

	})

}

// FuzzPersonIDRoundTrip verifies forward and inverse mapping for arbitrary nonzero indices.
func FuzzPersonIDRoundTrip(f *testing.F) {

	for _, index := range []uint64{0, 1, 2, 42, 1537291, 1 << 63, ^uint64(0)} {
		f.Add(index)
	}
	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	f.Fuzz(func(t *testing.T, value uint64) {

		if value == 0 {
			_, err := PersonIDForIndex(namespace, 0)
			if err != nil {
				if !errors.Is(err, ErrInvalidIndex) {
					t.Fatalf("expected invalid index, got %v", err)
				}
				return
			}
			t.Fatal("expected invalid index, got nil")
		}

		index := PersonIndex(value)
		id, err := PersonIDForIndex(namespace, index)
		if err != nil {
			t.Fatalf("expected nil forward error for %d, got %v", index, err)
		}
		decoded, err := PersonIndexFromID(namespace, id)
		if err != nil {
			t.Fatalf("expected nil inverse error for %q, got %v", id, err)
		}
		if decoded != index {
			t.Fatalf("expected %d, got %d", index, decoded)
		}

	})

}
