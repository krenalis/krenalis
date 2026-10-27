// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type compiler struct {
	dialect    warehouses.QueryDialect
	bind       func(*state.Where) (warehouses.Expr, error)
	at         time.Time
	ctes       []string
	joins      []string
	arguments  []warehouses.QueryArgument
	counts     int
	currencies []CurrencyBinding
}

func (c *compiler) count(count *Count) (string, error) {

	n := strconv.Itoa(c.counts)
	c.counts++
	q := c.dialect.QuoteIdentifier
	name, err := c.parameter(types.String(), count.Scope.Event)
	if err != nil {
		return "", err
	}
	selector := q("event") + "=" + name
	qual := "TRUE"
	if count.Where != nil {
		expr, err := c.bind(count.Where)
		if err != nil {
			return "", err
		}
		qual, err = c.dialect.Record(expr)
		if err != nil {
			return "", err
		}
	}
	if count.Currency != nil {
		var binding *CurrencyBinding
		for i, candidate := range c.currencies {
			if candidate.Definition == count.Currency.Definition && candidate.Revision == count.Currency.Revision {
				binding = &c.currencies[i]
				break
			}
		}
		if binding == nil {
			return "", fmt.Errorf("unresolved NativeCurrency meaning revision")
		}
		if binding.Evidence != IndependentOccurrenceValues {
			return "", &UnsupportedError{Capability: "governed evidence without independent occurrence values"}
		}
		column := warehouses.Column{Name: "properties", Type: types.JSON(), Nullable: true}
		value, err := c.dialect.KnownText(column, binding.Path[1:])
		if err != nil {
			return "", err
		}
		parameter, err := c.parameter(types.String(), count.Currency.Value)
		if err != nil {
			return "", err
		}
		qual = "(" + value + ")=" + parameter
	}
	// The named-event test is a local qualifier, not a WHERE that would discard
	// an existing event with an unknown name before the evidence is consumed.
	qual = "(" + selector + ") AND (" + qual + ")"
	where := "TRUE"
	start, err := c.parameter(types.DateTime(), count.Scope.Start)
	if err != nil {
		return "", err
	}
	end, err := c.parameter(types.DateTime(), count.Scope.End)
	if err != nil {
		return "", err
	}
	where += " AND " + q("timestamp") + ">=" + start + " AND " + q("timestamp") + "<" + end
	at, err := c.parameter(types.DateTime(), c.at)
	if err != nil {
		return "", err
	}
	where += " AND " + q("timestamp") + "<" + at

	// Deduplicate by the determined occurrence identity, never by payload. This
	// also protects against multiplicity introduced by a source view's joins.
	c.ctes = append(c.ctes, "occ"+n+" AS (SELECT DISTINCT "+q("kpid")+" AS profile_id,"+q("message_id")+
		" AS occurrence_id,("+qual+") AS qual FROM "+q("krenalis_events")+" WHERE "+where+")")
	c.ctes = append(c.ctes, "cnt"+n+` AS (SELECT profile_id,
 SUM(CASE WHEN qual=TRUE THEN 1 ELSE 0 END) AS lo,
 SUM(CASE WHEN qual=FALSE THEN 0 ELSE 1 END) AS hi FROM occ`+n+" GROUP BY profile_id)")
	alias := "v" + n
	c.joins = append(c.joins, "LEFT JOIN cnt"+n+" "+alias+" ON "+alias+".profile_id=p."+q("_kpid"))
	lo, hi := "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)"

	return countTerminal(count.Test, count.Minimum, lo, hi), nil
}

func (c *compiler) parameter(typ types.Type, value any) (string, error) {
	index := len(c.arguments) + 1
	text, err := c.dialect.Parameter(index, typ)
	if err != nil {
		return "", err
	}
	c.arguments = append(c.arguments, warehouses.QueryArgument{Type: typ, Value: value})
	return text, nil
}

func (c *compiler) rule(rule Rule) (string, error) {
	if rule.Count != nil {
		return c.count(rule.Count)
	}
	parts := make([]string, len(rule.Children))
	for i, child := range rule.Children {
		text, err := c.rule(child)
		if err != nil {
			return "", err
		}
		parts[i] = "(" + text + ")"
	}
	operator := " AND "
	if rule.Operator == state.OpOr {
		operator = " OR "
	}
	return strings.Join(parts, operator), nil
}

// Compile validates and binds the Count subset, then emits a single statement
// for its outcome page and all three profile totals. bindRecord uses the
// existing datastore Record binding, without interpreting its operators again.
func Compile(root *Rule, bindings Bindings, at time.Time, page Page, dialect warehouses.QueryDialect, bindRecord func(*state.Where) (warehouses.Expr, error)) (warehouses.CompiledQuery, error) {

	err := Validate(root, at, page)
	if err != nil {
		return warehouses.CompiledQuery{}, err
	}
	if dialect.QuoteIdentifier == nil || dialect.Record == nil || dialect.Parameter == nil ||
		dialect.KnownText == nil || bindRecord == nil {
		return warehouses.CompiledQuery{}, fmt.Errorf("core/internal/unified: missing compiler adapter")
	}
	if len(bindings.Currencies) > 100 {
		return warehouses.CompiledQuery{}, fmt.Errorf("invalid currency binding count")
	}
	for i, b := range bindings.Currencies {
		if b.Definition == "" || b.Revision == "" || len(b.Definition) > 256 || len(b.Revision) > 256 ||
			len(b.Path) < 2 || len(b.Path) > 32 || b.Path[0] != "properties" {
			return warehouses.CompiledQuery{}, fmt.Errorf("invalid resolved currency binding")
		}
		for _, other := range bindings.Currencies[:i] {
			if b.Definition == other.Definition && b.Revision == other.Revision {
				return warehouses.CompiledQuery{}, fmt.Errorf("duplicate currency binding")
			}
		}
	}

	c := compiler{dialect: dialect, bind: bindRecord, currencies: bindings.Currencies, at: at.UTC()}
	truth := "TRUE"
	if root != nil {
		truth, err = c.rule(*root)
		if err != nil {
			return warehouses.CompiledQuery{}, err
		}
	}
	profiles := dialect.QuoteIdentifier("profiles")
	profileID := dialect.QuoteIdentifier("_kpid")
	c.ctes = append(c.ctes, "evaluated AS (SELECT p."+profileID+" AS profile_id, ("+truth+") AS truth FROM "+
		profiles+" p "+strings.Join(c.joins, " ")+")")
	c.ctes = append(c.ctes, `totals AS (SELECT
 COALESCE(SUM(CASE WHEN truth=TRUE THEN 1 ELSE 0 END),0) AS m,
 COALESCE(SUM(CASE WHEN truth=FALSE THEN 1 ELSE 0 END),0) AS f,
 COALESCE(SUM(CASE WHEN truth IS NULL THEN 1 ELSE 0 END),0) AS u FROM evaluated)`)
	selection := ""
	if page.MatchingOnly {
		selection = " WHERE truth=TRUE"
	}
	c.ctes = append(c.ctes, "outcome_page AS (SELECT profile_id,truth,TRUE AS is_profile FROM evaluated"+selection+
		" ORDER BY profile_id LIMIT "+strconv.Itoa(page.Limit)+" OFFSET "+strconv.Itoa(page.First)+")")
	query := warehouses.CompiledQuery{
		SQL: "WITH " + strings.Join(c.ctes, ",\n") + `
SELECT CAST(b.profile_id AS VARCHAR(36)),b.truth,COALESCE(b.is_profile,FALSE),a.m,a.f,a.u
FROM totals a LEFT JOIN outcome_page b ON TRUE ORDER BY b.profile_id`,
		Arguments: c.arguments,
		Columns: []warehouses.Column{
			{Name: "profile_id", Type: types.UUID(), Nullable: true},
			{Name: "truth", Type: types.Boolean(), Nullable: true},
			{Name: "is_profile", Type: types.Boolean()},
			{Name: "match_count", Type: types.Int(64)},
			{Name: "no_match_count", Type: types.Int(64)},
			{Name: "unknown_count", Type: types.Int(64)},
		},
	}

	return query, nil
}

// countTerminal is exact because each integer in [lo,hi] is reachable for the
// admitted independent occurrence memberships. It cannot be reused for shares,
// shared governed facts, uncertain grain or K's compositional quantification.
func countTerminal(test CountTest, minimum uint64, lo, hi string) string {
	lower, upper := minimum, uint64(math.MaxInt64)
	upperBounded := false
	var yes, no string
	switch test.Operator {
	case Different:
		value := strconv.FormatUint(test.Value, 10)
		minText := strconv.FormatUint(minimum, 10)
		yes = lo + ">=" + minText + " AND (" + value + "<" + lo + " OR " + value + ">" + hi + ")"
		no = hi + "<" + minText + " OR (" + hi + "=" + value + " AND (" + lo + "=" + hi + " OR " + hi + "=" + minText + "))"
	case Equal:
		lower, upper = max(lower, test.Value), test.Value
		upperBounded = true
	case Less:
		if test.Value == 0 {
			return "FALSE"
		}
		upper = test.Value - 1
		upperBounded = true
	case LessOrEqual:
		upper = test.Value
		upperBounded = true
	case Greater:
		if test.Value == math.MaxInt64 {
			return "FALSE"
		}
		lower = max(lower, test.Value+1)
	case GreaterOrEqual:
		lower = max(lower, test.Value)
	case Between:
		lower, upper = max(lower, test.Value), test.Upper
		upperBounded = true
	default:
		panic(fmt.Sprintf("core/internal/unified: unexpected Count terminal %d", test.Operator))
	}
	if test.Operator != Different {
		if upperBounded && lower > upper {
			return "FALSE"
		}
		low, high := strconv.FormatUint(lower, 10), strconv.FormatUint(upper, 10)
		yes, no = lo+">="+low, hi+"<"+low
		if upperBounded {
			yes += " AND " + hi + "<=" + high
			no += " OR " + lo + ">" + high
		}
	}
	return "CASE WHEN (" + yes + ") THEN TRUE WHEN (" + no + ") THEN FALSE ELSE CAST(NULL AS BOOLEAN) END"
}
