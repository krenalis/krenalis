// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"strings"
)

// locales contains locale codes, with the default appearing first, e.g.,
// "en-US" before "en-AU".
const locales = "en-USen-GBen-CAen-AUen-IEen-NZen-INen-ZAen-PHen-SGen-TTfr-FRfr-CAfr-BEfr-CHfr-LUit-ITit-CHde-DEde-ATde-CHes-ESes-MXes-ARes-COes-PEes-CLes-USes-VEes-ECpt-BRpt-PTnl-NLnl-BEsv-SEda-DKno-NBno-NOpl-PLru-RUru-UAbg-BGro-ROcs-CZhu-HUit-VIja-JPzh-CNzh-TWzh-HKzh-SGzh-MOko-KRhe-ILar-SAar-AEfi-FIel-GRtr-TRsk-ISsl-SImk-MKal-ALhr-HRlt-LTlv-LVet-EEuk-UAms-MYms-SGid-IDvi-VNth-THhi-INbn-INta-INml-INur-PKfa-IRps-AFyo-NGsw-KEzu-ZAga-IEsq-ALmt-MTcy-GBaz-AZkk-KZuz-UZky-KGhy-AMtg-TJmn-MNka-GEfo-FOsm-MXlo-LAkm-KHbo-INsi-LK"

// countryCode returns the provided code and a boolean indicating whether it is
// a valid country code.
func countryCode(code string) (string, bool) {
	if len(code) == 2 && 'A' <= code[0] && code[0] <= 'Z' && 'A' <= code[1] && code[1] <= 'Z' {
		if i := strings.Index(locales, code); i >= 0 {
			return locales[i : i+2], true
		}
	}
	return "", false
}

// localeCode returns the locale code and a boolean indicating whether the
// provided code is a valid locale code. If a language code (e.g., "en") is
// provided, it returns the default locale for that language (e.g., "en-US")
// along with true.
func localeCode(code string) (string, bool) {
	switch {
	case len(code) == 5 && code[2] == '-':
		fallthrough
	case len(code) == 2 && 'a' <= code[0] && code[0] <= 'z' && 'a' <= code[1] && code[1] <= 'z':
		if i := strings.Index(locales, code); i >= 0 {
			return locales[i : i+5], true
		}
	}
	return "", false
}
