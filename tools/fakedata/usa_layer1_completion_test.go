// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/json"
)

const usaCompletionDirectory = "../../synthetic-world-usa-l1-us-2026a-BASE-v2-CANDIDATE/"

// BenchmarkUSALayer1Completion reports full admission with prepared evidence
// separately from random-access generation; no public World is constructed.
func BenchmarkUSALayer1Completion(b *testing.B) {

	evidence := readUSACompletionEvidence(b)
	encoded := readUSACompletionFile(b, "assets/base.json")
	base, err := decodeUSABaseCandidate(b.Context(), bytes.NewReader(encoded), evidence)
	if err != nil {
		b.Fatalf("expected admitted USA base, got %v", err)
	}
	phones, err := decodeUSANPACandidate(b.Context(), bytes.NewReader(readUSACompletionFile(b, "assets/npas.json")))
	if err != nil {
		b.Fatalf("expected admitted NPA rule, got %v", err)
	}
	w, err := newUSALayer1Harness(IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026, base, phones)
	if err != nil {
		b.Fatalf("expected internal harness, got %v", err)
	}
	b.Run("decode-and-prepare-base", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := decodeUSABaseCandidate(b.Context(), bytes.NewReader(encoded), evidence)
			if err != nil {
				b.Fatalf("expected prepared candidate, got %v", err)
			}
		}
	})
	b.Run("person-random-access", func(b *testing.B) {
		b.ReportAllocs()
		index := PersonIndex(1)
		for b.Loop() {
			_, err := w.person(index)
			if err != nil {
				b.Fatalf("expected USA person, got %v", err)
			}
			index = index%Layer1MaxPersonIndex + 1
		}
	})
	b.Run("contacts-long-prefixes", func(b *testing.B) {
		first, last := strings.Repeat("a", 80), strings.Repeat("b", 80)
		b.ReportAllocs()
		for b.Loop() {
			_, _, err := w.contacts.compose(42, first, last)
			if err != nil {
				b.Fatalf("expected bounded contacts, got %v", err)
			}
		}
	})
	b.Run("phone", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := w.phones.phone(42)
			if err != nil {
				b.Fatalf("expected fiction number, got %v", err)
			}
		}
	})

}

// TestUSABaseCompletionAdmission checks C1 identity, F.4 and owned values.
func TestUSABaseCompletionAdmission(t *testing.T) {

	evidence := readUSACompletionEvidence(t)
	encoded := readUSACompletionFile(t, "assets/base.json")
	var document usaBaseDocument
	err := decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), &document)
	if err != nil {
		t.Fatalf("expected closed national base, got %v", err)
	}
	base, err := prepareUSABaseCandidate(&document, evidence)
	if err != nil {
		t.Fatalf("expected candidate preparation, got %v", err)
	}
	canonical, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected C1 serialization, got %v", err)
	}
	if !bytes.Equal(canonical, encoded) || len(encoded) != 3583463 ||
		fmt.Sprintf("%x", sha256.Sum256(encoded)) != "d8fd7c76142a86917ed4acec065951f8c5e9b44865e1bf6eed7f09f847eb5b80" {
		t.Fatal("expected identified canonical base, got different bytes")
	}
	if len(base.lastNames.names.entries) != 50000 || len(base.firstNames.groups) != 10 ||
		len(base.ages) != 55 || len(base.streets) != 256 || len(base.geography.locations) != 127 {
		t.Fatal("expected complete national cardinalities, got incomplete USA base")
	}
	var common internationalBase
	err = json.Unmarshal(encoded, &common)
	if err != nil {
		t.Fatalf("expected shared schema values, got %v", err)
	}
	expected, err := internationalBaseFingerprints(&common)
	if err != nil {
		t.Fatalf("expected shared F.4 fingerprints, got %v", err)
	}
	if expected != base.fingerprints {
		t.Fatal("expected international F.4 identity, got national alternate projection")
	}
	for i, digest := range base.fingerprints {
		t.Logf("F[%d]=%x", i, digest)
	}
	t.Logf("D_base=%x bytes=%d", base.digest, len(encoded))

	// Prepared samplers retain no mutable alias to the decoded base.
	age, street := base.ages[0], base.streets[0]
	document.Ages[0].Weight++
	document.StreetNames.Entries[0].Text = "Changed Street"
	*document.LastNames[0].ASCIIAtom = "changed"
	document.Geography.PostalCodes[document.Geography.Places[0].ID][0] = "99999"
	if base.ages[0] != age || base.streets[0] != street || base.lastNames.names.entries[0].asciiAtom == "changed" ||
		base.geography.locations[0].zips[0] == "99999" {
		t.Fatal("expected owned admitted values, got mutable base alias")
	}

}

// TestUSABaseCompletionClosedAndBounded rejects malformed, excessive and
// canceled input through the real decoder.
func TestUSABaseCompletionClosedAndBounded(t *testing.T) {

	evidence := readUSACompletionEvidence(t)
	encoded := string(readUSACompletionFile(t, "assets/base.json"))
	cases := map[string]string{
		"unknown":   "{\"unknown\":1," + encoded[1:],
		"duplicate": "{\"country\":\"US\"," + encoded[1:],
		"null":      strings.Replace(encoded, "\"lastNames\":[", "\"lastNames\":null,\"discarded\":[", 1),
		"missing":   strings.Replace(encoded, "\"country\":\"US\",", "", 1),
		"fraction":  strings.Replace(encoded, "\"age\":21", "\"age\":21.0", 1),
		"trailing":  encoded + "{}",
		"size":      strings.Repeat(" ", maxInternationalDocumentBytes) + encoded,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := decodeUSABaseCandidate(t.Context(), strings.NewReader(input), evidence)
			if err != nil {
				return
			}
			t.Fatal("expected rejected malformed or oversized USA base, got nil")
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := decodeUSABaseCandidate(ctx, strings.NewReader(encoded), evidence)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
		return
	}

	t.Fatal("expected canceled decoder, got nil")

}

// TestUSABaseCompletionInvalid rejects changes to adopted national values.
func TestUSABaseCompletionInvalid(t *testing.T) {

	evidence := readUSACompletionEvidence(t)
	encoded := readUSACompletionFile(t, "assets/base.json")
	cases := []struct {
		name   string
		change func(*usaBaseDocument)
	}{
		{"schema", func(d *usaBaseDocument) { d.Schema = "sw-us-base-v1" }},
		{"release", func(d *usaBaseDocument) { d.Release = "us-2026a-release" }},
		{"profile", func(d *usaBaseDocument) { d.Profile = "it-2026a-base-v1" }},
		{"country", func(d *usaBaseDocument) { d.Country = "IT" }},
		{"anchor", func(d *usaBaseDocument) { d.AnchorDate = "2026-01-02" }},
		{"gender", func(d *usaBaseDocument) { d.Genders.Categories[0].Code = "other" }},
		{"age-mass", func(d *usaBaseDocument) { d.Ages[0].Weight++ }},
		{"age-count", func(d *usaBaseDocument) { d.Ages = append(d.Ages, d.Ages[0]) }},
		{"cohort", func(d *usaBaseDocument) { d.Cohorts[0].FromYear++ }},
		{"name-display", func(d *usaBaseDocument) { d.FirstNames[0].Entries[0].Display = "Invented" }},
		{"name-atom", func(d *usaBaseDocument) { d.FirstNames[0].Entries[0].ASCIIAtom = nil }},
		{"name-weight", func(d *usaBaseDocument) { d.FirstNames[0].Entries[0].Weight = math.MaxUint64 }},
		{"name-cohort", func(d *usaBaseDocument) { d.FirstNames[0].Cohort = "2000-2005" }},
		{"name-bound", func(d *usaBaseDocument) {
			d.FirstNames[0].Entries = append(d.FirstNames[0].Entries, d.FirstNames[0].Entries[0])
		}},
		{"surname-missing", func(d *usaBaseDocument) { d.LastNames = d.LastNames[:49999] }},
		{"surname-bound", func(d *usaBaseDocument) { d.LastNames = append(d.LastNames, d.LastNames[0]) }},
		{"surname-weight", func(d *usaBaseDocument) { d.LastNames[0].Weight++ }},
		{"place-system", func(d *usaBaseDocument) { d.Geography.PlaceCodeSystem = "it-istat" }},
		{"postal-system", func(d *usaBaseDocument) { d.Geography.PostalCodeSystem = "usps-2026" }},
		{"territory-system", func(d *usaBaseDocument) { d.Geography.TerritorySystem = "states" }},
		{"place-count", func(d *usaBaseDocument) { d.Geography.Places = append(d.Geography.Places, d.Geography.Places[0]) }},
		{"place-mass", func(d *usaBaseDocument) { d.Geography.Places[0].Population++ }},
		{"place-name", func(d *usaBaseDocument) { d.Geography.Places[0].Name = "Rome" }},
		{"place-division", func(d *usaBaseDocument) { d.Geography.Places[0].Territory = "pacific" }},
		{"place-state", func(d *usaBaseDocument) { *d.Geography.Places[0].Administrative[0].Code = "PR" }},
		{"postal-mismatch", func(d *usaBaseDocument) { d.Geography.PostalCodes[d.Geography.Places[0].ID][0] = "99999" }},
		{"postal-extra", func(d *usaBaseDocument) { d.Geography.PostalCodes["unknown"] = []string{"12345"} }},
		{"street-kind", func(d *usaBaseDocument) { d.StreetNames.Kind = "fragment" }},
		{"street-count", func(d *usaBaseDocument) { d.StreetNames.Entries = d.StreetNames.Entries[:255] }},
		{"street-unit-weight", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Weight = 2 }},
		{"street-case-duplicate", func(d *usaBaseDocument) {
			d.StreetNames.Entries[1].Text = strings.ToUpper(d.StreetNames.Entries[0].Text)
		}},
		{"street-suffix", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Text = "Acacia" }},
		{"street-repeated-suffix", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Text = "Acacia Lane Street" }},
		{"street-italian", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Text = "Via Roma" }},
		{"street-content", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Text = "Acacia Court" }},
		{"street-long", func(d *usaBaseDocument) { d.StreetNames.Entries[0].Text = strings.Repeat("a", 81) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var document usaBaseDocument
			err := json.Unmarshal(encoded, &document)
			if err != nil {
				t.Fatalf("expected fresh base fixture, got %v", err)
			}
			test.change(&document)
			input, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("expected mutation transport, got %v", err)
			}
			_, err = decodeUSABaseCandidate(t.Context(), bytes.NewReader(input), evidence)
			if err != nil {
				return
			}
			t.Fatal("expected rejected USA base mutation, got nil")
		})
	}
	var document usaBaseDocument
	err := json.Unmarshal(encoded, &document)
	if err != nil {
		t.Fatalf("expected base fixture, got %v", err)
	}
	for _, invalid := range []usaBaseEvidence{
		{}, {ages: evidence.ages, firstNames: evidence.firstNames, lastNames: evidence.lastNames},
		{
			ages: evidence.ages, firstNames: evidence.firstNames, lastNames: evidence.lastNames,
			geography: &usaGeographyCandidate{},
		},
	} {
		_, err := prepareUSABaseCandidate(&document, invalid)
		if err != nil {
			continue
		}
		t.Fatal("expected rejected incomplete or wrong evidence, got nil")
	}

}

// TestUSACompletionIdentityIsolation separates canonical IDs, national paths
// and each dataset dependency from observational contact parameters.
func TestUSACompletionIdentityIsolation(t *testing.T) {

	w := readUSACompletionHarness(t)
	other, err := newUSALayer1Harness(IdentityNamespace{Name: "other-namespace", Generation: 2}, 2026, w.base, w.phones)
	if err != nil {
		t.Fatalf("expected second identity namespace, got %v", err)
	}
	for _, index := range []PersonIndex{1, 42, 65536, Layer1MaxPersonIndex} {
		first, err := w.person(index)
		if err != nil {
			t.Fatalf("expected first USA identity, got %v", err)
		}
		second, err := other.person(index)
		if err != nil {
			t.Fatalf("expected second USA identity, got %v", err)
		}
		if first.person.ID == second.person.ID || strings.Contains(first.person.Email, first.person.ID) ||
			strings.Contains(first.person.SiteURL, first.person.ID) {
			t.Fatal("expected identity namespace separation from contacts, got canonical ID leakage")
		}
		codecToken := first.person.ID[strings.LastIndexByte(first.person.ID, '_')+1:]
		if strings.Contains(first.person.Email, codecToken) || strings.Contains(first.person.SiteURL, codecToken) {
			t.Fatal("expected absence of the Person codec token, got canonical token leakage")
		}
		first.person.ID, second.person.ID = "", ""
		if first != second {
			t.Fatal("expected namespace-independent observational values, got canonical identity dependency")
		}
	}

	// Dependency changes affect only the corresponding stream family.
	for family := range w.base.dependencies {
		changedBase := *w.base
		changedBase.fingerprints[family][0] ^= 1
		changedBase.dependencies[family], err = BytesComponent(w.base.dependencies[family].name,
			changedBase.fingerprints[family][:])
		if err != nil {
			t.Fatalf("expected alternate dependency, got %v", err)
		}
		changed := *w
		changed.base = &changedBase
		for i, path := range []string{
			streamIdentityBirthAge, streamIdentityNameFirst, streamIdentityNameLast,
			streamGeographyPlacePopulation, streamGeographyPlacePostal, streamGeographyAddressFragment,
		} {
			before, err := w.stream(42, path, w.base.dependencies[i])
			if err != nil {
				t.Fatalf("expected base stream, got %v", err)
			}
			after, err := changed.stream(42, path, changed.base.dependencies[i])
			if err != nil {
				t.Fatalf("expected alternate stream, got %v", err)
			}
			if (before.state != after.state) != (i == family) {
				t.Fatalf("expected only family %d to change, got change in %d", family, i)
			}
		}
		contacts, err := newUSAContacts(changed.streams)
		if err != nil {
			t.Fatalf("expected unchanged contacts, got %v", err)
		}
		if *contacts != *w.contacts {
			t.Fatal("expected contacts independent of all dataset fingerprints, got dependency leakage")
		}
	}

	// The same procedure under Italian paths retains its original result.
	italianEmail, err := deriveAffinePermutation(w.streams, streamIdentityEmailToken)
	if err != nil {
		t.Fatalf("expected Italian email parameters, got %v", err)
	}
	italianSite, err := deriveAffinePermutation(w.streams, streamIdentitySiteToken)
	if err != nil {
		t.Fatalf("expected Italian site parameters, got %v", err)
	}
	if italianEmail == w.contacts.emailToken || italianSite == w.contacts.siteToken {
		t.Fatal("expected national path separation for this seed, got identical IT and US parameters")
	}
	for range 10 {
		_, err := newUSAContacts(w.streams)
		if err != nil {
			t.Fatalf("expected stateless USA derivation, got %v", err)
		}
	}
	replayedEmail, err := deriveAffinePermutation(w.streams, streamIdentityEmailToken)
	if err != nil {
		t.Fatalf("expected unchanged Italian derivation, got %v", err)
	}
	replayedSite, err := deriveAffinePermutation(w.streams, streamIdentitySiteToken)
	if err != nil {
		t.Fatalf("expected unchanged Italian derivation, got %v", err)
	}
	if replayedEmail != italianEmail || replayedSite != italianSite {
		t.Fatal("expected original Italian affine state, got USA interference")
	}

}

// TestUSACompletionInvalidCoordinates rejects invalid coordinates and setup
// before composing a person, address, phone or observational contacts.
func TestUSACompletionInvalidCoordinates(t *testing.T) {

	w := readUSACompletionHarness(t)
	for _, index := range []PersonIndex{0, Layer1MaxPersonIndex + 1, PersonIndex(math.MaxUint64)} {
		var expected error = ErrLayer1IndexOutOfRange
		if index == 0 {
			expected = ErrInvalidIndex
		}
		_, personErr := w.person(index)
		_, phoneErr := w.phones.phone(index)
		_, _, contactErr := w.contacts.compose(index, "mary", "smith")
		_, _, addressErr := w.address(index)
		for _, err := range []error{personErr, phoneErr, contactErr, addressErr} {
			if err != nil {
				if !errors.Is(err, expected) {
					t.Fatalf("expected domain error %v, got %v", expected, err)
				}
				continue
			}
			t.Fatalf("expected rejected coordinate %d, got success", index)
		}
	}
	for _, candidate := range []*usaContacts{
		nil, {}, {emailToken: affinePermutation{a: 2}, siteToken: affinePermutation{a: 1}},
	} {
		_, _, err := candidate.compose(1, "mary", "smith")
		if err != nil {
			continue
		}
		t.Fatal("expected rejected uninitialized contacts, got nil")
	}
	for _, candidate := range []*usaLayer1Harness{nil, {}} {
		_, err := candidate.person(1)
		if err != nil {
			continue
		}
		t.Fatal("expected rejected uninitialized harness, got nil")
	}
	_, err := newUSALayer1Harness(IdentityNamespace{}, 2026, w.base, w.phones)
	if err != nil {
		if !errors.Is(err, ErrInvalidIdentityNamespace) {
			t.Fatalf("expected invalid namespace identity, got %v", err)
		}
		return
	}

	t.Fatal("expected rejected invalid namespace, got nil")

}

// TestUSACompletionPostalBoundsLeaveItalyUnchanged checks local USA ZIP
// bounds and the unchanged Italian postal and nominal limits.
func TestUSACompletionPostalBoundsLeaveItalyUnchanged(t *testing.T) {

	for _, count := range []int{8, 9, 1024, 1025} {
		document := internationalGeography{
			PlaceCodeSystem: "test", TerritorySystem: "test", PostalCodeSystem: "test",
			Places: []internationalPlace{}, PostalCodes: map[string][]string{"test": make([]string, count)},
		}
		for i := range document.PostalCodes["test"] {
			document.PostalCodes["test"][i] = "12345"
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("expected postal fixture, got %v", err)
		}
		var italian internationalGeography
		err = decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), &italian)
		if err != nil {
			if count <= 8 {
				t.Fatalf("expected Italian bound eight, got %v", err)
			}
		}
		if err == nil && count > 8 {
			t.Fatalf("expected Italian rejection above eight, got %d accepted ZIPs", count)
		}
		var usa usaBaseGeography
		err = decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), &usa)
		if err != nil {
			if count <= 1024 {
				t.Fatalf("expected USA bound 1024, got %v", err)
			}
		}
		if err == nil && count > 1024 {
			t.Fatal("expected USA rejection above 1024, got accepted postal list")
		}
	}
	if reflect.TypeFor[internationalBase]().Field(9).Tag.Get("max") != "600" ||
		reflect.TypeFor[internationalFirstNames]().Field(2).Tag.Get("max") != "80" {
		t.Fatal("expected unchanged Italian nominal decoder limits, got widening")
	}

}

// TestUSAContactsCompletion verifies the original derivations, syntax,
// complete tokens and uniqueness with truncated nominal prefixes.
func TestUSAContactsCompletion(t *testing.T) {

	w := readUSACompletionHarness(t)
	email, err := deriveAffinePermutation(w.streams, "us/v1/identity/email/token")
	if err != nil {
		t.Fatalf("expected original email derivation procedure, got %v", err)
	}
	site, err := deriveAffinePermutation(w.streams, "us/v1/identity/site/token")
	if err != nil {
		t.Fatalf("expected original site derivation procedure, got %v", err)
	}
	if email != w.contacts.emailToken || site != w.contacts.siteToken {
		t.Fatal("expected exact path-prefixed affine derivations, got different parameters")
	}
	t.Logf("email a=%d b=%d; site a=%d b=%d", email.a, email.b, site.a, site.b)
	for _, atoms := range [][2]string{
		{"mary", "smith"}, {strings.Repeat("a", 57), strings.Repeat("b", 80)},
		{strings.Repeat("a", 56) + "-z", strings.Repeat("b", 80)},
		{strings.Repeat("a", 80), strings.Repeat("b", 80)},
	} {
		seenEmail, seenSite := map[string]bool{}, map[string]bool{}
		for i := PersonIndex(1); i <= 10000; i++ {

			e, s, err := w.contacts.compose(i, atoms[0], atoms[1])
			if err != nil {
				t.Fatalf("expected reserved contacts, got %v", err)
			}
			local, domain, _ := strings.Cut(e, "@")
			if len(local) > 64 || domain != "people.example.test" || strings.Contains(local, "..") ||
				!strings.HasSuffix(local, "."+encodeBase36Fixed5(email.value(i))) {
				t.Fatalf("expected bounded complete email token, got %s", e)
			}
			_, err = mail.ParseAddress(e)
			if err != nil {
				t.Fatalf("expected syntactic email, got %v", err)
			}
			u, err := url.Parse(s)
			if err != nil {
				t.Fatalf("expected syntactic site URL, got %v", err)
			}
			label, rest, _ := strings.Cut(u.Hostname(), ".")
			if u.Scheme != "https" || rest != "profiles.example.test" || len(label) > 63 ||
				!strings.HasSuffix(label, "-"+encodeBase36Fixed5(site.value(i))) || seenEmail[e] || seenSite[s] {
				t.Fatalf("expected independent site token and unique contacts, got %s/%s", e, s)
			}
			seenEmail[e], seenSite[s] = true, true

		}
	}
	for _, pair := range [][2]string{{"", "smith"}, {"a.b", "smith"}, {"MARY", "smith"}, {"mary", "x@y"},
		{strings.Repeat("a", 81), "smith"}, {"-mary", "smith"}, {"mary", "smith--jones"}} {
		_, _, err := w.contacts.compose(1, pair[0], pair[1])
		if err != nil {
			continue
		}
		t.Fatalf("expected rejected contact atoms, got %v", pair)
	}
	leading := &usaContacts{emailToken: affinePermutation{a: 1}, siteToken: affinePermutation{a: 5, b: 1}}
	e, s, err := leading.compose(1, "mary", "smith")
	if err != nil {
		t.Fatalf("expected leading-zero contacts, got %v", err)
	}
	if e != "mary.smith.00000@people.example.test" || s != "https://mary-smith-00001.profiles.example.test" {
		t.Fatalf("expected complete zero-padded tokens, got %s/%s", e, s)
	}

}

// TestUSAContactsCompletionFullDomain exhausts both integer permutations with
// two bitsets (15,116,544 bytes), not sixty million allocated contact strings.
func TestUSAContactsCompletionFullDomain(t *testing.T) {

	w := readUSACompletionHarness(t)
	for _, permutation := range []affinePermutation{w.contacts.emailToken, w.contacts.siteToken} {
		if greatestCommonDivisor(permutation.a, emailAndSiteTokenModulus) != 1 {
			t.Fatal("expected coprime affine multiplier, got non-bijection")
		}
		seen := make([]byte, (emailAndSiteTokenModulus+7)/8)
		for i := PersonIndex(1); i <= Layer1MaxPersonIndex; i++ {
			value := permutation.value(i)
			bit := byte(1 << (value % 8))
			if seen[value/8]&bit != 0 {
				t.Fatalf("expected unique full-domain token, got collision at %d", i)
			}
			seen[value/8] |= bit
		}
	}
	t.Logf("exhaustive checks=%d per permutation; bitset bytes=%d per permutation", Layer1MaxPersonIndex,
		(emailAndSiteTokenModulus+7)/8)

}

// TestUSALayer1CompletionHarness replays 20,000 real-asset coordinates in
// reverse order and verifies geographic, nominal and contact coherence.
func TestUSALayer1CompletionHarness(t *testing.T) {

	const count = 20000
	w := readUSACompletionHarness(t)
	replay, err := newUSALayer1Harness(w.namespace, 2026, w.base, w.phones)
	if err != nil {
		t.Fatalf("expected independent B harness, got %v", err)
	}
	expected := make([][sha256.Size]byte, count)
	ages, genders, cohorts := map[int]int{}, map[Gender]int{}, map[string]int{}
	states, divisions, localities, streets := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	emails, sites := map[string]bool{}, map[string]bool{}
	nominal := map[firstNameGroup]map[string]bool{}
	cohortIDs := []string{"1950-1964", "1965-1979", "1980-1989", "1990-1999", "2000-2005"}
	for i, group := range w.base.firstNames.groups {
		gender := GenderFemale
		if i >= 5 {
			gender = GenderMale
		}
		key := firstNameGroup{gender: gender, cohort: cohortIDs[i%5]}
		nominal[key] = map[string]bool{}
		for _, name := range group.entries {
			nominal[key][name.display] = true
		}
	}
	surnames := map[string]bool{}
	for _, name := range w.base.lastNames.names.entries {
		surnames[name.display] = true
	}
	rolling := sha256.New()
	samples := []any{}
	for i := PersonIndex(1); i <= count; i++ {

		p, err := w.person(i)
		if err != nil {
			t.Fatalf("expected USA person at %d, got %v", i, err)
		}
		birth, err := time.Parse(time.DateOnly, p.person.BirthDate)
		if err != nil {
			t.Fatalf("expected ISO birth date, got %v", err)
		}
		cohort, err := cohortForYear(birth.Year())
		if err != nil {
			t.Fatalf("expected qualified birth cohort, got %v", err)
		}
		if p.anchorAge < 21 || p.anchorAge > 75 || p.person.CurrentAge != p.anchorAge || cohort != p.cohort ||
			!nominal[firstNameGroup{gender: p.person.Gender, cohort: p.cohort}][p.person.FirstName] ||
			!surnames[p.person.LastName] ||
			p.person.FirstName == strings.ToUpper(p.person.FirstName) ||
			p.person.LastName == strings.ToUpper(p.person.LastName) {
			t.Fatalf("expected adopted natural names and compatible age/cohort, got %+v", p)
		}
		// Replay one WeightedIndex and its final RNG state; state supplies Division.
		rng, err := w.stream(i, streamGeographyPlacePopulation, w.base.dependencies[3])
		if err != nil {
			t.Fatalf("expected locality stream, got %v", err)
		}
		manual := rng
		position, err := manual.WeightedIndex(w.base.geography.weights)
		if err != nil {
			t.Fatalf("expected single proportional draw, got %v", err)
		}
		place, err := w.base.geography.selectLocation(&rng)
		if err != nil {
			t.Fatalf("expected replayed locality, got %v", err)
		}
		admitted := w.base.geography.locations[position]
		address := p.person.Address
		if rng.state != manual.state || place.id != admitted.id || p.localityID != admitted.id ||
			address.Municipality != admitted.name || address.StateProv != admitted.state.USPS ||
			p.division != admitted.state.Division || p.divisionOrdinal != admitted.state.DivisionOrdinal ||
			!slices.Contains(admitted.zips, address.PostalCode) || address.Country != "US" {
			t.Fatalf("expected single coherent locality/ZIP/state/Division at %d, got %+v", i, p)
		}
		civicText, street, _ := strings.Cut(address.Street, " ")
		civic, err := strconv.Atoi(civicText)
		if err != nil {
			t.Fatalf("expected numeric civic, got %v", err)
		}
		if civic < 1 || civic > 9999 || !slices.Contains(w.base.streets, street) ||
			emails[p.person.Email] || sites[p.person.SiteURL] {
			t.Fatalf("expected bounded full-name street and unique contacts, got %+v", p)
		}
		line := fmt.Sprintf("%s, %s, %s %s", address.Street, address.Municipality, address.StateProv, address.PostalCode)
		if p.addressLine != line {
			t.Fatalf("expected national address line %s, got %s", line, p.addressLine)
		}
		if strings.Contains(p.person.Email, p.person.ID) || strings.Contains(p.person.SiteURL, p.person.ID) ||
			p.person.PhotoID != "" {
			t.Fatal("expected observational contacts and separately qualified photography, got identity leakage")
		}
		emails[p.person.Email], sites[p.person.SiteURL] = true, true
		ages[p.anchorAge]++
		genders[p.person.Gender]++
		cohorts[p.cohort]++
		states[address.StateProv]++
		divisions[p.division]++
		localities[p.localityID]++
		streets[street]++
		encoded := usaCompletionRecord(t, p)
		expected[i-1] = sha256.Sum256(encoded)
		_, err = rolling.Write(encoded)
		if err != nil {
			t.Fatalf("expected hashed harness record, got %v", err)
		}
		if i <= 8 {
			samples = append(samples, []any{
				p.person, p.anchorAge, p.cohort, p.localityID, p.division, p.divisionOrdinal, p.addressLine,
			})
		}

	}
	if len(ages) != 55 || len(genders) != 2 || len(cohorts) != 5 || len(states) != 51 ||
		len(divisions) != 9 || len(streets) != 256 {
		t.Fatalf("expected all national age/gender/cohort/state/Division/street supports observed, got %d/%d/%d/%d/%d/%d",
			len(ages), len(genders), len(cohorts), len(states), len(divisions), len(streets))
	}
	for i := PersonIndex(count); i > 0; i-- {
		p, err := replay.person(i)
		if err != nil {
			t.Fatalf("expected reverse-order B generation, got %v", err)
		}
		if sha256.Sum256(usaCompletionRecord(t, p)) != expected[i-1] {
			t.Fatalf("expected deterministic A/B random access at %d, got different person", i)
		}
	}
	coordinates := splitMix64{state: 909}
	for range 1000 {
		value, err := coordinates.Uint64n(uint64(Layer1MaxPersonIndex))
		if err != nil {
			t.Fatalf("expected random test coordinate, got %v", err)
		}
		first, err := w.person(PersonIndex(value + 1))
		if err != nil {
			t.Fatalf("expected high-coordinate A person, got %v", err)
		}
		second, err := replay.person(PersonIndex(value + 1))
		if err != nil {
			t.Fatalf("expected high-coordinate B person, got %v", err)
		}
		if first != second {
			t.Fatal("expected stable noncontiguous access, got order dependence")
		}
	}
	for _, index := range []PersonIndex{1, 9999, 65536, Layer1MaxPersonIndex} {
		p, err := w.person(index)
		if err != nil {
			t.Fatalf("expected boundary person, got %v", err)
		}
		if index == Layer1MaxPersonIndex {
			samples = append(samples, []any{
				p.person, p.anchorAge, p.cohort, p.localityID, p.division, p.divisionOrdinal, p.addressLine,
			})
		}
	}
	summary := map[string]any{
		"status":          "INTERNAL CANDIDATE; NO PUBLIC WORLD OR PHOTO RIGHTS APPROVAL",
		"mainCoordinates": count, "ABGenerations": 2 * count, "noncontiguousABCoordinates": 1000,
		"boundaryCoordinates":  4,
		"recordFormat":         "test-only tuple: Person, anchorAge, cohort, localityID, Division, ordinal, addressLine",
		"orderedRecordsSHA256": fmt.Sprintf("%x", rolling.Sum(nil)), "baseLogicalDigest": fmt.Sprintf("%x", w.base.digest),
		"fingerprints": w.base.fingerprints, "phoneIdentity": fmt.Sprintf("%x", w.phones.identity),
		"observedAges": len(ages), "genders": genders, "cohorts": cohorts, "observedStates": len(states),
		"observedDivisions": len(divisions), "observedLocalities": len(localities),
		"observedStreets": len(streets), "samples": samples,
	}
	if path := os.Getenv("KRENALIS_USA_LAYER1_EVIDENCE_OUTPUT"); path != "" {
		encoded, err := internationalCanonical(summary)
		if err != nil {
			t.Fatalf("expected harness evidence serialization, got %v", err)
		}
		err = os.WriteFile(path, encoded, 0o644)
		if err != nil {
			t.Fatalf("expected written requested evidence, got %v", err)
		}
	}
	t.Logf("main=%d A/B=%d random=%d localities=%d states=%d divisions=%d digest=%x", count, 2*count, 1000,
		len(localities), len(states), len(divisions), rolling.Sum(nil))

}

// TestUSANPACompletion exhausts the fiction cycle and checks balanced reuse.
func TestUSANPACompletion(t *testing.T) {

	w := readUSACompletionHarness(t)
	count := uint64(len(w.phones.npas))
	period := 100 * count
	seen := map[string]bool{}
	for i := uint64(1); i <= period; i++ {
		phone, err := w.phones.phone(PersonIndex(i))
		if err != nil {
			t.Fatalf("expected reserved fiction number, got %v", err)
		}
		r := i - 1
		expected := "+1" + w.phones.npas[r%count] + "555" + fmt.Sprintf("%04d", 100+r/count)
		if phone != expected || len(phone) != 12 || seen[phone] {
			t.Fatalf("expected unique exact first-cycle number %s, got %s", expected, phone)
		}
		seen[phone] = true
		replayed, err := w.phones.phone(PersonIndex(i + period))
		if err != nil {
			t.Fatalf("expected repeated balanced number, got %v", err)
		}
		if replayed != phone {
			t.Fatalf("expected period %d, got %s/%s", period, phone, replayed)
		}
	}
	frequencies := map[string]int{}
	for i := uint64(123); i < 123+2*period+137; i++ {
		phone, err := w.phones.phone(PersonIndex(i))
		if err != nil {
			t.Fatalf("expected contiguous phone interval, got %v", err)
		}
		frequencies[phone]++
	}
	for phone, frequency := range frequencies {
		if frequency != 2 && frequency != 3 {
			t.Fatalf("expected balanced reuse two or three times, got %s=%d", phone, frequency)
		}
	}
	t.Logf("qualified NPAs=%d fiction period=%d tested contiguous=%d", count, period, 2*period+137)

}

// TestUSANPACompletionInvalid rejects unsafe or unidentified NPA inputs.
func TestUSANPACompletionInvalid(t *testing.T) {

	encoded := readUSACompletionFile(t, "assets/npas.json")
	cases := []struct {
		name   string
		change func(*usaNPADocument)
	}{
		{"source", func(d *usaNPADocument) { d.SourceSHA256 = strings.Repeat("0", 64) }},
		{"history", func(d *usaNPADocument) { d.HistoricalAsOf = "2026-10-16" }},
		{"status", func(d *usaNPADocument) { d.Status = "APPROVED" }},
		{"non-geographic", func(d *usaNPADocument) { d.Entries[0].NPA = "800" }},
		{"territory", func(d *usaNPADocument) { d.Entries[0].StateUSPS = "PR" }},
		{"future", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "2026-01-02" }},
		{"invalid-date", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "2025-02-30" }},
		{"missing", func(d *usaNPADocument) { d.Entries = d.Entries[:367] }},
		{"duplicate", func(d *usaNPADocument) { d.Entries[1] = d.Entries[0] }},
		{"wrong-state", func(d *usaNPADocument) { d.Entries[0].StateUSPS = "CA" }},
		{"wrong-date", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "1948-01-01" }},
		{"679", func(d *usaNPADocument) { d.Entries[0].NPA = "679" }},
		{"bound", func(d *usaNPADocument) {
			for len(d.Entries) <= 400 {
				d.Entries = append(d.Entries, d.Entries[0])
			}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var document usaNPADocument
			err := json.Unmarshal(encoded, &document)
			if err != nil {
				t.Fatalf("expected NPA fixture, got %v", err)
			}
			test.change(&document)
			input, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("expected mutated NPA document, got %v", err)
			}
			_, err = decodeUSANPACandidate(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected rejected NPA mutation, got nil")
		})
	}

}

// TestUSAStreetAddressTechnicalForms exercises APT/UNIT and leading-zero ZIP
// fixtures without enabling units in the initial distribution.
func TestUSAStreetAddressTechnicalForms(t *testing.T) {

	// Units are fixture-only: the initial distribution emits neither form.
	for _, test := range []struct{ street, line string }{
		{"1 Maple Street", "1 Maple Street, Boston, MA 02108"},
		{"9999 Oak Avenue APT 4B", "9999 Oak Avenue APT 4B, Boston, MA 02108"},
		{"42 Park Drive UNIT 12", "42 Park Drive UNIT 12, Boston, MA 02108"},
	} {
		address := Address{Street: test.street, Municipality: "Boston", StateProv: "MA", PostalCode: "02108", Country: "US"}
		line := fmt.Sprintf("%s, %s, %s %s", address.Street, address.Municipality, address.StateProv, address.PostalCode)
		if line != test.line || !validFixedDigits(address.PostalCode, 5) {
			t.Fatalf("expected technical address %s, got %s", test.line, line)
		}
	}

}

// readUSACompletionEvidence admits the unchanged AGE, nominal v2 and
// geography v2 inputs for the test or benchmark lifetime.
func readUSACompletionEvidence(t testing.TB) usaBaseEvidence {

	t.Helper()
	read := func(path string) []byte {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected existing adopted input, got %v", err)
		}
		return data
	}
	ages, err := decodeUSAAgeCandidate(t.Context(), bytes.NewReader(read(usaCandidateDirectory+"ages.json")))
	if err != nil {
		t.Fatalf("expected qualified AGE, got %v", err)
	}
	firstBytes := read(usaFirstNamesV2Directory + "assets/first-names.json")
	first, err := decodeUSAFirstNamesV2Candidate(t.Context(), bytes.NewReader(firstBytes))
	if err != nil {
		t.Fatalf("expected owner-adopted first names, got %v", err)
	}
	lastBytes := read(usaSurname50KDirectory + "assets/surnames.json")
	last, err := decodeUSASurname50KCandidate(t.Context(), bytes.NewReader(lastBytes))
	if err != nil {
		t.Fatalf("expected adopted 50K surnames, got %v", err)
	}
	geographyBytes := read(usaGeographyV2Directory + "assets/geography.json")
	geography, err := decodeUSAGeographyV2Candidate(t.Context(), bytes.NewReader(geographyBytes))
	if err != nil {
		t.Fatalf("expected adopted geography v2, got %v", err)
	}

	return usaBaseEvidence{ages: ages, firstNames: first, lastNames: last, geography: geography}
}

func readUSACompletionFile(t testing.TB, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(usaCompletionDirectory + name)
	if err != nil {
		t.Fatalf("expected new USA candidate file, got %v", err)
	}
	return encoded
}

// readUSACompletionHarness prepares actual USA assets and the private harness.
func readUSACompletionHarness(t testing.TB) *usaLayer1Harness {

	t.Helper()
	evidence := readUSACompletionEvidence(t)
	encoded := readUSACompletionFile(t, "assets/base.json")
	base, err := decodeUSABaseCandidate(t.Context(), bytes.NewReader(encoded), evidence)
	if err != nil {
		t.Fatalf("expected prepared national base, got %v", err)
	}
	phones, err := decodeUSANPACandidate(t.Context(), bytes.NewReader(readUSACompletionFile(t, "assets/npas.json")))
	if err != nil {
		t.Fatalf("expected qualified NANPA rule, got %v", err)
	}
	w, err := newUSALayer1Harness(IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026, base, phones)
	if err != nil {
		t.Fatalf("expected internal USA harness, got %v", err)
	}

	return w
}

// usaCompletionRecord encodes only test evidence, with the complete national
// address line as the final tuple value.
func usaCompletionRecord(t *testing.T, p usaLayer1Person) []byte {
	t.Helper()
	encoded, err := json.Marshal([]any{
		p.person, p.anchorAge, p.cohort, p.localityID, p.division, p.divisionOrdinal, p.addressLine,
	})
	if err != nil {
		t.Fatalf("expected deterministic harness record, got %v", err)
	}
	return encoded
}
