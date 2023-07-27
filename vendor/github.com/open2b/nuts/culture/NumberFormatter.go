//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2018 Open2b
//

package culture

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/open2b/nuts/decimal"
)

var errInvalidNumberFormat = errors.New("culture: invalid number format")
var errInvalidNumberString = errors.New("culture: string to parse is not a valid number")
var errNullLocale = errors.New("culture: locale cannot be null")

var numberNegativePatterns = []string{"(n)", "-n", "- n", "n-", "n -"}
var percentPositivePatterns = []string{"n %", "n%", "%n", "% n"}
var percentNegativePatterns = []string{"-n %", "-n%", "-%n", "%-n", "%n-", "n-%",
	"n%-", "-% n", "n %-", "% n-", "% -n", "n- %"}

var numberFormatReg = regexp.MustCompile(`^(?:([0-9]{0,100})\-)?([0-9]{0,100})(%?)$`)

//
//   format  : [(minDigits-)maxDigits][(zeros)][%]
//
//       minDigits: numero minimo di cifre decimali da visualizzare
//       maxDigits: numero massimo di cifre decimali da visualizzare
//            '3'   : 3 cifre
//            '2-5' : da 2 a 5 cifre, si visualizzeranno più di 2 cifre solo se non sono '0'
//
//       % : se presente si formatteranno numeri percentuali
//
// un format non definito è uguale a '0'
//

// FormatNumber formatta d secondo il formato format e nel locale indicato.
func FormatNumber(d decimal.Dec, locale *LocaleInfo, format string) string {
	return NewNumberFormatter(locale, format).Format(d)
}

type NumberFormatter struct {
	minDigits        int
	maxDigits        int
	decimalSeparator string
	groupSeparator   string
	groupSizes       int
	negativePattern  string
	percentSymbol    string
	positivePattern  string
}

// NewNumberFormatter ritorna un formatter che formatta un decimale
// secondo il locale e il formato indicati.
func NewNumberFormatter(locale *LocaleInfo, format string) *NumberFormatter {

	if locale == nil {
		panic(errNullLocale)
	}

	var minDigits int
	var maxDigits int
	var usePercent bool

	if format != "" {
		parts := numberFormatReg.FindStringSubmatch(format)
		if len(parts) != 4 {
			panic(errInvalidNumberFormat)
		}
		if dd, err := strconv.Atoi(parts[2]); err == nil {
			minDigits = dd
			maxDigits = dd
		}
		if dd, err := strconv.Atoi(parts[1]); err == nil {
			minDigits = dd
		}
		usePercent = parts[3] == "%"
	}

	var nf = &NumberFormatter{
		minDigits:     minDigits,
		maxDigits:     maxDigits,
		percentSymbol: locale.PercentSymbol(),
	}

	if usePercent {
		nf.decimalSeparator = locale.PercentDecimalSeparator()
		nf.groupSeparator = locale.PercentGroupSeparator()
		nf.groupSizes = locale.PercentGroupSizes()
		nf.negativePattern = percentNegativePatterns[locale.PercentNegativePattern()]
		nf.positivePattern = percentPositivePatterns[locale.PercentPositivePattern()]
	} else {
		nf.decimalSeparator = locale.NumberDecimalSeparator()
		nf.groupSeparator = locale.NumberGroupSeparator()
		nf.groupSizes = locale.NumberGroupSizes()
		nf.negativePattern = numberNegativePatterns[locale.NumberNegativePattern()]
		nf.positivePattern = "n"
	}

	return nf
}

func (nf *NumberFormatter) Format(amount decimal.Dec) string {

	minDigits := nf.minDigits
	maxDigits := nf.maxDigits

	amount = amount.Rounded(maxDigits, "HalfDown")

	var isNegative = amount.IsNegative()
	var integerPart string
	var decimalPart string
	{
		parts := strings.Split(amount.String(), ".")
		integerPart = parts[0]
		if isNegative {
			integerPart = integerPart[1:]
		}
		if len(parts) > 1 {
			decimalPart = parts[1]
		}
	}

	if nf.groupSeparator != "" {
		for i := len(integerPart); i > nf.groupSizes; {
			i -= nf.groupSizes
			integerPart = integerPart[:i] + nf.groupSeparator + integerPart[i:]
		}
	}

	if minDigits < len(decimalPart) {
		re := regexp.MustCompile(`0{1,` + strconv.Itoa(len(decimalPart)-minDigits) + `}$`)
		decimalPart = re.ReplaceAllString(decimalPart, "")
	} else if maxDigits > len(decimalPart) {
		decimalPart += strings.Repeat("0", maxDigits-len(decimalPart))
	}

	var formattedAmount string
	if isNegative {
		formattedAmount = nf.negativePattern
	} else {
		formattedAmount = nf.positivePattern
	}

	var newAmount = integerPart
	if decimalPart != "" {
		newAmount += nf.decimalSeparator + decimalPart
	}

	formattedAmount = strings.Replace(formattedAmount, "n", newAmount, 1)
	formattedAmount = strings.Replace(formattedAmount, "%", nf.percentSymbol, 1)

	return formattedAmount
}

func (nf *NumberFormatter) Parse(str string, maxDigits int) (decimal.Dec, error) {

	if str == "" {
		return decimal.Dec{}, errInvalidNumberString
	}

	if maxDigits == -1 {
		maxDigits = nf.maxDigits
	}

	var gsizes = nf.groupSizes
	var gsizesf = gsizes - 1

	// Nota: nei paesi arabi il segno si mette sulla destra.
	// Nota: in alcune lingue, come quelle dell'India e alcune locali del Canada, GroupSizes prende più valori

	reg, err := regexp.Compile(`^\s*([+-])?\s*(\d+|[1-9]\d{0,` + strconv.Itoa(gsizesf) + `}(?:` +
		regexp.QuoteMeta(nf.groupSeparator) + `\d{` + strconv.Itoa(gsizes) + `})*)(?:` +
		regexp.QuoteMeta(nf.decimalSeparator) + `(\d+))?\s*$`)
	if err != nil {
		return decimal.Dec{}, err
	}

	var isNegative bool
	var integerPart, decimalPart int
	if parts := reg.FindStringSubmatch(str); len(parts) == 4 {
		isNegative = parts[1] == "-"
		parts[2] = strings.ReplaceAll(parts[2], nf.groupSeparator, "")
		integerPart, _ = strconv.Atoi(parts[2])
		decimalPart, _ = strconv.Atoi(parts[3])
	} else {
		return decimal.Dec{}, errInvalidNumberString
	}

	var newAmount string
	if isNegative {
		newAmount = "-"
	}
	newAmount += strconv.Itoa(integerPart)
	if decimalPart > 0 {
		newAmount += "." + strconv.Itoa(decimalPart)
	}

	return decimal.String(newAmount).Rounded(maxDigits, "Down"), nil
}
