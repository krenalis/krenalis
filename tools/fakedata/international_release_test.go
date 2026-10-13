// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"strings"
	"testing"
)

// TestInternationalRelease checks the descriptor independently of acquisition.
func TestInternationalRelease(t *testing.T) {

	document := internationalRelease{
		Schema: "sw-country-release-v1", Release: "artificial-release", Country: "IT",
		Base:          internationalAssetReference{"data/base.json", strings.Repeat("a", 64)},
		Socioeconomic: internationalAssetReference{"data/socioeconomic.json", strings.Repeat("b", 64)},
		Photos:        internationalAssetReference{"photos/catalog.json", strings.Repeat("c", 64)},
	}
	input, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected descriptor fixture, got %v", err)
	}
	data, err := DecodeCountryReleaseData(t.Context(), bytes.NewReader(input))
	if err != nil {
		t.Fatalf("expected valid descriptor, got %v", err)
	}
	canonical, err := data.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical descriptor, got %v", err)
	}
	if !bytes.Equal(input, canonical) {
		t.Fatal("expected identical descriptor round trip, got altered fields")
	}
	want, err := data.LogicalDigest()
	if err != nil {
		t.Fatalf("expected descriptor digest, got %v", err)
	}
	reordered, err := DecodeCountryReleaseData(t.Context(), bytes.NewReader(reverseInternationalObjects(t, input)))
	if err != nil {
		t.Fatalf("expected reordered descriptor, got %v", err)
	}
	got, err := reordered.LogicalDigest()
	if err != nil {
		t.Fatalf("expected reordered digest, got %v", err)
	}
	if got != want {
		t.Fatal("expected order independent descriptor digest, got different digest")
	}

	for _, test := range []struct{ old, replacement string }{
		{`"schema":"sw-country-release-v1"`, `"schema":"sw-country-release-v2"`},
		{`"schema":`, `"other":0,"schema":`},
		{`"schema":`, `"\u0073chema":"sw-country-release-v1","schema":`},
		{`"release":"artificial-release"`, `"release":"invalid release"`},
		{`"country":"IT"`, `"country":"It"`},
		{`"country":"IT",`, ``},
		{`"base":{`, `"base":null,"other":{`},
		{`"path":"data/base.json"`, `"path":"../base.json"`},
		{`"path":"data/base.json"`, `"path":"/base.json"`},
		{`"path":"data/base.json"`, `"path":"https://host/base.json"`},
		{`"path":"data/base.json"`, `"path":"data/%62ase.json"`},
		{`"path":"data/base.json"`, `"path":"base.json","extra":1`},
		{strings.Repeat("a", 64), strings.Repeat("A", 64)},
		{strings.Repeat("a", 64), strings.Repeat("a", 63)},
	} {
		altered := bytes.Replace(input, []byte(test.old), []byte(test.replacement), 1)
		if bytes.Equal(input, altered) {
			t.Fatalf("expected descriptor substitution %s, got unchanged input", test.old)
		}
		_, err = DecodeCountryReleaseData(t.Context(), bytes.NewReader(altered))
		if err != nil {
			continue
		}
		t.Fatalf("expected descriptor rejected for %s, got success", test.replacement)
	}
	for _, altered := range [][]byte{
		append([]byte{0xef, 0xbb, 0xbf}, input...), append(bytes.Clone(input), input...),
	} {
		_, err = DecodeCountryReleaseData(t.Context(), bytes.NewReader(altered))
		if err != nil {
			continue
		}
		t.Fatal("expected invalid framing rejected, got success")
	}

}
