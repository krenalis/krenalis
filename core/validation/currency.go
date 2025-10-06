//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package validation

import (
	"strings"
)

// IsValidCurrencyCode returns true when code is a valid ISO 4217 currency identifier.
func IsValidCurrencyCode(code string) bool {
	// Fast path.
	switch code {
	case "USD", "EUR", "JPY", "GBP", "AUD", "CNY":
		return true
	}
	// Slow path.
	if len(code) != 3 {
		return false
	}
	idx := strings.Index(currencyCodes, code)
	return idx >= 0 && idx%3 == 0
}

// All ISO 4217 currency codes except those handled in the fast path.
const currencyCodes = "" +
	"AED" +
	"AFN" +
	"ALL" +
	"AMD" +
	"ANG" +
	"AOA" +
	"ARS" +
	"AWG" +
	"AZN" +
	"BAM" +
	"BBD" +
	"BDT" +
	"BGN" +
	"BHD" +
	"BIF" +
	"BMD" +
	"BND" +
	"BOB" +
	"BOV" +
	"BRL" +
	"BSD" +
	"BTN" +
	"BWP" +
	"BYN" +
	"BZD" +
	"CAD" +
	"CDF" +
	"CHE" +
	"CHF" +
	"CHW" +
	"CLF" +
	"CLP" +
	"COP" +
	"COU" +
	"CRC" +
	"CUP" +
	"CVE" +
	"CZK" +
	"DJF" +
	"DKK" +
	"DOP" +
	"DZD" +
	"EGP" +
	"ERN" +
	"ETB" +
	"FJD" +
	"FKP" +
	"GEL" +
	"GHS" +
	"GIP" +
	"GMD" +
	"GNF" +
	"GTQ" +
	"GYD" +
	"HKD" +
	"HNL" +
	"HTG" +
	"HUF" +
	"IDR" +
	"ILS" +
	"INR" +
	"IQD" +
	"IRR" +
	"ISK" +
	"JMD" +
	"JOD" +
	"KES" +
	"KGS" +
	"KHR" +
	"KMF" +
	"KPW" +
	"KRW" +
	"KWD" +
	"KYD" +
	"KZT" +
	"LAK" +
	"LBP" +
	"LKR" +
	"LRD" +
	"LSL" +
	"LYD" +
	"MAD" +
	"MDL" +
	"MGA" +
	"MKD" +
	"MMK" +
	"MNT" +
	"MOP" +
	"MRU" +
	"MUR" +
	"MVR" +
	"MWK" +
	"MXN" +
	"MXV" +
	"MYR" +
	"MZN" +
	"NAD" +
	"NGN" +
	"NIO" +
	"NOK" +
	"NPR" +
	"NZD" +
	"OMR" +
	"PAB" +
	"PEN" +
	"PGK" +
	"PHP" +
	"PKR" +
	"PLN" +
	"PYG" +
	"QAR" +
	"RON" +
	"RSD" +
	"RUB" +
	"RWF" +
	"SAR" +
	"SBD" +
	"SCR" +
	"SDG" +
	"SEK" +
	"SGD" +
	"SHP" +
	"SLE" +
	"SLL" +
	"SOS" +
	"SRD" +
	"SSP" +
	"STN" +
	"SVC" +
	"SYP" +
	"SZL" +
	"THB" +
	"TJS" +
	"TMT" +
	"TND" +
	"TOP" +
	"TRY" +
	"TTD" +
	"TWD" +
	"TZS" +
	"UAH" +
	"UGX" +
	"USN" +
	"UYI" +
	"UYU" +
	"UYW" +
	"UZS" +
	"VED" +
	"VES" +
	"VND" +
	"VUV" +
	"WST" +
	"XAF" +
	"XAG" +
	"XAU" +
	"XBA" +
	"XBB" +
	"XBC" +
	"XBD" +
	"XCD" +
	"XDR" +
	"XOF" +
	"XPD" +
	"XPF" +
	"XPT" +
	"XSU" +
	"XTS" +
	"XUA" +
	"XXX" +
	"YER" +
	"ZAR" +
	"ZMW" +
	"ZWL"
