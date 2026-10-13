// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestInternationalBaseEquivalence reconstructs every original CSV byte from
// the decoded v1 document, including every field, row and meaningful ordering.
func TestInternationalBaseEquivalence(t *testing.T) {

	base, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
	if err != nil {
		t.Fatalf("expected original Italian CSV conversion, got %v", err)
	}
	canonical, err := base.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected C1 base JSON, got %v", err)
	}
	decoded, err := DecodeItalianBaseData(t.Context(), bytes.NewReader(canonical))
	if err != nil {
		t.Fatalf("expected typed base decoding, got %v", err)
	}
	if !reflect.DeepEqual(base.document, decoded.document) {
		t.Fatal("expected complete typed base round trip, got divergence")
	}

	d := &decoded.document
	reconstructed := map[string][][]string{
		"age_weights.csv":  {{"age", "weight"}},
		"first_names.csv":  {{"gender", "cohort", "name", "ascii_atom", "weight"}},
		"last_names.csv":   {{"name", "ascii_atom", "weight"}},
		"places.csv":       {{"istat_code", "city", "state_prov", "region", "population"}},
		"postal_codes.csv": {{"istat_code", "postal_codes"}},
		"street_names.csv": {{"name", "weight"}},
	}
	for _, row := range d.Ages {
		reconstructed["age_weights.csv"] = append(reconstructed["age_weights.csv"], []string{
			strconv.FormatUint(uint64(row.Age), 10), strconv.FormatUint(row.Weight, 10),
		})
	}
	for _, group := range d.FirstNames {
		for _, row := range group.Entries {
			reconstructed["first_names.csv"] = append(reconstructed["first_names.csv"], []string{
				group.Gender, group.Cohort, row.Display, *row.ASCIIAtom, strconv.FormatUint(row.Weight, 10),
			})
		}
	}
	for _, row := range d.LastNames {
		reconstructed["last_names.csv"] = append(reconstructed["last_names.csv"], []string{
			row.Display, *row.ASCIIAtom, strconv.FormatUint(row.Weight, 10),
		})
	}
	for _, row := range d.Geography.Places {
		reconstructed["places.csv"] = append(reconstructed["places.csv"], []string{
			row.ID, row.Name, *row.Administrative[0].Code, row.Territory, strconv.FormatUint(row.Population, 10),
		})
		reconstructed["postal_codes.csv"] = append(reconstructed["postal_codes.csv"], []string{
			row.ID, strings.Join(d.Geography.PostalCodes[row.ID], ";"),
		})
	}
	for _, row := range d.StreetNames.Entries {
		reconstructed["street_names.csv"] = append(reconstructed["street_names.csv"], []string{
			row.Text, strconv.FormatUint(row.Weight, 10),
		})
	}

	for _, name := range []string{
		"age_weights.csv", "first_names.csv", "last_names.csv", "places.csv", "postal_codes.csv", "street_names.csv",
	} {
		original, err := os.ReadFile("data/it-2026a/" + name)
		if err != nil {
			t.Fatalf("expected original %s, got %v", name, err)
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		err = writer.WriteAll(reconstructed[name])
		if err != nil {
			t.Fatalf("expected reconstructed CSV, got %v", err)
		}
		if !bytes.Equal(original, buffer.Bytes()) {
			t.Fatalf("expected every original byte of %s, got different reconstructed CSV", name)
		}
		t.Logf("%s: all %d rows and all fields equal; SHA-256 %x",
			name, len(reconstructed[name])-1, sha256.Sum256(original))
	}

	second, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
	if err != nil {
		t.Fatalf("expected repeated conversion, got %v", err)
	}
	secondBytes, err := second.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected repeated C1 bytes, got %v", err)
	}
	if !bytes.Equal(canonical, secondBytes) {
		t.Fatal("expected reproducible base bytes, got divergence")
	}
	want := "37f1c42a3c0c85b84c379ea552888306fe2e4217958d0ae47868ec8fa18791e4"
	if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != want {
		t.Fatalf("expected converted base SHA-256 %s, got %s", want, got)
	}

}

// TestInternationalDocumentIdentities tests transport invariance, closed
// projections and isolation using full base data and an artificial L2 model.
func TestInternationalDocumentIdentities(t *testing.T) {

	base, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
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

// TestInternationalSocioeconomicEquivalence uses the real local production
// asset. Set SYNTHETIC_WORLD_ITALIAN_MODEL to require a specific external file;
// otherwise a checkout without the local asset explicitly skips this test.
func TestInternationalSocioeconomicEquivalence(t *testing.T) {

	path := os.Getenv("SYNTHETIC_WORLD_ITALIAN_MODEL")
	isRequired := path != ""
	if !isRequired {
		path = "../../D2B2-logical-assets-v0.json"
	}
	originalBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !isRequired {
			t.Skip("real Italian asset unavailable; set SYNTHETIC_WORLD_ITALIAN_MODEL to require it")
		}
		t.Fatalf("expected real canonical Italian model, got %v", err)
	}

	original, err := decodeSocioeconomicModelJSON(bytes.NewReader(originalBytes))
	if err != nil {
		t.Fatalf("expected original typed model, got %v", err)
	}
	legacy, err := decodeSocioeconomicModel(bytes.NewReader(originalBytes))
	if err != nil {
		t.Fatalf("expected original compiler path, got %v", err)
	}

	converted, err := ConvertItalianSocioeconomicData(t.Context(), bytes.NewReader(originalBytes))
	if err != nil {
		t.Fatalf("expected real socioeconomic conversion, got %v", err)
	}
	canonical, err := converted.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected international C1 bytes, got %v", err)
	}
	decoded, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(canonical))
	if err != nil {
		t.Fatalf("expected typed international decoder, got %v", err)
	}

	projection, err := projectItalianSocioeconomic(&decoded.document)
	if err != nil {
		t.Fatalf("expected typed projection, got %v", err)
	}
	if !reflect.DeepEqual(original, projection) {
		t.Fatal("expected equality of every original typed property, got projection divergence")
	}
	if !reflect.DeepEqual(legacy, decoded.model) {
		t.Fatal("expected identical complete compiled models, got divergence")
	}
	reconstructed, err := decoded.model.canonicalModel()
	if err != nil {
		t.Fatalf("expected canonical reconstruction, got %v", err)
	}
	digest := fmt.Sprintf("%x", decoded.model.digest)
	if !bytes.Equal(originalBytes, reconstructed) || digest != italianSourceModelSHA256 {
		t.Fatal("expected identical frozen canonical bytes and logical digest, got divergence")
	}
	for i, path := range socioeconomicPaths {
		if decoded.model.fingerprints[i] != legacy.fingerprints[i] {
			t.Fatalf("expected unchanged fingerprint at %s, got divergence", path)
		}
		t.Logf("%s: %x", path, decoded.model.fingerprints[i])
	}

	var weights, references, nulls, laws, points int
	for _, categorical := range decoded.document.Models.categorical() {
		weights += len(categorical.Weights)
	}
	for _, income := range decoded.document.Models.Income.ordered() {
		references += len(income.TupleMap)
		laws += len(income.Laws)
		for _, law := range income.Laws {
			points += len(law.Values)
		}
		for _, reference := range income.TupleMap {
			if reference == nil {
				nulls++
			}
		}
	}
	if weights != 12194 || references != 76160 || nulls != 11200 || laws != 60 || points != 23100 {
		t.Fatalf("expected 12194/76160/11200/60/23100, got %d/%d/%d/%d/%d", weights, references, nulls, laws, points)
	}
	for _, law := range projection.employeeCount {
		if len(law.values) != 128 {
			t.Fatalf("expected 128 EmployeeCount points, got %d", len(law.values))
		}
	}
	for i, age := range projection.ages {
		if len(age.compiled) != 55 || !slices.Equal(age.compiled, original.ages[i].compiled) {
			t.Fatalf("expected all 55 compiled positions in partition %d, got divergence", i)
		}
	}
	t.Logf("all properties equal: %d weights, 14 count laws, 140+16 factors, %d references, "+
		"%d nulls, %d Income laws, %d points, 440 age positions", weights, references, nulls, laws, points)
	t.Logf("frozen logical digest: %x", decoded.model.digest)

	second, err := ConvertItalianSocioeconomicData(t.Context(), bytes.NewReader(originalBytes))
	if err != nil {
		t.Fatalf("expected repeated conversion, got %v", err)
	}
	secondBytes, err := second.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected repeated C1 bytes, got %v", err)
	}
	if !bytes.Equal(canonical, secondBytes) {
		t.Fatal("expected reproducible socioeconomic bytes, got divergence")
	}
	want := "9e203ceef78aa94f403dababc51505c87faeea0a93843f3a7f89d0df22291ea0"
	if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != want {
		t.Fatalf("expected qualified socioeconomic byte hash, got %s", got)
	}

	logical, err := decoded.LogicalDigest()
	if err != nil {
		t.Fatalf("expected international socioeconomic digest, got %v", err)
	}
	want = "ad59a4c3f302b7186113e38ce8a498c79d7a14102675eb23818fafd0995f64d6"
	if got := fmt.Sprintf("%x", logical); got != want {
		t.Fatalf("expected qualified international digest %s, got %s", want, got)
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
