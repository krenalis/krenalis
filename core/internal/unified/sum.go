// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// MonetaryField is a resolved reading of a reported monetary event value.
// Its reading precision does not attest a domain for unknown historical values.
type MonetaryField struct {
	definition, revision string
}

// ResolveMonetaryField resolves the Order Completed reported total, revision 1.
// Known values are JSON Numbers exactly representable as Decimal(18,4).
func ResolveMonetaryField(definition, revision string) (*MonetaryField, error) {
	if definition != "commerce.order_completed.total" || revision != "1" {
		return nil, fmt.Errorf("unresolved monetary field meaning revision")
	}
	return &MonetaryField{definition: definition, revision: revision}, nil
}

// Reference returns the resolved definition and immutable meaning revision.
func (field *MonetaryField) Reference() (definition, revision string) {
	if field == nil || field.definition == "" {
		panic("core/internal/unified: unresolved monetary field")
	}
	return field.definition, field.revision
}

// Sum adds independently qualified recorded messages in one native currency.
// Unknown amounts have no certified sign or finite business-domain bounds.
type Sum struct {
	Scope          Scope
	Field          *MonetaryField
	NativeCurrency string
	Test           SumTest
}

func (sum *Sum) prepare(at time.Time, source SourceBinding) (warehouses.FilterCondition, error) {

	if source.Record == nil {
		return nil, fmt.Errorf("core/internal/unified: missing occurrence Record binder")
	}
	err := source.Occurrences.Validate()
	if err != nil {
		return nil, err
	}
	if !warehouses.IsValidIdentifier(source.Properties.Name) || !source.Properties.Type.IsJSON() {
		return nil, fmt.Errorf("core/internal/unified: invalid resolved monetary column")
	}

	selector, err := source.Record(&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs, Values: []any{sum.Scope.Event}},
		&state.WhereCondition{Property: []string{"type"}, Operator: state.OpIs, Values: []any{"track"}},
	}})
	if err != nil {
		return nil, err
	}
	start, end, err := sum.Scope.bounds(at)
	if err != nil {
		return nil, err
	}
	threshold, atLeast, err := sum.Test.threshold()
	if err != nil {
		return nil, err
	}

	return &warehouses.IndependentOccurrenceSum{
		Source: source.Occurrences, Start: start, End: end, UnboundedStart: sum.Scope.History != nil,
		Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: []warehouses.RowQualification{
			&warehouses.RecordQualification{Expr: selector},
			&warehouses.KnownTextEqual{Column: source.Properties, Keys: []string{"currency"}, Value: sum.NativeCurrency},
		}},
		Value:     warehouses.DecimalReading{Column: source.Properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
		Threshold: threshold, AtLeast: atLeast,
	}, nil
}

func (sum *Sum) validate(at time.Time) error {
	if sum == nil || sum.Field == nil || sum.Field.definition == "" {
		return fmt.Errorf("unresolved monetary field")
	}
	if sum.Scope.Event != "Order Completed" || sum.Scope.Grain != "registered-message" {
		return fmt.Errorf("monetary field is incompatible with the occurrence source")
	}
	if len(sum.NativeCurrency) != 3 {
		return fmt.Errorf("nativeCurrency requires three uppercase letters")
	}
	for _, c := range sum.NativeCurrency {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("nativeCurrency requires three uppercase letters")
		}
	}
	if sum.Test.Operator > Between || !types.IsDecimal(sum.Test.Value) ||
		!types.IsDecimal(sum.Test.Upper) || sum.Test.Operator == Between && sum.Test.Upper.Less(sum.Test.Value) {
		return fmt.Errorf("invalid Sum terminal")
	}
	_, _, err := sum.Scope.bounds(at)
	if err != nil {
		return err
	}
	if sum.Test.Operator == Equal || sum.Test.Operator == Different || sum.Test.Operator == Between {
		return &UnsupportedError{Capability: "Sum equality or interval reachability"}
	}
	return nil
}

// SumTest compares an exact monetary total with an exact signed literal.
type SumTest struct {
	Operator     Comparison
	Value, Upper decimal.Decimal
}

// threshold normalizes an ordered test to an inclusive bound on the 1e-4 grid.
// With at most MaxInt64 occurrences of Decimal(18,4), every finite total lies
// strictly between -1e33 and 1e33. Clipping the normalized threshold there is
// exact, including for unbounded unknown amounts. Overflow remains a fault.
func (test SumTest) threshold() (decimal.Decimal, bool, error) {

	_, scale := test.Value.PrecisionScale()
	rational, ok := new(big.Rat).SetString(fmt.Sprintf("%.*f", scale, test.Value))
	if !ok {
		return decimal.Decimal{}, false, fmt.Errorf("core/internal/unified: invalid prepared Sum literal")
	}
	numerator := new(big.Int).Mul(rational.Num(), big.NewInt(10000))
	coefficient, remainder := new(big.Int), new(big.Int)
	coefficient.QuoRem(numerator, rational.Denom(), remainder)
	atLeast := test.Operator == Greater || test.Operator == GreaterOrEqual
	switch test.Operator {
	case Greater, LessOrEqual:
		if remainder.Sign() < 0 {
			coefficient.Sub(coefficient, big.NewInt(1))
		}
		if test.Operator == Greater {
			coefficient.Add(coefficient, big.NewInt(1))
		}
	case GreaterOrEqual, Less:
		if remainder.Sign() > 0 {
			coefficient.Add(coefficient, big.NewInt(1))
		}
		if test.Operator == Less {
			coefficient.Sub(coefficient, big.NewInt(1))
		}
	default:
		return decimal.Decimal{}, false, fmt.Errorf("core/internal/unified: unsupported prepared Sum comparison")
	}

	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(37), nil)
	if coefficient.Cmp(limit) > 0 {
		coefficient.Set(limit)
	} else if negative := new(big.Int).Neg(limit); coefficient.Cmp(negative) < 0 {
		coefficient.Set(negative)
	}

	negative := coefficient.Sign() < 0
	text := new(big.Int).Abs(coefficient).String()
	if len(text) < 5 {
		text = strings.Repeat("0", 5-len(text)) + text
	}
	text = text[:len(text)-4] + "." + text[len(text)-4:]
	if negative {
		text = "-" + text
	}
	value, err := decimal.Parse(text, 38, 4)

	return value, atLeast, err
}
