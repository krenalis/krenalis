// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/core/internal/state"
)

// NativeCurrency binds a resolved occurrence-local currency definition.
// Missing, JSON null and non-text values are unknown business currency.
// The definition must guarantee independent unknown values per occurrence.
type NativeCurrency struct {
	Definition string
	Revision   string
	Value      string
	field      *currencyField
}

// ResolveNativeCurrency resolves a literal predicate for the given event.
// Only commerce.order_completed.currency, meaning revision 1, is available.
func ResolveNativeCurrency(definition, revision, value, event string) (*NativeCurrency, error) {
	field, err := resolveCurrencyField(definition, revision)
	if err != nil {
		return nil, err
	}
	currency := &NativeCurrency{Definition: definition, Revision: revision, Value: value, field: field}
	err = currency.validate(event)
	if err != nil {
		return nil, err
	}
	return currency, nil
}

// ValidateWhere checks qualification independence from this currency reading.
// It returns *UnsupportedError for overlapping paths or unbound readings.
func (currency *NativeCurrency) ValidateWhere(where *state.Where) error {
	if currency == nil {
		return fmt.Errorf("invalid resolved NativeCurrency predicate")
	}
	if where == nil {
		return nil
	}
	if currency.field == nil {
		nodes := 0
		err := validateWhere(where, 1, &nodes)
		if err != nil {
			return err
		}
		return &UnsupportedError{Capability: "mixed governed and raw occurrence predicates"}
	}
	return validateIndependentWhere(where, [][]string{currency.field.path},
		"Count qualification dependent on currency")
}

func (currency *NativeCurrency) validate(event string) error {
	if currency == nil || currency.Definition == "" || currency.Revision == "" ||
		len(currency.Definition) > 256 || len(currency.Revision) > 256 || len(currency.Value) != 3 {
		return fmt.Errorf("invalid resolved NativeCurrency predicate")
	}
	for _, c := range currency.Value {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("invalid currency literal")
		}
	}
	if field := currency.field; field != nil {
		if currency.Definition != field.definition || currency.Revision != field.revision {
			return fmt.Errorf("invalid resolved currency field reference")
		}
		if event != field.event {
			return fmt.Errorf("currency field is incompatible with the occurrence source")
		}
		if len(field.path) < 2 || len(field.path) > 32 || field.path[0] != "properties" {
			return fmt.Errorf("core/internal/unified: invalid resolved currency field path")
		}
	}
	return nil
}

type currencyField struct {
	definition, revision, event string
	path                        []string
}

func resolveCurrencyField(definition, revision string) (*currencyField, error) {
	if definition != "commerce.order_completed.currency" || revision != "1" {
		return nil, fmt.Errorf("unresolved currency field meaning revision")
	}
	return &currencyField{definition: definition, revision: revision, event: "Order Completed",
		path: []string{"properties", "currency"}}, nil
}

// validateIndependentWhere excludes paths overlapping a governed reading.
func validateIndependentWhere(where *state.Where, paths [][]string, capability string) error {

	if where == nil {
		return nil
	}
	nodes := 0
	err := validateWhere(where, 1, &nodes)
	if err != nil {
		return err
	}

	stack := []*state.Where{where}
	for len(stack) > 0 {
		group := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, rule := range group.Rules {
			switch rule := rule.(type) {
			case *state.Where:
				stack = append(stack, rule)
			case *state.WhereCondition:
				for _, path := range paths {
					n := min(len(rule.Property), len(path))
					if slices.Equal(rule.Property[:n], path[:n]) {
						return &UnsupportedError{Capability: capability}
					}
				}
			default:
				return fmt.Errorf("core/internal/unified: unexpected prepared Record node %T", rule)
			}
		}
	}

	return nil
}
