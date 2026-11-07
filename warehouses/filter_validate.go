// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// ValidateProfileFilter checks the prepared Profiles subset and request. Core
// still owns normative admissibility and evidence contracts. Expanded Record
// expressions share a whole-filter budget because object presence may expand
// a single public condition into many physical-column predicates. It returns
// ErrProfileProjectionTooWide when the projection exceeds its column budget.
func ValidateProfileFilter(filter Filter, request ProfileOutcomeRequest) error {
	if request.Limit < 1 || request.Limit > 1000 || request.First < 0 || int64(request.First) > math.MaxInt32 {
		return fmt.Errorf("warehouses: invalid profile outcome page")
	}
	err := request.Population.Validate()
	if err != nil {
		return err
	}
	if len(request.Columns) > MaxProfileProjectionColumns {
		return ErrProfileProjectionTooWide
	}
	if len(request.OrderBy) > 1 {
		return fmt.Errorf("warehouses: profile ordering exceeds limits")
	}
	for _, column := range request.Columns {
		if !IsValidIdentifier(column.Name) || !column.Type.Valid() || column.Type.IsObject() {
			return fmt.Errorf("warehouses: invalid profile projection column")
		}
	}
	for _, column := range request.OrderBy {
		if !IsValidIdentifier(column.Name) || !column.Type.Valid() || column.Type.IsJSON() ||
			column.Type.IsArray() || column.Type.IsMap() || column.Type.IsObject() {
			return fmt.Errorf("warehouses: invalid profile ordering column")
		}
	}
	if filter.Root == nil {
		return nil
	}
	nodes, physical := 0, 0
	return validateFilterCondition(filter.Root, 0, &nodes, &physical)
}

func validateFilterCondition(node FilterCondition, depth int, nodes, physical *int) error {
	*nodes++
	if *nodes > 101 {
		return fmt.Errorf("warehouses: prepared filter exceeds node budget")
	}
	switch n := node.(type) {
	case *FilterGroup:
		if n == nil || depth >= 4 || (n.Operator != OpAnd && n.Operator != OpOr) || len(n.Children) == 0 {
			return fmt.Errorf("warehouses: invalid prepared filter group")
		}
		for _, child := range n.Children {
			err := validateFilterCondition(child, depth+1, nodes, physical)
			if err != nil {
				return err
			}
		}
	case *RecordFilter:
		if n == nil {
			return fmt.Errorf("warehouses: nil profile Record condition")
		}
		return validateRecordQualification(n.Expr, 1, physical)
	case *DeterminedOccurrenceAverage:
		err := n.validate()
		if err != nil {
			return err
		}
		if n.Qualification != nil {
			return validateQualification(n.Qualification, 1, physical)
		}
	case *IndependentOccurrenceSum:
		err := n.validate()
		if err != nil {
			return err
		}
		if n.Qualification != nil {
			return validateQualification(n.Qualification, 1, physical)
		}
	case *IndependentOccurrenceCount:
		if n == nil {
			return fmt.Errorf("warehouses: nil prepared Count")
		}
		if n.Minimum < 0 {
			return fmt.Errorf("warehouses: invalid prepared sample minimum")
		}
		err := n.Source.Validate()
		if err != nil {
			return err
		}
		bounds := []time.Time{n.End}
		if n.UnboundedStart {
			if !n.Start.IsZero() {
				return fmt.Errorf("warehouses: unbounded prepared period cannot have a start")
			}
		} else {
			bounds = append(bounds, n.Start)
		}
		for _, bound := range bounds {
			utc := bound.UTC()
			if utc.IsZero() || utc.Year() < 1 || utc.Year() > 9999 || utc.Nanosecond()%1000 != 0 {
				return fmt.Errorf("warehouses: invalid prepared timestamp")
			}
		}
		if !n.UnboundedStart && !n.Start.Before(n.End) {
			return fmt.Errorf("warehouses: invalid prepared period")
		}
		if len(n.Accept) > 2 {
			return fmt.Errorf("warehouses: too many accepted Count ranges")
		}
		for i, r := range n.Accept {
			if r.Lower < 0 || r.Upper != nil && *r.Upper < r.Lower {
				return fmt.Errorf("warehouses: invalid accepted Count range")
			}
			if i > 0 {
				previous := n.Accept[i-1].Upper
				if previous == nil || *previous == math.MaxInt64 || r.Lower <= *previous+1 {
					return fmt.Errorf("warehouses: Count ranges must be normalized")
				}
			}
		}
		if n.Qualification != nil {
			return validateQualification(n.Qualification, 1, physical)
		}
	default:
		return fmt.Errorf("warehouses: unexpected prepared filter node %T", node)
	}
	return nil
}

func validateQualification(node RowQualification, depth int, nodes *int) error {
	*nodes++
	if depth > 3 || *nodes > 16384 {
		return fmt.Errorf("warehouses: qualification exceeds structural budget")
	}
	switch n := node.(type) {
	case *QualificationGroup:
		if n == nil || len(n.Children) == 0 || (n.Operator != OpAnd && n.Operator != OpOr) {
			return fmt.Errorf("warehouses: invalid qualification group")
		}
		for _, child := range n.Children {
			err := validateQualification(child, depth+1, nodes)
			if err != nil {
				return err
			}
		}
	case *RecordQualification:
		if n == nil {
			return fmt.Errorf("warehouses: nil Record qualification")
		}
		return validateRecordQualification(n.Expr, 1, nodes)
	case *KnownTextEqual:
		if n == nil || !IsValidIdentifier(n.Column.Name) || !n.Column.Type.IsJSON() || len(n.Keys) > 32 ||
			len(n.Value) > 4096 || !utf8.ValidString(n.Value) || strings.ContainsRune(n.Value, 0) {
			return fmt.Errorf("warehouses: invalid known-text qualification")
		}
		for _, key := range n.Keys {
			if len(key) > 1024 || !utf8.ValidString(key) || strings.ContainsRune(key, 0) {
				return fmt.Errorf("warehouses: invalid known-text key")
			}
		}
	default:
		return fmt.Errorf("warehouses: unexpected qualification %T", node)
	}
	return nil
}

func validateRecordQualification(expr Expr, depth int, nodes *int) error {
	*nodes++
	if depth > 4 || *nodes > 16384 {
		return fmt.Errorf("warehouses: Record qualification exceeds structural budget")
	}
	switch n := expr.(type) {
	case *MultiExpr:
		if n == nil || len(n.Operands) == 0 || (n.Operator != OpAnd && n.Operator != OpOr) {
			return fmt.Errorf("warehouses: invalid Record qualification group")
		}
		for _, operand := range n.Operands {
			err := validateRecordQualification(operand, depth+1, nodes)
			if err != nil {
				return err
			}
		}
	case *BaseExpr:
		if n == nil || !IsValidIdentifier(n.Column.Name) || !n.Column.Type.Valid() || n.Operator < OpIs ||
			n.Operator > OpDoesNotExist {
			return fmt.Errorf("warehouses: invalid prepared Record qualification")
		}
		for _, value := range n.Values {
			if _, ok := value.(Column); ok {
				return fmt.Errorf("warehouses: Record qualification requires literal operands")
			}
		}
	default:
		return fmt.Errorf("warehouses: unexpected Record expression %T", expr)
	}
	return nil
}
