// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"os"
	"testing"
)

// TestInternationalApplicationFixtures keeps portable application assets
// reproducible from the artificial fixtures used by the decoder tests.
func TestInternationalApplicationFixtures(t *testing.T) {
	for name, document := range map[string]any{
		"international-photos-v1.json":        internationalPhotoFixture(),
		"international-socioeconomic-v1.json": internationalSocioeconomicFixture(t),
	} {
		got, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatalf("expected portable %s fixture, got %v", name, err)
		}
		want := internationalTestJSON(t, document)
		if !bytes.Equal(got, want) {
			t.Fatalf("expected reproducible artificial %s, got different JSON", name)
		}
	}
}
