//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002 Open2b
//

package culture

import (
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var addressPatterns = []string{
	"{pc} {C}", "{c}§{PC}", "{C} {sp} {pc}", "{pc} {c} ({sp})", "{C} {sp} {pc}",
	"{pc} {C} ({SPN})", "{C} {pc}", "{pc} {C} {sp}", "{c}§{pc} {SPN}", "{pc}§{C}",
	"{C} - {sp}§{pc}", "{pc} {SPN}", "{c}, {pc}"}

//
// NOTA: I campi NativeDigits,  AbbreviatedMonthGenitiveNames, MonthGenitiveNames
// e NativeCalendarName non sono currentemente implementati nella corrente
// versione di mono (1.2.5.1)
//

type LocaleInfo struct {
	englishName                string
	name                       string
	nativeName                 string
	threeLetterISOLanguageName string
	currencyDecimalDigits      int
	currencyDecimalSeparator   string
	currencyGroupSeparator     string
	currencyGroupSizes         []int
	currencyNegativePattern    int
	currencyPositivePattern    int
	currencySymbol             string
	nanSymbol                  string
	negativeInfinitySymbol     string
	negativeSign               string
	numberDecimalDigits        int
	numberDecimalSeparator     string
	numberGroupSeparator       string
	numberGroupSizes           int
	numberNegativePattern      int
	percentDecimalDigits       int
	percentDecimalSeparator    string
	percentGroupSeparator      string
	percentGroupSizes          int
	percentNegativePattern     int
	percentPositivePattern     int
	percentSymbol              string
	perMilleSymbol             string
	positiveInfinitySymbol     string
	positiveSign               string
	addressPattern             int
	abbreviatedDayNames        []string
	abbreviatedMonthNames      []string
	amDesignator               string
	dateSeparator              string
	dayNames                   []string
	firstDayOfWeek             string
	fullDateTimePattern        string
	longDatePattern            string
	longTimePattern            string
	monthDayPattern            string
	monthNames                 []string
	pmDesignator               string
	shortDatePattern           string
	shortestDayNames           []string
	shortTimePattern           string
	sortableDateTimePattern    string
	timeSeparator              string
	yearMonthPattern           string
}

var localeInfoByName = map[string]*LocaleInfo{}

func init() {
	for _, locale := range localeInfos {
		localeInfoByName[locale.name] = locale
	}
}

func Locale(name string) *LocaleInfo {

	if len(name) == 2 {
		switch name {
		case "bg":
			name = "bg-BG" // Bulgarian (Bulgaria)
		case "ca":
			name = "ca-ES" // Catalan (Spain)
		case "cs":
			name = "cs-CZ" // Czech (Czech Republic)
		case "da":
			name = "da-DK" // Danish (Denmark)
		case "de":
			name = "de-DE" // German (Germany)
		case "el":
			name = "el-GR" // Greek (Greece)
		case "en":
			name = "en-GB" // English (United Kingdom)
		case "es":
			name = "es-ES" // Spanish (Spain)
		case "et":
			name = "et-EE" // Estonian (Estonia)
		case "eu":
			name = "eu-ES" // Basque (Spain)
		case "fi":
			name = "fi-FI" // Finnish (Finland)
		case "fr":
			name = "fr-FR" // French (France)
		case "gl":
			name = "gl-ES" // Gallegan (Spain)
		case "hr":
			name = "hr-HR" // Croatian (Croatia)
		case "hu":
			name = "hu-HU" // Hungarian (Hungary)
		case "is":
			name = "is-IS" // Icelandic (Iceland)
		case "it":
			name = "it-IT" // Italian (Italy)
		case "lv":
			name = "lv-LV" // Latvian (Latvia)
		case "nb":
			name = "nb-NO" // Norwegian Bokmål (Norway)
		case "nl":
			name = "nl-NL" // Dutch (Netherlands)
		case "pl":
			name = "pl-PL" // Polish (Poland)
		case "pt":
			name = "pt-PT" // Portuguese (Portugal)
		case "ro":
			name = "ro-RO" // Romanian (Romania)
		case "ru":
			name = "ru-RU" // Russian (Russia)
		case "sk":
			name = "sk-SK" // Slovak (Slovakia)
		case "sl":
			name = "sl-SI" // Slovenian (Slovenia)
		case "sq":
			name = "sq-AL" // Albanian (Albania)
		case "sr":
			name = "sr-RS" // Serbian (Serbia)
		case "sv":
			name = "sv-SE" // Swedish (Sweden)
		case "uk":
			name = "uk-UA" // Ukrainian (Ukraine)
		}
	}

	return localeInfoByName[name]
}

func Locales() []*LocaleInfo {
	var locales = make([]*LocaleInfo, 0, len(localeInfos))
	for _, localeInfo := range localeInfos {
		locales = append(locales, localeInfo)
	}
	return locales
}

func (this *LocaleInfo) EnglishName() string {
	return this.englishName
}

func (this *LocaleInfo) Name() string {
	return this.name
}

func (this *LocaleInfo) NativeName() string {
	return this.nativeName
}

func (this *LocaleInfo) ThreeLetterISOLanguageName() string {
	return this.threeLetterISOLanguageName
}

func (this *LocaleInfo) CurrencyDecimalDigits() int {
	return this.currencyDecimalDigits
}

func (this *LocaleInfo) CurrencyDecimalSeparator() string {
	return this.currencyDecimalSeparator
}

func (this *LocaleInfo) CurrencyGroupSeparator() string {
	return this.currencyGroupSeparator
}

func (this *LocaleInfo) CurrencyGroupSizes() []int {
	var sizes = make([]int, 0, len(this.currencyGroupSizes))
	for _, size := range this.currencyGroupSizes {
		sizes = append(sizes, size)
	}
	return sizes
}

func (this *LocaleInfo) CurrencyNegativePattern() int {
	return this.currencyNegativePattern
}

func (this *LocaleInfo) CurrencyPositivePattern() int {
	return this.currencyPositivePattern
}

func (this *LocaleInfo) CurrencySymbol() string {
	return this.currencySymbol
}

func (this *LocaleInfo) NaNSymbol() string {
	return this.nanSymbol
}

func (this *LocaleInfo) NegativeInfinitySymbol() string {
	return this.negativeInfinitySymbol
}

func (this *LocaleInfo) NegativeSign() string {
	return this.negativeSign
}

func (this *LocaleInfo) NumberDecimalDigits() int {
	return this.numberDecimalDigits
}

func (this *LocaleInfo) NumberDecimalSeparator() string {
	return this.numberDecimalSeparator
}

func (this *LocaleInfo) NumberGroupSeparator() string {
	return this.numberGroupSeparator
}

func (this *LocaleInfo) NumberGroupSizes() int {
	return this.numberGroupSizes
}

func (this *LocaleInfo) NumberNegativePattern() int {
	return this.numberNegativePattern
}

func (this *LocaleInfo) PercentDecimalDigits() int {
	return this.percentDecimalDigits
}

func (this *LocaleInfo) PercentDecimalSeparator() string {
	return this.percentDecimalSeparator
}

func (this *LocaleInfo) PercentGroupSeparator() string {
	return this.percentGroupSeparator
}

func (this *LocaleInfo) PercentGroupSizes() int {
	return this.percentGroupSizes
}

func (this *LocaleInfo) PercentNegativePattern() int {
	return this.percentNegativePattern
}

func (this *LocaleInfo) PercentPositivePattern() int {
	return this.percentPositivePattern
}

func (this *LocaleInfo) PercentSymbol() string {
	return this.percentSymbol
}

func (this *LocaleInfo) PerMilleSymbol() string {
	return this.perMilleSymbol
}

func (this *LocaleInfo) PositiveInfinitySymbol() string {
	return this.positiveInfinitySymbol
}

func (this *LocaleInfo) PositiveSign() string {
	return this.positiveSign
}

func (this *LocaleInfo) AddressPattern() string {
	return addressPatterns[this.addressPattern]
}

func (this *LocaleInfo) AbbreviatedDayNames() []string {
	var names = make([]string, 0, len(this.abbreviatedDayNames))
	for _, name := range this.abbreviatedDayNames {
		names = append(names, name)
	}
	return names
}

func (this *LocaleInfo) AbbreviatedMonthNames() []string {
	var names = make([]string, 0, len(this.abbreviatedMonthNames))
	for _, name := range this.abbreviatedMonthNames {
		names = append(names, name)
	}
	return names
}

func (this *LocaleInfo) AMDesignator() string {
	return this.amDesignator
}

func (this *LocaleInfo) DateSeparator() string {
	return this.dateSeparator
}

func (this *LocaleInfo) DayNames() []string {
	var names = make([]string, 0, len(this.dayNames))
	for _, name := range this.dayNames {
		names = append(names, name)
	}
	return names
}

func (this *LocaleInfo) FirstDayOfWeek() string {
	return this.firstDayOfWeek
}

func (this *LocaleInfo) FullDateTimePattern() string {
	return this.fullDateTimePattern
}

func (this *LocaleInfo) LongDatePattern() string {
	return this.longDatePattern
}

func (this *LocaleInfo) LongTimePattern() string {
	return this.longTimePattern
}

func (this *LocaleInfo) MonthDayPattern() string {
	return this.monthDayPattern
}

func (this *LocaleInfo) MonthNames() []string {
	var names = make([]string, 0, len(this.monthNames))
	for _, name := range this.monthNames {
		names = append(names, name)
	}
	return names
}

func (this *LocaleInfo) PMDesignator() string {
	return this.pmDesignator
}

func (this *LocaleInfo) ShortDatePattern() string {
	return this.shortDatePattern
}

func (this *LocaleInfo) ShortestDayNames() []string {
	var names = make([]string, 0, len(this.shortestDayNames))
	for _, name := range this.shortestDayNames {
		names = append(names, name)
	}
	return names
}

func (this *LocaleInfo) ShortTimePattern() string {
	return this.shortTimePattern
}

func (this *LocaleInfo) SortableDateTimePattern() string {
	return this.sortableDateTimePattern
}

func (this *LocaleInfo) TimeSeparator() string {
	return this.timeSeparator
}

func (this *LocaleInfo) YearMonthPattern() string {
	return this.yearMonthPattern
}

// Ritorna il nome abbreviato, specifico della lingua, del giorno della settimana specificato
func (this *LocaleInfo) AbbreviatedDayName(weekday time.Weekday) string {
	if weekday < 0 || 6 < weekday {
		panic("culture: invalid dayofweek " + strconv.Itoa(int(weekday)))
	}
	return this.abbreviatedDayNames[weekday]
}

// Ritorna il nome abbreviato, specifico della lingua, del mese specificato
func (this *LocaleInfo) AbbreviatedMonthName(month time.Month) string {
	return this.abbreviatedMonthNames[month-1]
}

// Ritorna il nome esteso, specifico della lingua, del giorno della settimana specificato
func (this *LocaleInfo) DayName(weekday time.Weekday) string {
	return this.dayNames[weekday]
}

// Ritorna il codice della lingua
func (this *LocaleInfo) LanguageCode() string {
	return this.name[:strings.IndexByte(this.name, '-')]
}

// Ritorna il nome inglese della lingua
func (this *LocaleInfo) LanguageName() string {
	var nativeName = this.nativeName[:strings.IndexByte(this.nativeName, '(')-1]
	var englishName = this.englishName[:strings.IndexByte(this.englishName, '(')-1]
	return toUpperFirst(nativeName) + " (" + englishName + ")"
}

// Ritorna il nome esteso, specifico della lingua, del mese specificato
func (this *LocaleInfo) MonthName(month time.Month) string {
	if month < 1 || 12 < month {
		panic("Invalid month: " + strconv.Itoa(int(month)))
	}
	return this.monthNames[month-1]
}

// Ritorna il codice della regione
func (this *LocaleInfo) RegionCode() string {
	return this.name[strings.IndexByte(this.name, '-')+1:]
}

// Ritorna il nome inglese della regione
func (this *LocaleInfo) RegionName() string {
	var nativeName = this.nativeName[strings.IndexByte(this.nativeName, '(')+1 : len(this.nativeName)-1]
	var englishName = this.englishName[strings.IndexByte(this.englishName, '(')+1 : len(this.englishName)-1]
	return toUpperFirst(nativeName) + " (" + englishName + ")"
}

// Ritorna il nome abbreviato più corto, specifico della lingua, del giorno delle settimana specificato
func (this *LocaleInfo) ShortestDayName(day int) string {
	if day < 0 || day > 6 {
		panic("Invalid day: " + strconv.Itoa(day))
	}
	return this.shortestDayNames[day]
}

// Formato per un orario, basato sul RFC 1123. Corrisponde ai caratteri di formattazione "r" e "R"
func RFC1123Pattern() string {
	return "ddd, dd MMM yyyy HH':'mm':'ss 'GMT'"
}

// Formato per una data e ora ordinabili. Corrisponde al carattere di formattazione "s"
func SortableDateTimePattern() string {
	return "yyyy'-'MM'-'dd'T'HH':'mm':'ss"
}

// Codice ISO 639-1 di due lettere della lingua. Es. "fr"
func (this *LocaleInfo) TwoLetterISOLanguageName() string {
	return this.name[:2]
}

// Formato per un valore di data e ora universali ordinabili. Corrisponde al carattere di formattazione "u"
func UniversalSortableDateTimePattern() string {
	return "yyyy'-'MM'-'dd HH':'mm':'ss'Z'"
}

//
// private methods
//

func toUpperFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
