//
//  author       : Marco Gazerro <gazerro@open2b.com>
//  initial date : 08/10/2007
//
// date : 12/04/2014
//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2014 Open2b
//

package culture

type CurrencyInfo struct {
	code          string
	name          string
	symbol        string
	decimalDigits int
}

var currencyInfos = []*CurrencyInfo{
	{"AED", "United Arab Emirates Dirham", "AED", 2},
	//{ "AFN", "Afghanistani afghani", 2, "Af " },
	{"ALL", "Albanian Lek", "Lek", 2},
	//{ "AMD", "Armenian Dram", 2 },
	//{ "ANG", "Netherlands Antillian Guilder", 2 },
	//{ "AOA", "Angolan Kwanza", 2 },
	{"ARS", "Argentine Peso", "$", 2},
	{"AUD", "Australian Dollar", "$", 2},
	//{ "AWG", "Aruban Florin", 2 },
	//{ "AZN", "Azerbaijan New Manat", 2 },
	{"BAM", "Bosnian Mark", "KM", 2},
	//{ "BBD", "Barbados Dollar", 2 },
	//{ "BDT", "Bangladeshi Taka", 2 },
	{"BGN", "Bulgarian Lev", "лв", 2},
	//{ "BHD", "Bahraini Dinar", 3 },
	//{ "BIF", "Burundi Franc", 0 },
	//{ "BMD", "Bermudian Dollar", 2 },
	//{ "BND", "Brunei Dollar", 2 },
	//{ "BOB", "Bolivian Boliviano", 2 },
	{"BRL", "Brazilian Real", "R$", 2},
	//{ "BSD", "Bahamian Dollar", 2 },
	//{ "BTN", "Bhutan Ngultrum", 2 },
	//{ "BWP", "Botswana Pula", 2 },
	//{ "BYR", "Belarusian Ruble", 0 },
	//{ "BZD", "Belize Dollar", 2 },
	{"CAD", "Canadian Dollar", "CDN$", 2},
	//{ "CDF", "Congolese Franc", 2 },
	{"CHF", "Swiss Franc", "CHF", 2},
	{"CLP", "Chilean Peso", "$", 0},
	{"CNY", "Chinese Yuan Renminbi", "¥", 2},
	{"COP", "Colombian Peso", "$", 0},
	{"CRC", "Costa Rican Colon", "", 2}, // !!!! il simbolo  !!!!
	//{ "CUP", "Cuban Peso", 2 },
	//{ "CVE", "Cape Verde Escudo", 2 },
	{"CZK", "Czech Koruna", "Kč", 2},
	//{ "DJF", "Djibouti Franc", 2 },
	{"DKK", "Danish Krone", "kr", 2},
	{"DOP", "Dominican R. Peso", "RD$", 2},
	//{ "DZD", "Algerian Dinar", 2 },
	{"EEK", "Estonian Kroon", "ks", 2},
	//{ "EGP", "Egyptian Pound", 2 },
	//{ "ERN", "Eritrean Nakfa", 2 },
	//{ "ETB", "Ethiopian Birr", 2 },
	{"EUR", "Euro", "€", 2},
	//{ "FJD", "Fiji Dollar", 2 },
	//{ "FKP", "Falkland Islands Pound", 2 },
	{"GBP", "British Pound Sterling", "£", 2},
	//{ "GEL", "Georgian Lari", 2 },
	//{ "GHS", "Ghanaian New Cedi", 2 },
	//{ "GIP", "Gibraltar Pound", 2 },
	//{ "GMD", "Gambian Dalasi", 2 },
	//{ "GNF", "Guinea Franc", 0 },
	{"GTQ", "Guatemalan Quetzal", "Q", 2},
	//{ "GYD", "Guyanese Dollar", 2 },
	//{ "HKD", "Hong Kong Dollar", 2 },
	{"HNL", "Honduran Lempira", "L.", 2},
	{"HRK", "Croatian Kuna", "kn", 2},
	//{ "HTG", "Haitian Gourde", 2 },
	{"HUF", "Hungarian Forint", "Ft", 2},
	//{ "IDR", "Indonesian Rupiah", 2 },
	{"ILS", "Israeli New Shekel", "₪", 2},
	{"INR", "Indian Rupee", "Rs.", 2},
	//{ "IQD", "Iraqi Dinar", 3 },
	//{ "IRR", "Iranian Rial", 2 },
	{"ISK", "Iceland Krona", "kr", 0},
	//{ "JMD", "Jamaican Dollar", 2 },
	//{ "JOD", "Jordanian Dinar", 3 },
	{"JPY", "Japanese Yen", "¥", 0},
	//{ "KES", "Kenyan Shilling", 2 },
	//{ "KGS", "Kyrgyzstanian Som", 2 },
	//{ "KHR", "Cambodian Riel", 2 },
	//{ "KMF", "Comoros Franc", 0 },
	//{ "KPW", "North Korean Won", 2 },
	//{ "KRW", "South-Korean Won", 0 },
	//{ "KWD", "Kuwaiti Dinar", 3 },
	//{ "KYD", "Cayman Islands Dollar", 2 },
	//{ "KZT", "Kazakhstan Tenge", 2 },
	//{ "LAK", "Lao Kip", 2 },
	//{ "LBP", "Lebanese Pound", 2 },
	//{ "LKR", "Sri Lanka Rupee", 2 },
	//{ "LRD", "Liberian Dollar", 2 },
	//{ "LSL", "Lesotho Loti", 2 },
	{"LTL", "Lithuanian Litas", "Lt", 2},
	{"LVL", "Latvian Lats", "Ls", 2},
	//{ "LYD", "Libyan Dinar", 3 },
	//{ "MAD", "Moroccan Dirham", 2 },
	//{ "MDL", "Moldovan Leu", 2 },
	//{ "MGA", "Malagasy Ariary", 0 },
	//{ "MKD", "Macedonian Denar", 2 },
	//{ "MMK", "Myanmar Kyat", 2 },
	//{ "MNT", "Mongolian Tugrik", 2 },
	//{ "MOP", "Macau Pataca", 2 },
	//{ "MRO", "Mauritanian Ouguiya", 2 },
	//{ "MTL", "Maltese Lira", 2 },
	//{ "MUR", "Mauritius Rupee", 2 },
	//{ "MVR", "Maldive Rufiyaa", 2 },
	//{ "MWK", "Malawi Kwacha", 2 },
	{"MXN", "Mexican Peso", "$", 2},
	//{ "MZN", "Mozambique New Metical" },
	//{ "NGN", "Nigerian Naira", 2 },
	{"NIO", "Nicaraguan Cordoba Oro", "C$", 2},
	{"NOK", "Norwegian Kroner", "kr", 2},
	//{ "NPR", "Nepalese Rupee", 2 },
	{"NZD", "New Zealand Dollar", "$", 2},
	//{ "OMR", "Omani Rial", 3 },
	{"PAB", "Panamanian Balboa", "B/.", 2},
	{"PEN", "Peruvian Nuevo Sol", "S/.", 2},
	//{ "PGK", "Papua New Guinea Kina", 2 },
	//{ "PHP", "Philippine Peso", 2 },
	//{ "PKR", "Pakistan Rupee", 2 },
	{"PLN", "Polish Zloty", "zł", 2},
	{"PYG", "Paraguay Guarani", "Gs", 2},
	//{ "QAR", "Qatari Rial", 2 },
	{"RON", "Romanian New Lei", "lei", 2},
	{"RSD", "Serbian Dinar", "din", 2},
	{"RUB", "Russian Rouble", "руб", 2},
	//{ "RWF", "Rwandan Franc", 0 },
	//{ "SAR", "Saudi Riyal", 2 },
	//{ "SBD", "Solomon Islands Dollar", 2 },
	//{ "SCR", "Seychelles Rupee", 2 },
	//{ "SDG", "Sudanese Pound", 2 },
	{"SEK", "Swedish Krona", "kr", 2},
	//{ "SGD", "Singapore Dollar", 2 },
	//{ "SHP", "St. Helena Pound", 2 },
	//{ "SKK", "Slovak Koruna", 2, "Slovakia" },
	//{ "SLL", "Sierra Leone Leone", 2 },
	//{ "SOS", "Somali Shilling", 2 },
	//{ "SRD", "Suriname Dollar", 2 },
	//{ "SYP", "Syrian Pound", 2 },
	//{ "SZL", "Swaziland Lilangeni", 2 },
	//{ "THB", "Thai Baht", 2 },
	//{ "TJS", "Tajikistani Somoni", 2 },
	//{ "TMM", "Turkmenistan Manat", 2 },
	{"TND", "Tunisian Dinar", "TND", 2},
	//{ "TOP", "Tonga Pa'anga", 2 },
	//{ "TRY", "Turkish New Lira", 2 },
	//{ "TTD", "Trinidad/Tobago Dollar", 2 },
	//{ "TWD", "Taiwan Dollar", 2 },
	//{ "TZS", "Tanzanian Shilling", 2 },
	{"UAH", "Ukraine Hryvnia", "грн", 2},
	//{ "UGX", "Uganda Shilling", 2 },
	{"USD", "US Dollar", "$", 2},
	{"UYU", "Uruguayan Peso", "$U", 2},
	//{ "UZS", "Uzbekistani Som", 2 },
	{"VEF", "Venezuelan Bolivar", "Bs", 2},
	//{ "VND", "Vietnamese Dong", 2 },
	//{ "VUV", "Vanuatu Vatu", 0 },
	//{ "WST", "Samoan Tala", 2 },
	//{ "XAF", "Central African CFA Franc", 0 },
	//{ "XCD", "East Caribbean Dollar", 2 },
	//{ "XOF", "CFA Franc", 0 },
	//{ "XPF", "CFP Franc", 0 },
	//{ "YER", "Yemeni Rial", 2 },
	{"ZAR", "South African Rand", "R", 2},
	//{ "ZMK", "Zambian Kwacha", 2 },
	//{ "ZWD", "Zimbabwe Dollar", 2, "\$", "" },
}

var currencyInfoByCode = map[string]*CurrencyInfo{}

func init() {
	for _, currency := range currencyInfos {
		currencyInfoByCode[currency.code] = currency
	}
}

func Currency(code string) *CurrencyInfo {
	return currencyInfoByCode[code]
}

func Currencies() []*CurrencyInfo {
	var currencies = make([]*CurrencyInfo, len(currencyInfos))
	for i, currency := range currencyInfos {
		currencies[i] = currency
	}
	return currencies
}

func (this *CurrencyInfo) Code() string {
	return this.code
}

func (this *CurrencyInfo) DecimalDigits() int {
	return this.decimalDigits
}

func (this *CurrencyInfo) Name() string {
	return this.name
}

func (this *CurrencyInfo) Symbol() string {
	return this.symbol
}
