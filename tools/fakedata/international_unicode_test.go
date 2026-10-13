// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// TestInternationalItalianText checks D1-B without regenerating ASCII atoms.
func TestInternationalItalianText(t *testing.T) {

	for _, value := range []string{"Émilie", "D'Angelo", "Gian Luca", "Sant'Anna", "A\u0305", "XXV Aprile"} {
		_, err := italianDisplayKey(value)
		if err != nil {
			t.Fatalf("expected supported display %q, got %v", value, err)
		}
	}
	for _, value := range []string{
		"E\u0301milie", " Émilie", "Gian  Luca", "Luca2", "A.B", "D’Angelo", "ß", "Ł", "Ø", "😀",
		"\u0301A", "A'\u0301", "A\u034f", "A\u200b", "A\ufffd", "'", "-",
	} {
		_, err := italianDisplayKey(value)
		if err != nil {
			continue
		}
		t.Fatalf("expected rejected display %q, got success", value)
	}
	for _, value := range []string{"a0", "a-b", "12-3"} {
		if !internationalASCIIAtom(value) {
			t.Fatalf("expected valid ASCII atom %q, got rejection", value)
		}
	}
	for _, value := range []string{"", "a--b", "-a", "a-", "A", "à", "a_b"} {
		if internationalASCIIAtom(value) {
			t.Fatalf("expected invalid ASCII atom %q, got success", value)
		}
	}

	atom := "same-atom"
	t.Run("unsorted", func(t *testing.T) {
		entries := []internationalName{{"À", &atom, 1}, {"Á", &atom, 2}, {"B", &atom, 4}}
		err := validateItalianNames(entries, 3, 1, 1, 1)
		if err != nil {
			return
		}
		t.Fatal("expected unsorted names rejected, got success")
	})
	// Sort by D, where the ASCII B precedes accented letters.
	entries := []internationalName{{"B", &atom, 4}, {"À", &atom, 1}, {"Á", &atom, 2}}
	err := validateItalianNames(entries, 3, 1, 1, 1)
	if err != nil {
		t.Fatalf("expected atom collisions accepted, got %v", err)
	}
	entries[2].Display = "à"
	err = validateItalianNames(entries, 3, 1, 1, 1)
	if err != nil {
		return
	}
	t.Fatal("expected folded display collision rejected, got success")

}

// TestInternationalUnicode16 verifies every admitted scalar and the complete
// normalization closure against pinned Unicode-16 data.
func TestInternationalUnicode16(t *testing.T) {

	data, err := os.ReadFile("testdata/unicode16/repertoire.json")
	if err != nil {
		t.Fatalf("expected pinned Unicode-16 fixture, got %v", err)
	}

	var fixture struct {
		Version    string `json:"version"`
		Characters []struct {
			Code    rune   `json:"code"`
			Allowed bool   `json:"allowed"`
			CCC     uint8  `json:"ccc"`
			NFD     string `json:"nfd"`
			NFC     string `json:"nfc"`
			Fold    string `json:"fold"`
		} `json:"characters"`
		Compositions []struct {
			Input  string `json:"input"`
			Output string `json:"output"`
		} `json:"compositions"`
		Sequences []struct {
			Input  string `json:"input"`
			Output string `json:"output"`
		} `json:"sequences"`
	}
	err = json.Unmarshal(data, &fixture)
	if err != nil {
		t.Fatalf("expected Unicode-16 fixture decoded, got %v", err)
	}
	if fixture.Version != "16.0.0" || len(fixture.Characters) != 677 || len(fixture.Compositions) != 505 {
		t.Fatalf("expected complete Unicode-16 closure, got %s/%d/%d",
			fixture.Version, len(fixture.Characters), len(fixture.Compositions))
	}

	var repertoire strings.Builder
	for _, row := range fixture.Characters {
		value := string(row.Code)
		if row.Allowed {
			repertoire.WriteRune(row.Code)
		}
		if norm.NFD.String(value) != row.NFD || norm.NFC.String(value) != row.NFC || italianNFC(value) != row.NFC ||
			norm.NFD.PropertiesString(value).CCC() != row.CCC || cases.Fold().String(value) != row.Fold {
			t.Fatalf("expected Unicode-16 normalization, CCC and folding at U+%04X, got divergence", row.Code)
		}
	}
	if repertoire.String() != italianUnicode16Repertoire {
		t.Fatal("expected exact pinned Unicode-16 display repertoire, got divergence")
	}

	for _, row := range fixture.Compositions {
		if got := italianNFC(row.Input); got != row.Output || norm.NFC.String(row.Input) != row.Output {
			t.Fatalf("expected Unicode-16 composition %q, got %q", row.Output, got)
		}
	}
	for _, row := range fixture.Sequences {
		if got := italianNFC(row.Input); got != row.Output {
			t.Fatalf("expected ordinary Unicode-16 NFC %q, got %q", row.Output, got)
		}
	}

	// All pairwise mark orderings exercise blocking and canonical combining
	// classes; the scalar and composition checks cover the normalizer's tables.
	for _, a := range fixture.Characters {
		if a.Code < 0x300 || a.Code > 0x36f {
			continue
		}
		for _, b := range fixture.Characters {
			if b.Code < 0x300 || b.Code > 0x36f || len([]rune(a.NFD)) != 1 || len([]rune(b.NFD)) != 1 {
				continue
			}
			input := "A" + string(a.Code) + string(b.Code)
			want := "A" + a.NFD + b.NFD
			if b.CCC != 0 && a.CCC > b.CCC {
				want = "A" + b.NFD + a.NFD
			}
			if got := norm.NFD.String(input); got != want {
				t.Fatalf("expected Unicode-16 mark order %q, got %q", want, got)
			}
		}
	}

}
