// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package profilefilter implements the shared physical Profiles filter path.
package profilefilter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type compiler struct {
	dialect   Dialect
	ctes      []string
	joins     []string
	faults    []string
	records   []string
	arguments []Argument
	counts    int
	explain   bool
	profile   string
	nodes     []explanationTerm
}

func (c *compiler) condition(node warehouses.FilterCondition) (string, error) {
	switch n := node.(type) {
	case *warehouses.RecordFilter:
		text, err := c.dialect.Record(n.Expr)
		if err != nil {
			return "", err
		}
		alias := c.dialect.QuoteIdentifier("$record" + strconv.Itoa(len(c.records)))
		c.records = append(c.records, "("+text+") AS "+alias)
		return "p." + alias, nil
	case *warehouses.IndependentOccurrenceCount:
		return c.count(n)
	case *warehouses.IndependentOccurrenceSum:
		return c.sum(n)
	case *warehouses.FilterGroup:
		parts := make([]string, len(n.Children))
		for i, child := range n.Children {
			text, err := c.rule(child)
			if err != nil {
				return "", err
			}
			parts[i] = "(" + text + ")"
		}
		operator := " AND "
		if n.Operator == warehouses.OpOr {
			operator = " OR "
		}
		return strings.Join(parts, operator), nil
	default:
		return "", fmt.Errorf("warehouses/internal/profilefilter: unexpected condition %T", node)
	}
}

func (c *compiler) count(node *warehouses.IndependentOccurrenceCount) (string, error) {

	n := strconv.Itoa(c.counts)
	c.counts++
	q := c.dialect.QuoteIdentifier
	lower := ""
	if !node.UnboundedStart {
		start, err := c.parameter(types.DateTime(), node.Start)
		if err != nil {
			return "", err
		}
		lower = start + " AS lower_bound,"
	}
	end, err := c.parameter(types.DateTime(), node.End)
	if err != nil {
		return "", err
	}
	qual := "TRUE"
	if node.Qualification != nil {
		qual, err = c.qualification(node.Qualification)
		if err != nil {
			return "", err
		}
	}
	c.ctes = append(c.ctes, "bounds"+n+" AS (SELECT "+lower+end+" AS upper_bound)")
	within := q(node.Source.Timestamp.Name) + "<(SELECT upper_bound FROM bounds" + n + ")"
	if !node.UnboundedStart {
		within = q(node.Source.Timestamp.Name) + ">=(SELECT lower_bound FROM bounds" + n + ") AND " + within
	}
	table, id, profile := q(node.Source.Table), q(node.Source.Identity.Name), q(node.Source.Profile.Name)

	// Seed identities by period, then inspect all copies, including those outside
	// it. Grouping by (profile,identity) or filtering the copies first would hide
	// attribution and period conflicts. Qualification is consumed only inside.
	profileOnly, identityProfile := "", ""
	if c.profile != "" {
		profileOnly = " AND CAST(" + profile + " AS VARCHAR(36))=" + c.profile
		identityProfile = " AND " + id + " IN (SELECT " + id + " FROM " + table +
			" WHERE CAST(" + profile + " AS VARCHAR(36))=" + c.profile + ")"
	}
	c.ctes = append(c.ctes, "ids"+n+" AS (SELECT DISTINCT "+id+" AS occurrence_id FROM "+table+
		" WHERE "+within+identityProfile+")")
	c.ctes = append(c.ctes, "copies"+n+" AS (SELECT "+profile+" AS profile_id,"+id+" AS occurrence_id,"+
		"CASE WHEN "+within+" THEN 1 ELSE 0 END AS in_period,"+
		"CASE WHEN "+within+" THEN ("+qual+") ELSE CAST(NULL AS BOOLEAN) END AS qualification FROM "+table+
		" WHERE "+id+" IN (SELECT occurrence_id FROM ids"+n+"))")
	state := "CASE WHEN qualification=TRUE THEN 1 WHEN qualification=FALSE THEN 0 ELSE 2 END"
	c.ctes = append(c.ctes, "facts"+n+` AS (SELECT occurrence_id,
 MIN(CAST(profile_id AS VARCHAR(36))) AS profile_id,
 COUNT(DISTINCT profile_id)+CASE WHEN COUNT(*)>COUNT(profile_id) THEN 1 ELSE 0 END AS profiles,
 MIN(in_period) AS period_min,MAX(in_period) AS period_max,
 MIN(CASE WHEN in_period=1 THEN `+state+` END) AS state_min,
 MAX(CASE WHEN in_period=1 THEN `+state+` END) AS state_max
 FROM copies`+n+" GROUP BY occurrence_id)")
	c.ctes = append(c.ctes, "cnt"+n+` AS (SELECT profile_id,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS lo,
 SUM(CASE WHEN state_min=0 THEN 0 ELSE 1 END) AS hi
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=period_max
 AND state_min=state_max GROUP BY profile_id)`)

	// All-unattributed identities are outside the profile view. A NULL/non-NULL
	// mixture violates determined attribution. Counters have an explicit signed
	// 64-bit transport domain even on backends whose SUM uses a wider number.
	c.ctes = append(c.ctes, "fault"+n+` AS (SELECT
 (SELECT COALESCE(SUM(CASE WHEN profile_id IS NOT NULL AND
 (profiles>1 OR period_min<>period_max OR state_min<>state_max) THEN 1 ELSE 0 END),0) FROM facts`+n+`)
 +(SELECT COUNT(*) FROM cnt`+n+` WHERE lo>9223372036854775807 OR hi>9223372036854775807)
 +(SELECT COUNT(*) FROM `+table+" WHERE "+within+profileOnly+
		" AND "+profile+" IS NOT NULL AND "+id+" IS NULL) AS failures)")
	c.faults = append(c.faults, "fault"+n)
	alias := "v" + n
	c.joins = append(c.joins, "LEFT JOIN cnt"+n+" "+alias+" ON "+alias+".profile_id=p."+q("$profile_id"))

	lo, hi := "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)"
	return withSampleMinimum(node.Minimum, lo, hi, decision(node.Accept, lo, hi)), nil
}

func (c *compiler) parameter(typ types.Type, value any) (string, error) {
	index := len(c.arguments) + 1
	text, err := c.dialect.Parameter(index, typ)
	if err != nil {
		return "", err
	}
	c.arguments = append(c.arguments, Argument{Type: typ, Value: value})
	return text, nil
}

func (c *compiler) qualification(node warehouses.RowQualification) (string, error) {
	switch n := node.(type) {
	case *warehouses.RecordQualification:
		return c.dialect.Record(n.Expr)
	case *warehouses.KnownTextEqual:
		value, err := c.dialect.KnownText(n.Column, n.Keys)
		if err != nil {
			return "", err
		}
		p, err := c.parameter(types.String(), n.Value)
		if err != nil {
			return "", err
		}
		return "(" + value + ")=" + p, nil
	case *warehouses.QualificationGroup:
		parts := make([]string, len(n.Children))
		for i, child := range n.Children {
			text, err := c.qualification(child)
			if err != nil {
				return "", err
			}
			parts[i] = "(" + text + ")"
		}
		operator := " AND "
		if n.Operator == warehouses.OpOr {
			operator = " OR "
		}
		return strings.Join(parts, operator), nil
	default:
		return "", fmt.Errorf("warehouses/internal/profilefilter: unexpected qualification %T", node)
	}
}

func (c *compiler) rule(node warehouses.FilterCondition) (string, error) {
	if !c.explain {
		return c.condition(node)
	}
	index, count := len(c.nodes), c.counts
	c.nodes = append(c.nodes, explanationTerm{})
	text, err := c.condition(node)
	if err != nil {
		return "", err
	}
	term := explanationTerm{truth: text}
	if _, ok := node.(*warehouses.IndependentOccurrenceCount); ok {
		alias := "v" + strconv.Itoa(count)
		term.lower, term.upper = "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)"
	}
	if _, ok := node.(*warehouses.IndependentOccurrenceSum); ok {
		alias := "v" + strconv.Itoa(count)
		term.lower, term.upper = "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)"
		term.sumLower, term.sumUpper = sumBounds(alias)
	}
	c.nodes[index] = term
	return text, nil
}

// Build renders the prepared subset into one statement for the page and totals.
// It never resolves business definitions, chooses an evidence model or invents
// correlations. Dialect and protocol details remain below the warehouse API.
func Build(filter warehouses.Filter, request warehouses.ProfileOutcomeRequest, dialect Dialect) (Statement, error) {
	err := warehouses.ValidateProfileFilter(filter, request)
	if err != nil {
		return Statement{}, err
	}
	if dialect.QuoteIdentifier == nil || dialect.Record == nil || dialect.KnownText == nil || dialect.Parameter == nil {
		return Statement{}, fmt.Errorf("warehouses/internal/profilefilter: missing dialect")
	}
	c := compiler{dialect: dialect}
	truth := "TRUE"
	if filter.Root != nil {
		truth, err = c.rule(filter.Root)
		if err != nil {
			return Statement{}, err
		}
	}
	q := dialect.QuoteIdentifier
	projection := ""
	if len(c.records) > 0 {
		projection = "," + strings.Join(c.records, ",")
	}
	projected, evaluated, selected := "", "", ""
	for i, column := range request.Columns {
		alias := q("$value" + strconv.Itoa(i))
		projected += "," + q(column.Name) + " AS " + alias
		evaluated += ",p." + alias
		selected += "," + alias
	}
	order := "profile_id"
	if len(request.OrderBy) > 0 {
		projected += "," + q(request.OrderBy[0].Name) + " AS " + q("$sort")
		evaluated += ",p." + q("$sort")
		selected += "," + q("$sort")
		order = q("$sort")
		if request.OrderDesc {
			order += " DESC"
		}
		order += " NULLS LAST,profile_id"
	}
	c.ctes = append(c.ctes,
		"population AS (SELECT CAST("+q(request.Population.Key.Name)+" AS VARCHAR(36)) AS "+
			q("$profile_id")+projection+projected+" FROM "+q(request.Population.Table)+")")
	c.ctes = append(c.ctes,
		"evaluated AS (SELECT p."+q("$profile_id")+" AS profile_id,("+truth+") AS truth"+evaluated+
			" FROM population p "+strings.Join(c.joins, " ")+")")
	c.ctes = append(c.ctes, `totals AS (SELECT
 COALESCE(SUM(CASE WHEN truth=TRUE THEN 1 ELSE 0 END),0) AS m,
 COALESCE(SUM(CASE WHEN truth=FALSE THEN 1 ELSE 0 END),0) AS f,
 COALESCE(SUM(CASE WHEN truth IS NULL THEN 1 ELSE 0 END),0) AS u FROM evaluated)`)
	faults := "0 AS failures"
	if len(c.faults) > 0 {
		terms := make([]string, len(c.faults))
		for i, name := range c.faults {
			terms[i] = name + ".failures"
		}
		faults = strings.Join(terms, "+") + " AS failures FROM " + strings.Join(c.faults, " CROSS JOIN ")
	}
	c.ctes = append(c.ctes, "failures AS (SELECT "+faults+")")
	selection := ""
	if request.MatchingOnly {
		selection = " WHERE truth=TRUE"
	}
	c.ctes = append(c.ctes, "outcome_page AS (SELECT profile_id,truth,TRUE AS is_profile"+selected+
		" FROM evaluated"+selection+" ORDER BY "+order+" LIMIT "+strconv.Itoa(request.Limit)+
		" OFFSET "+strconv.Itoa(request.First)+")")
	fields, size := "", "0"
	if len(request.Columns) > 0 && dialect.ValueSize == nil {
		return Statement{}, fmt.Errorf("warehouses/internal/profilefilter: missing projection dialect")
	}
	for i := range request.Columns {
		ref := "b." + q("$value"+strconv.Itoa(i))
		size += "+COALESCE(" + dialect.ValueSize(ref) + ",0)"
	}
	c.ctes = append(c.ctes, "page_size AS (SELECT COALESCE(SUM("+size+
		"),0) AS bytes FROM outcome_page b)")
	for i := range request.Columns {
		ref := "b." + q("$value"+strconv.Itoa(i))
		fields += ",CASE WHEN z.bytes<=" + strconv.Itoa(warehouses.MaxProfilePageBytes) +
			" THEN " + ref + " ELSE NULL END"
	}
	finalOrder := "b.profile_id"
	if len(request.OrderBy) > 0 {
		finalOrder = "b." + q("$sort")
		if request.OrderDesc {
			finalOrder += " DESC"
		}
		finalOrder += " NULLS LAST,b.profile_id"
	}
	result := Statement{
		SQL: "WITH " + strings.Join(c.ctes, ",\n") +
			"\nSELECT b.profile_id,b.truth,COALESCE(b.is_profile,FALSE),a.m,a.f,a.u," +
			"e.failures,(z.bytes>" + strconv.Itoa(warehouses.MaxProfilePageBytes) + ")" + fields +
			" FROM totals a CROSS JOIN failures e CROSS JOIN page_size z LEFT JOIN outcome_page b ON TRUE" +
			" ORDER BY " + finalOrder,
		Arguments: c.arguments,
		Columns: []warehouses.Column{
			{Name: "profile_id", Type: types.UUID(), Nullable: true},
			{Name: "truth", Type: types.Boolean(), Nullable: true},
			{Name: "is_profile", Type: types.Boolean()},
			{Name: "match_count", Type: types.Int(64)},
			{Name: "no_match_count", Type: types.Int(64)},
			{Name: "unknown_count", Type: types.Int(64)},
			{Name: "source_failures", Type: types.Int(64)},
			{Name: "payload_exceeded", Type: types.Boolean()},
		},
	}
	for _, column := range request.Columns {
		column.Nullable = true
		result.Columns = append(result.Columns, column)
	}
	return result, nil
}

func decision(ranges []warehouses.CountRange, lo, hi string) string {
	if len(ranges) == 0 {
		return "FALSE"
	}
	yes, no := make([]string, len(ranges)), make([]string, len(ranges))
	for i, r := range ranges {
		low := strconv.FormatInt(r.Lower, 10)
		y, n := lo+">="+low, hi+"<"+low
		if r.Upper != nil {
			high := strconv.FormatInt(*r.Upper, 10)
			y += " AND " + hi + "<=" + high
			n += " OR " + lo + ">" + high
		}
		yes[i], no[i] = "("+y+")", "("+n+")"
	}
	return "CASE WHEN (" + strings.Join(yes, " OR ") + ") THEN TRUE WHEN (" + strings.Join(no,
		" AND ") + ") THEN FALSE ELSE CAST(NULL AS BOOLEAN) END"
}

// withSampleMinimum composes the sample guard and test using Kleene AND.
func withSampleMinimum(minimum int64, lo, hi, test string) string {
	if minimum == 0 {
		return test
	}
	guard := decision([]warehouses.CountRange{{Lower: minimum}}, lo, hi)
	return "(" + guard + ") AND (" + test + ")"
}
