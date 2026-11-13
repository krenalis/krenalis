// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestInternationalPhotoBounds rejects resource growth at the transport edge.
func TestInternationalPhotoBounds(t *testing.T) {

	face := `{"id":"a","gender":"female","requestedAge":1,"renditions":[]}`
	input := `{"faces":[` + strings.Repeat(face+",", maxFaceCatalogRecords) + face + `]}`
	_, err := DecodePhotoCatalogData(t.Context(), strings.NewReader(input))
	if err != nil {
		if !strings.Contains(err.Error(), "array exceeds 100000 entries") {
			t.Fatalf("expected candidate bound before semantic validation, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected excessive candidate count rejected, got success")
	}
	document := internationalPhotoFixture()
	document.Faces[0].Renditions = make([]internationalRendition, 17)
	for i := range document.Faces[0].Renditions {
		document.Faces[0].Renditions[i] = internationalRendition{
			Width: uint32(i + 1), Height: 256, Path: fmt.Sprintf("fixture/%d.webp", i),
		}
	}
	_, err = DecodePhotoCatalogData(t.Context(), bytes.NewReader(internationalTestJSON(t, document)))
	if err != nil {
		if !strings.Contains(err.Error(), "array exceeds 16 entries") {
			t.Fatalf("expected rendition allocation bound, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected excessive rendition count rejected, got success")
	}
	reader := io.MultiReader(bytes.NewReader(internationalTestJSON(t, internationalPhotoFixture())),
		strings.NewReader(strings.Repeat(" ", maxInternationalDocumentBytes)))
	_, err = DecodePhotoCatalogData(t.Context(), reader)
	if err != nil {
		return
	}

	t.Fatal("expected oversized transport rejected, got success")

}

// TestInternationalPhotoCatalog uses artificial candidates, not real images.
func TestInternationalPhotoCatalog(t *testing.T) {

	photos := mustInternationalPhotos(t, internationalPhotoFixture())
	if photos.document.Faces[0].ID != "female-a" || photos.document.Faces[0].Renditions[0].Width != 128 {
		t.Fatal("expected normalized face and rendition order, got transport order")
	}
	canonical, err := photos.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical photos, got %v", err)
	}
	decoded, err := DecodePhotoCatalogData(t.Context(), bytes.NewReader(reverseInternationalObjects(t, canonical)))
	if err != nil {
		t.Fatalf("expected reordered properties accepted, got %v", err)
	}
	if !reflect.DeepEqual(decoded, photos) {
		t.Fatal("expected identical normalized photo data, got divergence")
	}

	for _, test := range []struct {
		name             string
		mutate           func(*internationalPhotos)
		selectionChanges bool
		catalogChanges   bool
	}{
		{"transport order", func(d *internationalPhotos) {
			slices.Reverse(d.Faces)
			for i := range d.Faces {
				slices.Reverse(d.Faces[i].Renditions)
			}
		}, false, false},
		{"path", func(d *internationalPhotos) { d.Faces[0].Renditions[0].Path = "elsewhere/image.webp" }, false, true},
		{"release", func(d *internationalPhotos) { d.Release = "artificial-release-2" }, false, true},
		{"definition", func(d *internationalPhotos) {
			definition := "An explanatory description"
			d.GenderTaxonomy.Categories[0].Definition = &definition
		}, false, true},
		{"integrity", func(d *internationalPhotos) {
			d.Faces[0].Renditions[0].Integrity = &internationalIntegrity{strings.Repeat("a", 64), 100}
		}, false, true},
		{"rendition", func(d *internationalPhotos) { d.Faces[0].Renditions = d.Faces[0].Renditions[:1] }, false, true},
		{"candidate", func(d *internationalPhotos) { d.Faces = d.Faces[1:] }, true, true},
		{"ID", func(d *internationalPhotos) { d.Faces[0].ID = "changed" }, true, true},
		{"gender", func(d *internationalPhotos) { d.Faces[0].Gender = "male" }, true, true},
		{"age", func(d *internationalPhotos) { d.Faces[0].RequestedAge++ }, true, true},
		{"seed", func(d *internationalPhotos) { d.Selection.Seed = strings.Repeat("2", 64) }, true, true},
		{"version", func(d *internationalPhotos) { d.Selection.Version = "v2" }, true, true},
		{"taxonomy order", func(d *internationalPhotos) { slices.Reverse(d.GenderTaxonomy.Categories) }, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := internationalPhotoFixture()
			test.mutate(&document)
			changed := mustInternationalPhotos(t, document)
			if (changed.fingerprint != photos.fingerprint) != test.selectionChanges ||
				(changed.digest != photos.digest) != test.catalogChanges {
				t.Fatal("expected specified independent photo identities, got incorrect dependency")
			}
		})
	}

	// Independently computed using Python integer JSON and KFD1 framing.
	if fmt.Sprintf("%x", photos.digest) != "3e1ad618649b79035b4d5c6110f6fabd0fe22171b861b1ce4c3839ef972f6aa7" ||
		fmt.Sprintf("%x", photos.fingerprint) != "61637475e78bd6c9d9fef39b8a997296b188f782a12d98dcdd958c2b3e029bd7" {
		t.Fatal("expected independent photo identity vectors, got different digest or fingerprint")
	}

}

// TestInternationalPhotoInvalid exercises both closed decoding and semantics.
func TestInternationalPhotoInvalid(t *testing.T) {

	for _, test := range []struct {
		name   string
		mutate func(*internationalPhotos)
	}{
		{"schema", func(d *internationalPhotos) { d.Schema = "sw-photo-catalog-v2" }},
		{"release", func(d *internationalPhotos) { d.Release = "invalid release" }},
		{"selection version", func(d *internationalPhotos) { d.Selection.Version = "-v1" }},
		{"zero seed", func(d *internationalPhotos) { d.Selection.Seed = strings.Repeat("0", 64) }},
		{"upper seed", func(d *internationalPhotos) { d.Selection.Seed = strings.Repeat("A", 64) }},
		{"short seed", func(d *internationalPhotos) { d.Selection.Seed = "1234" }},
		{"empty", func(d *internationalPhotos) { d.Faces = []internationalFace{} }},
		{"ID duplicate", func(d *internationalPhotos) { d.Faces[0].ID = d.Faces[1].ID }},
		{"ID long", func(d *internationalPhotos) { d.Faces[0].ID = strings.Repeat("a", 129) }},
		{"ID characters", func(d *internationalPhotos) { d.Faces[0].ID = "face.1" }},
		{"gender", func(d *internationalPhotos) { d.Faces[0].Gender = "undeclared" }},
		{"taxonomy empty", func(d *internationalPhotos) { d.GenderTaxonomy.Categories = []internationalCategory{} }},
		{"taxonomy duplicate", func(d *internationalPhotos) {
			d.GenderTaxonomy.Categories[1].Code = d.GenderTaxonomy.Categories[0].Code
		}},
		{"no rendition", func(d *internationalPhotos) { d.Faces[0].Renditions = []internationalRendition{} }},
		{"zero width", func(d *internationalPhotos) { d.Faces[0].Renditions[0].Width = 0 }},
		{"zero height", func(d *internationalPhotos) { d.Faces[0].Renditions[0].Height = 0 }},
		{"duplicate size", func(d *internationalPhotos) { d.Faces[0].Renditions[1] = d.Faces[0].Renditions[0] }},
		{"different dimensions at path", func(d *internationalPhotos) {
			d.Faces[0].Renditions[1].Path = d.Faces[0].Renditions[0].Path
		}},
		{"integrity hash", func(d *internationalPhotos) {
			d.Faces[0].Renditions[0].Integrity = &internationalIntegrity{"a", 1}
		}},
		{"integrity bytes", func(d *internationalPhotos) {
			d.Faces[0].Renditions[0].Integrity = &internationalIntegrity{strings.Repeat("a", 64), 0}
		}},
		{"conflicting integrity", func(d *internationalPhotos) {
			for i := range 3 {
				d.Faces[i].Renditions[0].Path = "shared.webp"
			}
			d.Faces[0].Renditions[0].Integrity = &internationalIntegrity{strings.Repeat("a", 64), 1}
			d.Faces[2].Renditions[0].Integrity = &internationalIntegrity{strings.Repeat("b", 64), 1}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := internationalPhotoFixture()
			test.mutate(&document)
			input, err := internationalCanonical(document)
			if err != nil {
				t.Fatalf("expected encodable invalid fixture, got %v", err)
			}
			_, err = DecodePhotoCatalogData(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected invalid catalog rejected, got success")
		})
	}

	for _, path := range []string{
		"/photo.webp", "photo.webp/", "a//b.webp", "./a.webp", "a/../b.webp", "a/%62.webp",
		"https://example.com/a.webp", "a.webp?q=1", "a.webp#f", `a\b.webp`, "à.webp", "a.png",
	} {
		t.Run(path, func(t *testing.T) {
			d := internationalPhotoFixture()
			d.Faces[0].Renditions[0].Path = path
			_, err := preparePhotoCatalogData(d)
			if err != nil {
				return
			}
			t.Fatal("expected invalid AssetPath rejected, got success")
		})
	}

	input, err := internationalCanonical(internationalPhotoFixture())
	if err != nil {
		t.Fatalf("expected fixture JSON, got %v", err)
	}
	for _, test := range []struct{ old, replacement string }{
		{`"schema":`, `"unknown":0,"schema":`},
		{`"schema":`, `"\u0073chema":"sw-photo-catalog-v1","schema":`},
		{`"requestedAge":40`, `"requestedAge":40,"requestedAge":40`},
		{`"requestedAge":40`, `"requestedAge":-0`},
		{`"requestedAge":40`, `"requestedAge":1e2`},
		{`"requestedAge":40`, `"requestedAge":4294967296`},
		{`"requestedAge":40`, `"requestedAge":"40"`},
		{`,"requestedAge":40`, ``},
		{`"width":256`, `"width":null`},
		{`"version":"v1"`, `"version":true`},
		{`"release":"artificial-photos"`, `"release":"\ud800"`},
		{`"release":"artificial-photos"`, "\"release\":\"\xff\""},
	} {
		altered := bytes.Replace(input, []byte(test.old), []byte(test.replacement), 1)
		if bytes.Equal(altered, input) {
			t.Fatalf("expected matched substitution %s, got unchanged JSON", test.old)
		}
		_, err = DecodePhotoCatalogData(t.Context(), bytes.NewReader(altered))
		if err != nil {
			continue
		}
		t.Fatalf("expected malformed catalog rejected for %s, got success", test.replacement)
	}

}

// TestInternationalPhotoProfile separates format validity from Italian use.
func TestInternationalPhotoProfile(t *testing.T) {

	genders := internationalPhotoFixture().GenderTaxonomy
	for _, test := range []struct {
		name   string
		mutate func(*internationalPhotos)
	}{
		{"taxonomy", func(d *internationalPhotos) { d.GenderTaxonomy.SemanticID = "other-v1" }},
		{"taxonomy order", func(d *internationalPhotos) { slices.Reverse(d.GenderTaxonomy.Categories) }},
		{"missing gender", func(d *internationalPhotos) { d.Faces = d.Faces[:2] }},
		{"missing 256", func(d *internationalPhotos) { d.Faces[0].Renditions = d.Faces[0].Renditions[1:] }},
		{"byte representation", func(d *internationalPhotos) {
			d.Faces[0].Renditions[0].Integrity = &internationalIntegrity{strings.Repeat("a", 64), math.MaxUint64}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := internationalPhotoFixture()
			test.mutate(&document)
			photos := mustInternationalPhotos(t, document)
			_, err := prepareItalianPhotoCatalog(photos, genders)
			if err != nil {
				return
			}
			t.Fatal("expected unsupported Italian photo profile rejected, got success")
		})
	}

	// U32 ages are part of the format; there is no distance cutoff.
	document := internationalPhotoFixture()
	document.Faces[0].RequestedAge = math.MaxUint32
	document.Faces[1].RequestedAge = 0
	photos := mustInternationalPhotos(t, document)
	if photos.document.Faces[0].RequestedAge != 0 {
		t.Fatal("expected zero requested age retained, got altered age")
	}

}

func internationalPhotoFixture() internationalPhotos {

	d := internationalPhotos{
		Schema: "sw-photo-catalog-v1", Release: "artificial-photos",
		GenderTaxonomy: internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale}),
		Selection:      internationalSelection{Version: "v1", Seed: strings.Repeat("1", 64)},
	}
	for _, gender := range []string{"female", "male"} {
		for _, suffix := range []string{"b", "a"} {
			face := internationalFace{ID: gender + "-" + suffix, Gender: gender, RequestedAge: 40}
			if suffix == "a" {
				face.RequestedAge = 20
			}
			for _, size := range []uint32{256, 128} {
				face.Renditions = append(face.Renditions, internationalRendition{
					Width: size, Height: size, Path: fmt.Sprintf("fixture/%d/%s.webp", size, face.ID),
				})
			}
			d.Faces = append(d.Faces, face)
		}
	}

	return d
}

func mustInternationalPhotos(t *testing.T, document internationalPhotos) *PhotoCatalogData {
	t.Helper()
	encoded, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected encodable photo fixture, got %v", err)
	}
	data, err := DecodePhotoCatalogData(t.Context(), bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("expected valid photo fixture, got %v", err)
	}
	return data
}
