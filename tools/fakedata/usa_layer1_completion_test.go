// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/krenalis/krenalis/tools/json"
	"math"
	"net/mail"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
		_, err := w.person(index)
		if err != nil {
			t.Fatalf("expected boundary person, got %v", err)
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

// readUSACompletionHarness selects Layer 1 from the remotely loaded USA World.
func readUSACompletionHarness(t *testing.T) *usaLayer1Harness {
	t.Helper()
	return readUSAIncomeHarness(t).categorical.base
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
