// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// ProfileLastPreviousCheckExplanation retains the requested difference test.
// Numeric JSON uses the same exact decimal representation as monetary tests.
type ProfileLastPreviousCheckExplanation struct {
	Kind       string                                `json:"kind"`
	Requested  ProfileMoneyComparison                `json:"requested"`
	Normalized ProfileNormalizedDifferenceComparison `json:"normalized"`
}

// ProfileLastVersusPreviousExplanation describes an ordered pair comparison.
// Candidates counts message identities; Unique concerns the ordered pair.
// Field and native currency describe reported amounts, without conversion.
type ProfileLastVersusPreviousExplanation struct {
	Start          time.Time                           `json:"start,omitzero"`
	End            time.Time                           `json:"end"`
	UnboundedStart bool                                `json:"unboundedStart,omitzero"`
	History        *FilterHistory                      `json:"history,omitzero"`
	Field          FilterSumField                      `json:"field"`
	CurrencyField  FilterCurrencyField                 `json:"currencyField"`
	NativeCurrency string                              `json:"nativeCurrency"`
	Check          ProfileLastPreviousCheckExplanation `json:"check"`
	Selection      ProfileExperienceSelection          `json:"selection"`
}

// ProfileNormalizedDifferenceComparison is a bound or readableDifferences.
// None/all concern only pairs with two readable amounts. A possible pair with
// an unreadable amount remains UNKNOWN, and a missing pair never satisfies All.
type ProfileNormalizedDifferenceComparison struct {
	Operator            FilterOperator    `json:"operator,omitzero"`
	Values              []decimal.Decimal `json:"values,omitzero"`
	ReadableDifferences string            `json:"readableDifferences,omitzero"`
}

// MarshalJSON encodes exactly one closed normalized difference form.
func (comparison ProfileNormalizedDifferenceComparison) MarshalJSON() ([]byte, error) {
	if comparison.ReadableDifferences != "" {
		if comparison.ReadableDifferences != "none" && comparison.ReadableDifferences != "all" ||
			comparison.Operator != "" || comparison.Values != nil {
			return nil, errors.New("core: invalid normalized difference domain")
		}
	} else {
		if comparison.Operator != OpIsGreaterThanOrEqualTo && comparison.Operator != OpIsLessThanOrEqualTo ||
			len(comparison.Values) != 1 {
			return nil, errors.New("core: invalid normalized difference comparison")
		}
		value, limit := comparison.Values[0], decimal.MustParse("199999999999999.9998")
		_, scale := value.PrecisionScale()
		if !types.IsDecimal(value) || scale > 4 || value.Greater(limit) || value.Less(limit.Neg()) {
			return nil, errors.New("core: invalid normalized difference value")
		}
	}
	type encoded ProfileNormalizedDifferenceComparison
	return json.Marshal(encoded(comparison))
}
