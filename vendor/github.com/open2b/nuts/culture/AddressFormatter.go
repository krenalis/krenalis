//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2018 Open2b
//

package culture

import (
	"strings"
	"unicode"
)

type Address struct {
	Name        string
	FirstName   string
	LastName    string
	CompanyName string
	Street1     string
	Street2     string
	City        string
	PostalCode  string
	StateProv   string
	Country     string
}

var patternOfCountry map[string]string

func init() {
	patternOfCountry = map[string]string{}
	for _, locale := range localeInfos {
		c := locale.RegionCode()
		if _, ok := patternOfCountry[c]; !ok {
			patternOfCountry[c] = locale.AddressPattern()
		}
	}
}

func FormatAddress(address *Address) []string {

	pattern, ok := patternOfCountry[address.Country]
	if !ok {
		return nil
	}

	var spn string
	if strings.Contains(pattern, "{SPN}") && address.StateProv != "" {
		if sp := StateProv(address.Country, address.StateProv); sp != nil {
			spn = strings.ToUpper(sp.Name())
		}
	}

	r := strings.NewReplacer(
		"{pc}", address.PostalCode,
		"{PC}", strings.ToUpper(address.PostalCode),
		"{c}", address.City,
		"{C}", strings.ToUpper(address.City),
		"{sp}", address.StateProv,
		"{SPN}", spn,
	)

	var rows = strings.Split(pattern, "§")
	for i, row := range rows {
		rows[i] = r.Replace(row)
	}

	if !containsOnlySpaces(address.Street2) {
		rows = append([]string{address.Street2}, rows...)
	}
	rows = append([]string{address.Street1}, rows...)
	if address.Name != "" {
		rows = append([]string{address.Name}, rows...)
	} else if address.FirstName != "" {
		rows = append([]string{address.FirstName + " " + address.LastName}, rows...)
	}
	if address.CompanyName != "" {
		rows = append([]string{address.CompanyName}, rows...)
	}
	if country := Country(address.Country); country != nil {
		rows = append(rows, country.Name())
	}

	return rows
}

func containsOnlySpaces(s string) bool {
	for _, c := range s {
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}
