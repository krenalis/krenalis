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

var errInvalidCurrencyFormat = errors.New("culture: invalid currency format")
var errNullCurrency = errors.New("culture: currency cannot be null")

var positivePatterns = []string{`$n`, `n$`, `$ n`, `n $`}
var negativePatterns = []string{`($n)`, `-$n`, `$-n`, `$n-`, `(n$)`, `-n$`, `n-$`,
	`n$-`, `-n $`, `-$ n`, `n $-`, `$ n-`, `$ -n`, `n- $`, `($ n)`, `(n $)`}

//
// format:
//
// [symbol][digits]
//
// symbol: il simbolo della valuta da visualizzare
//         's': il simbolo di default
//         'c': il codice della valuta
//         'n': nessun simbolo
// digits: numero di cifre decimali da visualizzare
//         '3' : 3 cifre
//         '+3': 3 cifre oltre a quelle della valuta
//         ''  : quelle della valuta
//
// un format non definito è uguale a 's'
//

// FormatCurrency formatta d secondo il formato format e nel locale indicato.
func FormatCurrency(d decimal.Dec, locale *LocaleInfo, currency *CurrencyInfo, format string, decimalDigits int) string {
	return NewCurrencyFormatter(locale, currency, format).format(d, decimalDigits)
}

type CurrencyFormatter struct {
	decimalDigits    int
	decimalSeparator string
	groupSeparator   string
	groupSizes       int
	negativePattern  int
	positivePattern  int
	symbol           string
}

var formatCurrencyReg = regexp.MustCompile(`^(?:([scn])?(?:(\+)?([0-9]+))?)?`)

// NewCurrencyFormatter ritorna un formatter che formatta un decimale
// secondo il locale, la currency e il formato indicati.
func NewCurrencyFormatter(locale *LocaleInfo, currency *CurrencyInfo, format string) *CurrencyFormatter {

	if locale == nil {
		panic(errNullLocale)
	}

	if currency == nil {
		panic(errNullCurrency)
	}

	var symbol string
	var digits int

	if format == "" {
		symbol = currency.Symbol()
		digits = currency.DecimalDigits()
	} else {
		parts := formatCurrencyReg.FindStringSubmatch(format)
		if len(parts) != 4 {
			panic(errInvalidCurrencyFormat)
		}
		useSymbol := parts[1]
		if useSymbol == "c" {
			symbol = currency.Code()
		} else if useSymbol != "n" {
			symbol = currency.Symbol()
		}
		if parts[3] == "" {
			digits = currency.DecimalDigits()
		} else {
			useDigits, _ := strconv.Atoi(parts[3])
			if parts[2] != "" {
				useDigits += currency.DecimalDigits()
			}
			digits = useDigits
		}
	}

	return &CurrencyFormatter{
		decimalDigits:    digits,
		decimalSeparator: locale.CurrencyDecimalSeparator(),
		groupSeparator:   locale.CurrencyGroupSeparator(),
		groupSizes:       locale.CurrencyGroupSizes()[0],
		negativePattern:  locale.CurrencyNegativePattern(),
		positivePattern:  locale.CurrencyPositivePattern(),
		symbol:           symbol,
	}
}

var zero = decimal.Int(0)
var cleanSymbol = regexp.MustCompile(`\s*\$\s*`)

func (cf *CurrencyFormatter) Format(amount decimal.Dec) string {
	return cf.format(amount, cf.decimalDigits)
}

func (cf *CurrencyFormatter) format(amount decimal.Dec, decimalDigits int) string {

	if decimalDigits == -1 {
		decimalDigits = cf.decimalDigits
	}

	amount = amount.Rounded(decimalDigits, "HalfUp")

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

	if cf.groupSeparator != "" {
		for i := len(integerPart); i > cf.groupSizes; {
			i -= cf.groupSizes
			integerPart = integerPart[:i] + cf.groupSeparator + integerPart[i:]
		}
	}

	decimalPart += strings.Repeat("0", decimalDigits-len(decimalPart))

	var formattedAmount string
	if isNegative {
		formattedAmount = negativePatterns[cf.negativePattern]
	} else {
		formattedAmount = positivePatterns[cf.positivePattern]
	}

	var newAmount = integerPart
	if decimalPart != "" {
		newAmount += cf.decimalSeparator + decimalPart
	}

	formattedAmount = strings.Replace(formattedAmount, "n", newAmount, 1)

	if cf.symbol == "" {
		formattedAmount = cleanSymbol.ReplaceAllString(formattedAmount, "")
	} else {
		formattedAmount = strings.Replace(formattedAmount, "$", cf.symbol, 1)
	}

	return formattedAmount
}
