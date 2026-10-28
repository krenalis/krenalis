// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/types"
)

// unifiedCount is a programmatic Count request with a public Record predicate.
type unifiedCount struct {
	Scope unified.Scope
	Where *Filter
	Test  unified.CountTest
}

// unifiedQuery is an internal service input without a public or stored format.
type unifiedQuery struct {
	Root *unifiedRule
	At   time.Time
	Page unified.Page
}

// unifiedRule is a profile Record condition or group, Count or AND/OR group.
// Only explicit groups contribute a level and a node to the common tree.
type unifiedRule struct {
	Profile  FilterRule
	Count    *unifiedCount
	Operator FilterLogical
	Rules    []unifiedRule
}

func (this *Workspace) unifiedProfiles(ctx context.Context, query unifiedQuery) (*unified.Result, error) {

	this.core.mustBeOpen()

	var root *unified.Rule
	if query.Root != nil {
		nodes := 0
		prepared, err := prepareUnifiedRule(*query.Root, this.workspace.ProfileSchema, 0, &nodes)
		if err != nil {
			return nil, err
		}
		root = &prepared
	}

	return this.store.UnifiedProfiles(ctx, root, unified.Bindings{}, query.At, query.Page)
}

// convertWhereToUnified preserves the already validated Record boolean tree.
func convertWhereToUnified(where *state.Where) unified.Rule {
	result := unified.Rule{Operator: where.Operator, Children: make([]unified.Rule, len(where.Rules))}
	for i, rule := range where.Rules {
		switch rule := rule.(type) {
		case *state.Where:
			result.Children[i] = convertWhereToUnified(rule)
		case *state.WhereCondition:
			result.Children[i] = unified.Rule{Record: rule}
		default:
			panic(fmt.Sprintf("core: unexpected prepared Record rule %T", rule))
		}
	}
	return result
}

// prepareUnifiedRecordCount admits a Count with the existing public Record
// filter on its occurrences. It is an internal entry, not a new FilterRule wire
// tag. The store validates the remaining canonical scope and terminal.
func prepareUnifiedRecordCount(scope unified.Scope, filter *Filter, test unified.CountTest) (unified.Rule, error) {
	where, _, err := prepareFilter(filter, schemas.Event)
	if err != nil {
		return unified.Rule{}, err
	}
	return unified.Rule{Count: &unified.Count{Scope: scope, Where: where, Test: test}}, nil
}

func prepareUnifiedRule(rule unifiedRule, schema types.Type, depth int, nodes *int) (unified.Rule, error) {
	*nodes++
	if *nodes > 101 {
		return unified.Rule{}, fmt.Errorf("internal Unified request exceeds node budget")
	}
	if rule.Profile != nil {
		if rule.Count != nil || rule.Operator != "" || len(rule.Rules) != 0 {
			return unified.Rule{}, fmt.Errorf("profile Record input cannot contain other rules")
		}
		filter, group := rule.Profile.(*Filter)
		if group && filter == nil {
			return unified.Rule{}, fmt.Errorf("profile Record group cannot be nil")
		}
		if !group {
			filter = &Filter{Operator: OpAnd, Rules: []FilterRule{rule.Profile}}
		}
		where, _, err := prepareFilter(filter, schema)
		if err != nil {
			return unified.Rule{}, err
		}
		prepared := convertWhereToUnified(where)
		if !group {
			// The wrapper validates one condition; it is not an input group.
			return prepared.Children[0], nil
		}
		return prepared, nil
	}
	if rule.Count != nil {
		if rule.Operator != "" || len(rule.Rules) != 0 {
			return unified.Rule{}, fmt.Errorf("Count input cannot contain other rules")
		}
		return prepareUnifiedRecordCount(rule.Count.Scope, rule.Count.Where, rule.Count.Test)
	}
	if depth >= 4 || (rule.Operator != OpAnd && rule.Operator != OpOr) || len(rule.Rules) == 0 ||
		len(rule.Rules) > 100 {
		return unified.Rule{}, fmt.Errorf("invalid internal Unified group")
	}
	result := unified.Rule{Operator: convertLogicalToWhere(rule.Operator),
		Children: make([]unified.Rule, len(rule.Rules))}
	for i, child := range rule.Rules {
		prepared, err := prepareUnifiedRule(child, schema, depth+1, nodes)
		if err != nil {
			return unified.Rule{}, err
		}
		result.Children[i] = prepared
	}
	return result, nil
}
