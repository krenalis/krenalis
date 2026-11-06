// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type explanationTerm struct {
	truth, lower, upper string
}

// BuildExplanation reuses the closed Record/Count expressions and emits one
// bounded row per node for the requested profile, including non-matches.
func BuildExplanation(filter warehouses.Filter, population warehouses.ProfileSource, profile string, dialect Dialect) (Statement, error) {

	// Validate the prepared filter and execution inputs.
	err := warehouses.ValidateProfileFilter(filter, warehouses.ProfileOutcomeRequest{Population: population, Limit: 1})
	if err != nil {
		return Statement{}, err
	}
	if id, ok := types.NormalizeUUID(profile); !ok || id != profile {
		return Statement{}, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation profile")
	}
	if dialect.QuoteIdentifier == nil || dialect.Record == nil || dialect.KnownText == nil || dialect.Parameter == nil {
		return Statement{}, fmt.Errorf("warehouses/internal/profilefilter: missing dialect")
	}

	// Compile all closed outcomes for the target profile.
	c := compiler{dialect: dialect, explain: true}
	c.profile, err = c.parameter(types.String(), profile)
	if err != nil {
		return Statement{}, err
	}
	c.ctes = append(c.ctes, "target AS (SELECT "+c.profile+" AS profile_id)")
	c.profile = "(SELECT profile_id FROM target)"
	if filter.Root == nil {
		c.nodes = []explanationTerm{{truth: "TRUE"}}
	} else {
		_, err = c.rule(filter.Root)
		if err != nil {
			return Statement{}, err
		}
	}

	// Project the node outcomes into bounded protocol rows.
	q := dialect.QuoteIdentifier
	projection := ""
	if len(c.records) > 0 {
		projection = "," + strings.Join(c.records, ",")
	}
	c.ctes = append(c.ctes, "population AS (SELECT CAST("+q(population.Key.Name)+" AS VARCHAR(36)) AS "+
		q("$profile_id")+projection+" FROM "+q(population.Table)+
		" WHERE CAST("+q(population.Key.Name)+" AS VARCHAR(36))="+c.profile+")")
	terms, rows, counts := []string{}, []string{}, make([]bool, len(c.nodes))
	for i, node := range c.nodes {
		n := strconv.Itoa(i)
		terms = append(terms, "("+node.truth+") AS "+q("$truth"+n))
		lo, hi := "CAST(NULL AS BIGINT)", "CAST(NULL AS BIGINT)"
		if node.lower != "" {
			counts[i] = true
			terms = append(terms, node.lower+" AS "+q("$lo"+n), node.upper+" AS "+q("$hi"+n))
			lo, hi = q("$lo"+n), q("$hi"+n)
		}
		rows = append(rows, "SELECT profile_id,"+n+" AS node,"+q("$truth"+n)+" AS truth,"+
			lo+" AS lo,"+hi+" AS hi FROM evaluated")
	}
	c.ctes = append(c.ctes, "evaluated AS (SELECT p."+q("$profile_id")+" AS profile_id,"+
		strings.Join(terms, ",")+" FROM population p "+strings.Join(c.joins, " ")+")")
	failures := "0"
	if len(c.faults) > 0 {
		parts := make([]string, len(c.faults))
		for i, name := range c.faults {
			parts[i] = "(SELECT failures FROM " + name + ")"
		}
		failures = strings.Join(parts, "+")
	}
	c.ctes = append(c.ctes, "explanation AS ("+strings.Join(rows, " UNION ALL ")+")")

	return Statement{SQL: "WITH " + strings.Join(c.ctes, ",\n") +
		"\nSELECT profile_id,node,truth,lo,hi,(" + failures + ") FROM explanation ORDER BY node LIMIT " +
		strconv.Itoa(len(c.nodes)+1), Arguments: c.arguments, CountNodes: counts,
		Columns: []warehouses.Column{
			{Name: "profile_id", Type: types.UUID()}, {Name: "node", Type: types.Int(64)},
			{Name: "truth", Type: types.Boolean(), Nullable: true},
			{Name: "lo", Type: types.Int(64), Nullable: true}, {Name: "hi", Type: types.Int(64), Nullable: true},
			{Name: "source_failures", Type: types.Int(64)},
		}}, nil
}

// ReadExplanation consumes and closes a fixed-width explanation cursor.
// Missing profiles return nil; any failure discards the complete result.
func ReadExplanation(ctx context.Context, rows warehouses.Rows, countNodes []bool, profile string) (result []warehouses.FilterNodeOutcome, err error) {

	if rows == nil {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: missing explanation cursor")
	}
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil && err == nil {
			result, err = nil, closeErr
		}
	}()
	if len(countNodes) < 1 || len(countNodes) > 101 {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation node count")
	}
	if id, ok := types.NormalizeUUID(profile); !ok || id != profile {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation profile")
	}

	for rows.Next() {
		if len(result) >= len(countNodes) {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: explanation row limit exceeded")
		}
		var values [6]any
		err := rows.Scan(values[:]...)
		if err != nil {
			return nil, err
		}
		id, ok := values[0].(string)
		index, indexed := values[1].(int)
		failures, checked := values[5].(int)
		if !ok || id != profile || !indexed || index != len(result) || !checked || failures < 0 {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation row")
		}
		if failures != 0 {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: occurrence source violates the prepared Count contract")
		}
		node := warehouses.FilterNodeOutcome{Truth: warehouses.FilterUnknown}
		if values[2] != nil {
			value, ok := values[2].(bool)
			if !ok {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation truth")
			}
			node.Truth = warehouses.FilterNoMatch
			if value {
				node.Truth = warehouses.FilterMatch
			}
		}
		if countNodes[index] {
			lo, lok := values[3].(int)
			hi, hok := values[4].(int)
			if !lok || !hok || lo < 0 || hi < lo {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid explanation count bounds")
			}
			node.Count = &warehouses.OccurrenceCountBounds{Certain: int64(lo), Possible: int64(hi)}
		} else if values[3] != nil || values[4] != nil {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: unexpected explanation count bounds")
		}
		result = append(result, node)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	err = ctx.Err()
	if err != nil {
		return nil, err
	}
	if len(result) != 0 && len(result) != len(countNodes) {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: incomplete explanation")
	}

	return result, nil
}
