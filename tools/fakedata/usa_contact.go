// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"strings"
)

const usaLayer1StreamPrefix = "us/v1/"

// usaContacts contains only the two observational contact permutations.
type usaContacts struct {
	emailToken affinePermutation
	siteToken  affinePermutation
}

// newUSAContacts reuses the original affine procedure with national paths.
func newUSAContacts(factory streamFactory) (*usaContacts, error) {

	email, err := deriveAffinePermutation(factory, usaLayer1StreamPrefix+streamIdentityEmailToken)
	if err != nil {
		return nil, err
	}
	site, err := deriveAffinePermutation(factory, usaLayer1StreamPrefix+streamIdentitySiteToken)
	if err != nil {
		return nil, err
	}

	return &usaContacts{emailToken: email, siteToken: site}, nil
}

// compose validates coordinates before affine arithmetic and truncates only
// nominal prefixes; the full independent five-character tokens survive.
func (c *usaContacts) compose(index PersonIndex, first, last string) (string, string, error) {

	if index == 0 {
		return "", "", ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return "", "", ErrLayer1IndexOutOfRange
	}
	if c == nil || c.emailToken.a == 0 || c.siteToken.a == 0 ||
		c.emailToken.a >= emailAndSiteTokenModulus || c.siteToken.a >= emailAndSiteTokenModulus ||
		c.emailToken.b >= emailAndSiteTokenModulus || c.siteToken.b >= emailAndSiteTokenModulus ||
		greatestCommonDivisor(c.emailToken.a, emailAndSiteTokenModulus) != 1 ||
		greatestCommonDivisor(c.siteToken.a, emailAndSiteTokenModulus) != 1 {
		return "", "", fmt.Errorf("uninitialized USA contact permutations")
	}
	if len(first) > 80 || len(last) > 80 || !internationalASCIIAtom(first) || !internationalASCIIAtom(last) {
		return "", "", fmt.Errorf("invalid USA nominal contact atoms")
	}

	nominal := first + "." + last
	prefix := strings.TrimRight(nominal[:min(len(nominal), 58)], ".-")
	email := prefix + "." + encodeBase36Fixed5(c.emailToken.value(index)) + "@people.example.test"
	nominal = first + "-" + last
	prefix = strings.TrimRight(nominal[:min(len(nominal), 57)], "-")
	site := "https://" + prefix + "-" + encodeBase36Fixed5(c.siteToken.value(index)) + ".profiles.example.test"

	return email, site, nil
}
