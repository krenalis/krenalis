// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"slices"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// LastPreviousCheck is a closed comparison of one field on Last and Previous.
// Difference is Last minus Previous; Ratio has a positive Previous denominator.
// Test must be zero for value checks; Upper must be zero outside Between.
type LastPreviousCheck struct {
	Kind LastPreviousCheckKind
	Test SumTest
}

// NormalizedTest returns the exact ordered difference test on readable pairs.
// It returns *UnsupportedError for the other valid canonical checks.
func (check LastPreviousCheck) NormalizedTest() (NormalizedDifferenceTest, error) {
	return check.normalizedTest()
}

// Validate checks the complete requested check before capability admission.
func (check LastPreviousCheck) Validate() error {
	return check.validate()
}

func (check LastPreviousCheck) normalizedTest() (NormalizedDifferenceTest, error) {
	err := check.validate()
	if err != nil {
		return NormalizedDifferenceTest{}, err
	}
	if check.Kind != LastPreviousDifference || check.Test.Operator == Equal ||
		check.Test.Operator == Different || check.Test.Operator == Between {
		return NormalizedDifferenceTest{}, &UnsupportedError{Capability: "lastVersusPrevious check"}
	}
	value, atLeast, err := check.Test.threshold()
	if err != nil {
		return NormalizedDifferenceTest{}, err
	}
	maximum := decimal.MustParse("199999999999999.9998")
	minimum := maximum.Neg()
	mode := ""
	if atLeast {
		if value.Greater(maximum) {
			mode = "none"
		} else if !value.Greater(minimum) {
			mode = "all"
		}
	} else {
		if value.Less(minimum) {
			mode = "none"
		} else if !value.Less(maximum) {
			mode = "all"
		}
	}
	if mode != "" {
		return NormalizedDifferenceTest{ReadableDifferences: mode}, nil
	}
	operator := LessOrEqual
	if atLeast {
		operator = GreaterOrEqual
	}
	return NormalizedDifferenceTest{Comparison: &SumTest{Operator: operator, Value: value}}, nil
}

func (check LastPreviousCheck) validate() error {
	switch check.Kind {
	case LastPreviousSameValue, LastPreviousDifferentValue:
		if check.Test.Operator != 0 || !check.Test.Value.Equal(decimal.Decimal{}) ||
			!check.Test.Upper.Equal(decimal.Decimal{}) {
			return fmt.Errorf("lastVersusPrevious value checks admit only kind")
		}
	case LastPreviousDifference, LastPreviousRatio:
		if check.Test.Operator > Between || !types.IsDecimal(check.Test.Value) || !types.IsDecimal(check.Test.Upper) ||
			check.Test.Operator == Between && check.Test.Upper.Less(check.Test.Value) ||
			check.Test.Operator != Between && !check.Test.Upper.Equal(decimal.Decimal{}) {
			return fmt.Errorf("invalid lastVersusPrevious comparison")
		}
	default:
		return fmt.Errorf("invalid lastVersusPrevious check kind")
	}
	return nil
}

// LastPreviousCheckKind identifies the four canonical pair checks.
type LastPreviousCheckKind uint8

const (
	LastPreviousSameValue LastPreviousCheckKind = iota + 1
	LastPreviousDifferentValue
	LastPreviousDifference
	LastPreviousRatio
)

// LastVersusPrevious reads one field on jointly compatible ordered pairs.
// This subset requires independent memberships, determined identities and
// timestamps, and a base restricted to one governed native currency.
type LastVersusPrevious struct {
	Scope    Scope
	Where    *state.Where
	Currency *NativeCurrency
	Field    *MonetaryField
	Check    LastPreviousCheck
}

func (last *LastVersusPrevious) prepare(at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {
	where, err := RecordPredicate(last.Where)
	if err != nil {
		return nil, err
	}
	population, err := prepareRule(Rule{Count: &Count{Scope: last.Scope, Where: where, Currency: last.Currency,
		Test: CountTest{Operator: GreaterOrEqual, Value: 2}}}, Bindings{}, at, source)
	if err != nil {
		return nil, err
	}
	count, ok := population.(*warehouses.IndependentOccurrenceCount)
	if !ok {
		return nil, fmt.Errorf("core/internal/unified: invalid lastVersusPrevious population")
	}
	if !warehouses.IsValidIdentifier(source.Properties.Name) || !source.Properties.Type.IsJSON() ||
		len(last.Field.amountPath) < 2 || last.Field.amountPath[0] != "properties" {
		return nil, fmt.Errorf("core/internal/unified: invalid lastVersusPrevious reading")
	}
	normalized, err := last.Check.normalizedTest()
	if err != nil {
		return nil, err
	}
	result := &warehouses.LastPreviousDifference{Source: count.Source, Start: count.Start, End: count.End,
		UnboundedStart: count.UnboundedStart, Qualification: count.Qualification,
		Value: warehouses.DecimalReading{Column: source.Properties, Keys: slices.Clone(last.Field.amountPath[1:]),
			Type: last.Field.amountType}, ReadableDifferences: normalized.ReadableDifferences}
	if test := normalized.Comparison; test != nil {
		result.Threshold = &test.Value
		result.AtLeast = test.Operator == GreaterOrEqual
	}
	return result, nil
}

func (last *LastVersusPrevious) validate(at time.Time) error {
	if last == nil || last.Field == nil || last.Field.definition == "" {
		return fmt.Errorf("unresolved lastVersusPrevious field")
	}
	// Validate the source, check and Record shape.
	err := last.Scope.validateSource()
	if err != nil {
		return err
	}
	if last.Scope.eventType() != "track" || last.Scope.Event != "Order Completed" {
		return fmt.Errorf("lastVersusPrevious field is incompatible with the occurrence source")
	}
	err = last.Check.validate()
	if err != nil {
		return err
	}
	if last.Where != nil {
		nodes := 0
		err := validateWhere(last.Where, 1, &nodes)
		if err != nil {
			return err
		}
	}
	// Validate the shared unit before dependency and capability admission.
	if last.Currency != nil {
		if last.Currency.field == nil {
			return fmt.Errorf("unresolved lastVersusPrevious currency field")
		}
		err := last.Currency.validate(last.Scope.Event)
		if err != nil {
			return err
		}
		if !slices.Equal(last.Currency.field.path, last.Field.currencyPath) {
			return fmt.Errorf("lastVersusPrevious field and currency readings are incompatible")
		}
	}
	var unavailable error
	err = last.Field.ValidateWhere(last.Where)
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	_, _, err = last.Scope.bounds(at)
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	if last.Currency == nil {
		unavailable = &UnsupportedError{Capability: "lastVersusPrevious without base currency"}
	}
	_, err = last.Check.normalizedTest()
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			return err
		}
		unavailable = err
	}
	return unavailable
}

// NormalizedDifferenceTest contains an inclusive bound or a readable domain.
// None and All apply only when both selected amounts are readable.
type NormalizedDifferenceTest struct {
	Comparison          *SumTest
	ReadableDifferences string
}
