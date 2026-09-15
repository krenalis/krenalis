// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"net/url"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestLayer1AffineTokens verifies the fixed test-World parameters, coprimality,
// independence, token width, and operational-domain bijections.
func TestLayer1AffineTokens(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	if world.emailToken != (affinePermutation{a: 53_902_873, b: 51_295_596}) {
		t.Fatalf("expected email affine parameters 53902873/51295596, got %d/%d", world.emailToken.a, world.emailToken.b)
	}
	if world.siteToken != (affinePermutation{a: 30_673_949, b: 17_911_636}) {
		t.Fatalf("expected site affine parameters 30673949/17911636, got %d/%d", world.siteToken.a, world.siteToken.b)
	}
	if greatestCommonDivisor(world.emailToken.a, emailAndSiteTokenModulus) != 1 {
		t.Fatalf("expected coprime email multiplier, got %d", world.emailToken.a)
	}
	if greatestCommonDivisor(world.siteToken.a, emailAndSiteTokenModulus) != 1 {
		t.Fatalf("expected coprime site multiplier, got %d", world.siteToken.a)
	}
	if world.emailToken == world.siteToken {
		t.Fatalf("expected independent email/site permutations, got %#v", world.emailToken)
	}

	for _, index := range []PersonIndex{1, 2, 42, 1537291, Layer1MaxPersonIndex} {
		emailToken := encodeBase36Fixed5(world.emailToken.value(index))
		siteToken := encodeBase36Fixed5(world.siteToken.value(index))
		if !validBase36Token(emailToken) || !validBase36Token(siteToken) {
			t.Fatalf("expected five-character base36 tokens for %d, got %q/%q", index, emailToken, siteToken)
		}
		if emailToken == siteToken {
			t.Fatalf("expected independent tokens for %d, got %q", index, emailToken)
		}
	}

}

// TestLayer1EmailAndSiteSets verifies exact sets, grammar, isolated paths, and
// mechanical reachability without proportion assertions.
func TestLayer1EmailAndSiteSets(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	wantProviders := map[string]struct{}{
		"postauno.test": {}, "nuvolamail.test": {}, "mailia.test": {}, "casellablu.test": {},
		"lettera.test": {}, "reteposta.test": {}, "postafacile.test": {}, "casellamia.test": {},
	}
	wantPatterns := map[string]struct{}{
		"first.last.token": {}, "initial.last.token": {}, "first.token": {}, "first-last.token": {},
	}
	wantSiteDomains := map[string]struct{}{
		"profilo.test": {}, "pagina.test": {}, "identita.test": {}, "spaziopersonale.test": {},
	}
	if !sameStringSet(emailProviders.values, wantProviders) {
		t.Fatalf("expected provider set %#v, got %v", wantProviders, emailProviders.values)
	}
	if !sameStringSet(emailPatterns.values, wantPatterns) {
		t.Fatalf("expected pattern set %#v, got %v", wantPatterns, emailPatterns.values)
	}
	if !sameStringSet(siteDomains.values, wantSiteDomains) {
		t.Fatalf("expected site-domain set %#v, got %v", wantSiteDomains, siteDomains.values)
	}

	seenProviders := map[string]struct{}{}
	seenPatterns := map[string]struct{}{}
	seenSiteDomains := map[string]struct{}{}
	for index := PersonIndex(1); index <= 10_000; index++ {
		provider, err := world.weightedString(index, streamIdentityEmailProvider, emailProviders)
		if err != nil {
			t.Fatalf("expected provider for %d, got %v", index, err)
		}
		pattern, err := world.weightedString(index, streamIdentityEmailPattern, emailPatterns)
		if err != nil {
			t.Fatalf("expected pattern for %d, got %v", index, err)
		}
		domain, err := world.weightedString(index, streamIdentitySiteDomain, siteDomains)
		if err != nil {
			t.Fatalf("expected site domain for %d, got %v", index, err)
		}
		seenProviders[provider] = struct{}{}
		if _, exists := seenPatterns[pattern]; !exists {
			gender, err := world.gender(index)
			if err != nil {
				t.Fatalf("expected gender for pattern %s, got %v", pattern, err)
			}
			_, _, birthDate, err := world.birth(index)
			if err != nil {
				t.Fatalf("expected birth for pattern %s, got %v", pattern, err)
			}
			_, firstASCII, err := world.firstName(index, gender, birthDate.Year())
			if err != nil {
				t.Fatalf("expected first-name atom for pattern %s, got %v", pattern, err)
			}
			_, lastASCII, err := world.lastName(index)
			if err != nil {
				t.Fatalf("expected last-name atom for pattern %s, got %v", pattern, err)
			}
			email, err := world.email(index, firstASCII, lastASCII)
			if err != nil {
				t.Fatalf("expected email for pattern %s, got %v", pattern, err)
			}
			token := encodeBase36Fixed5(world.emailToken.value(index))
			wantLocal := ""
			switch pattern {
			case "first.last.token":
				wantLocal = firstASCII + "." + lastASCII + "." + token
			case "initial.last.token":
				wantLocal = firstASCII[:1] + "." + lastASCII + "." + token
			case "first.token":
				wantLocal = firstASCII + "." + token
			case "first-last.token":
				wantLocal = firstASCII + "-" + lastASCII + "." + token
			}
			if email != wantLocal+"@"+provider {
				t.Fatalf("expected pattern rendering %q, got %q", wantLocal+"@"+provider, email)
			}
		}
		seenPatterns[pattern] = struct{}{}
		seenSiteDomains[domain] = struct{}{}
	}
	if len(seenProviders) != len(wantProviders) {
		t.Fatalf("expected all providers reachable, got %v", seenProviders)
	}
	if len(seenPatterns) != len(wantPatterns) {
		t.Fatalf("expected all patterns reachable, got %v", seenPatterns)
	}
	if len(seenSiteDomains) != len(wantSiteDomains) {
		t.Fatalf("expected all site domains reachable, got %v", seenSiteDomains)
	}

	for _, index := range []PersonIndex{1, Layer1MaxPersonIndex} {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected boundary Person %d, got %v", index, err)
		}
		local, domain, found := strings.Cut(person.Email, "@")
		if !found {
			t.Fatalf("expected email address, got %q", person.Email)
		}
		if _, exists := wantProviders[domain]; !exists || !strings.HasSuffix(domain, ".test") {
			t.Fatalf("expected synthetic provider, got %q", domain)
		}
		if len(local) < 5 || !validBase36Token(local[len(local)-5:]) {
			t.Fatalf("expected trailing base36 token, got %q", local)
		}
		parsed, err := url.Parse(person.SiteURL)
		if err != nil {
			t.Fatalf("expected syntactically valid SiteURL, got %v", err)
		}
		if parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			t.Fatalf("expected canonical https SiteURL, got %q", person.SiteURL)
		}
		if _, exists := wantSiteDomains[parsed.Host]; !exists {
			t.Fatalf("expected synthetic site domain, got %q", parsed.Host)
		}
		if len(parsed.Path) != 6 || parsed.Path[0] != '/' || !validBase36Token(parsed.Path[1:]) {
			t.Fatalf("expected canonical SiteURL token path, got %q", parsed.Path)
		}
	}

}

// TestLayer1EmailUniqueness verifies 100,000 sequential and deterministically
// shuffled random-access email addresses.
func TestLayer1EmailUniqueness(t *testing.T) {
	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	const count = 100_000
	for _, shuffled := range []bool{false, true} {
		seen := make(map[string]struct{}, count)
		for position := 0; position < count; position++ {
			index := PersonIndex(position + 1)
			if shuffled {
				index = PersonIndex((position*7919)%count + 1)
			}
			email := emailForIndex(t, world, index)
			if _, exists := seen[email]; exists {
				t.Fatalf("expected unique email at index %d, got duplicate %q", index, email)
			}
			seen[email] = struct{}{}
		}
		if len(seen) != count {
			t.Fatalf("expected %d unique emails, got %d", count, len(seen))
		}
	}
}

// TestLayer1PhoneGoldenAndSemantics protects the frozen mapping and validates
// 100,000 deterministic outputs through the tracked phone semantic API.
func TestLayer1PhoneGoldenAndSemantics(t *testing.T) {

	goldens := []struct {
		index PersonIndex
		phone string
	}{
		{1, "+39000934416313"},
		{2, "+39000619422990"},
		{42, "+39000019689950"},
		{1537291, "+39000844059283"},
		{Layer1MaxPersonIndex, "+39000351602378"},
	}
	for _, golden := range goldens {
		if got := phoneForIndex(golden.index); got != golden.phone {
			t.Fatalf("expected phone %q for %d, got %q", golden.phone, golden.index, got)
		}
	}

	for _, boundary := range []string{"+39000000000010", "+39000999999999"} {
		normalized, ok := types.NormalizePhone(boundary)
		if !ok || normalized != boundary {
			t.Fatalf("expected boundary phone %q to normalize stably, got %q/%t", boundary, normalized, ok)
		}
		if !types.IsPhone(boundary) {
			t.Fatalf("expected boundary phone %q to be canonical and possible, got false", boundary)
		}
	}

	const count = 100_000
	seen := make(map[string]struct{}, count)
	for index := PersonIndex(1); index <= count; index++ {
		phone := phoneForIndex(index)
		if _, exists := seen[phone]; exists {
			t.Fatalf("expected unique phone at index %d, got duplicate %q", index, phone)
		}
		seen[phone] = struct{}{}
		normalized, ok := types.NormalizePhone(phone)
		if !ok || normalized != phone {
			t.Fatalf("expected canonical possible phone %q, got %q/%t", phone, normalized, ok)
		}
		if !types.IsPhone(phone) {
			t.Fatalf("expected phone recognition for %q, got false", phone)
		}
		again, ok := types.NormalizePhone(normalized)
		if !ok || again != phone {
			t.Fatalf("expected stable normalization %q, got %q/%t", phone, again, ok)
		}
	}
	if len(seen) != count {
		t.Fatalf("expected %d unique phones, got %d", count, len(seen))
	}

}

func emailForIndex(t *testing.T, world *World, index PersonIndex) string {
	t.Helper()
	gender, err := world.gender(index)
	if err != nil {
		t.Fatalf("expected gender for %d, got %v", index, err)
	}
	_, _, birthDate, err := world.birth(index)
	if err != nil {
		t.Fatalf("expected birth for %d, got %v", index, err)
	}
	_, firstASCII, err := world.firstName(index, gender, birthDate.Year())
	if err != nil {
		t.Fatalf("expected first name for %d, got %v", index, err)
	}
	_, lastASCII, err := world.lastName(index)
	if err != nil {
		t.Fatalf("expected last name for %d, got %v", index, err)
	}
	email, err := world.email(index, firstASCII, lastASCII)
	if err != nil {
		t.Fatalf("expected email for %d, got %v", index, err)
	}
	return email
}

func sameStringSet(values []string, want map[string]struct{}) bool {
	if len(values) != len(want) {
		return false
	}
	for _, value := range values {
		if _, exists := want[value]; !exists {
			return false
		}
	}
	return true
}

func validBase36Token(value string) bool {
	if len(value) != 5 {
		return false
	}
	for _, current := range []byte(value) {
		if (current < '0' || current > '9') && (current < 'a' || current > 'z') {
			return false
		}
	}
	return true
}
