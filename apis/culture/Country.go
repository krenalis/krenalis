//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2008 Open2b
//

package culture

var countryInfoByCode = map[string]*CountryInfo{}

type CountryInfo struct {
	code            string
	code3           string
	name            string
	kind            string
	stateProvs      []*StateProvInfo
	stateProvByCode map[string]*StateProvInfo
}

type StateProvInfo struct {
	code string
	name string
}

func init() {
	for _, country := range countryInfos {
		country.stateProvByCode = map[string]*StateProvInfo{}
		countryInfoByCode[country.code] = country
		for _, stateProv := range country.stateProvs {
			country.stateProvByCode[stateProv.code] = stateProv
		}
	}
}

func Country(code string) *CountryInfo {
	return countryInfoByCode[code]
}

func Countries() []*CountryInfo {
	var countries = make([]*CountryInfo, len(countryInfos))
	for i, country := range countryInfos {
		countries[i] = country
	}
	return countries
}

func StateProv(country, code string) *StateProvInfo {
	if countryInfo, ok := countryInfoByCode[country]; ok {
		return countryInfo.stateProvByCode[code]
	}
	return nil
}

func StateProvs(country string) []*StateProvInfo {
	if countryInfo, ok := countryInfoByCode[country]; ok {
		var stateProvs = make([]*StateProvInfo, len(countryInfo.stateProvs))
		for i, stateProv := range countryInfo.stateProvs {
			stateProvs[i] = stateProv
		}
		return stateProvs
	}
	return nil
}

func (country *CountryInfo) Code() string {
	return country.code
}

func (country *CountryInfo) Name() string {
	return country.name
}

func (country *CountryInfo) ThreeLetterISOCode() string {
	return country.code3
}

func (stateProv *StateProvInfo) Code() string {
	return stateProv.code
}

func (stateProv *StateProvInfo) Name() string {
	return stateProv.name
}

func IsCountryCode(c string) bool {
	return len(c) == 2 && c[0] >= 63 && c[0] <= 90 && c[1] >= 63 && c[1] <= 90
}

func IsStateProvCode(s string) bool {
	if !(len(s) > 0 && len(s) <= 3) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !((s[i] >= '0' && s[i] <= '9') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
			return false
		}
	}
	return true
}
