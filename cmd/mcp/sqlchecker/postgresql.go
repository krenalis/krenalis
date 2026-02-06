// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package sqlchecker validates PostgreSQL queries to ensure they are read-only
// and cannot overload the data warehouse. It uses static analysis on the AST
// produced by the PostgreSQL parser.
package sqlchecker

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	pg_query_wasi "github.com/wasilibs/go-pgquery"
)

// CheckPostgreSQL validates a PostgreSQL query ensuring it is read-only and
// safe to execute against the data warehouse. It returns an error if the query
// is not allowed.
//
// This function does not provide any security guarantee. It is a best-effort,
// static-analysis heuristic and must not be treated as a security boundary.
// Always enforce proper security controls (e.g. read-only database roles,
// restricted permissions, query allow-lists) at the infrastructure level.
func CheckPostgreSQL(query string) error {
	result, err := pg_query_wasi.Parse(query)
	if err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}
	if len(result.Stmts) == 0 {
		return fmt.Errorf("empty query")
	}
	for _, rawStmt := range result.Stmts {
		stmt := rawStmt.Stmt
		if stmt == nil {
			continue
		}
		selectStmt := stmt.GetSelectStmt()
		if selectStmt == nil {
			return fmt.Errorf("only SELECT statements are allowed")
		}
		if err := checkSelectStmt(selectStmt); err != nil {
			return err
		}
	}
	return nil
}

// dangerousFunctions is the set of PostgreSQL functions that are blocked
// because they have side effects or can cause denial of service.
var dangerousFunctions = map[string]string{
	// Sleep / blocking.
	"pg_sleep":          "blocks execution",
	"pg_sleep_for":      "blocks execution",
	"pg_sleep_until":    "blocks execution",
	"pg_terminate_backend": "can terminate other sessions",
	"pg_cancel_backend":    "can cancel other sessions",

	// Large objects.
	"lo_import": "large object operation",
	"lo_export": "large object operation",
	"lo_unlink": "large object operation",

	// External connections.
	"dblink":      "external database connection",
	"dblink_exec": "external database connection",

	// Configuration changes.
	"set_config":      "modifies server configuration",
	"pg_reload_conf":  "reloads server configuration",

	// Advisory locks.
	"pg_advisory_lock":             "acquires advisory lock",
	"pg_advisory_lock_shared":      "acquires advisory lock",
	"pg_advisory_xact_lock":        "acquires advisory lock",
	"pg_advisory_xact_lock_shared": "acquires advisory lock",
	"pg_try_advisory_lock":         "acquires advisory lock",
	"pg_try_advisory_lock_shared":  "acquires advisory lock",
	"pg_try_advisory_xact_lock":    "acquires advisory lock",
	"pg_try_advisory_xact_lock_shared": "acquires advisory lock",

	// Sequence side effects.
	"nextval": "modifies sequence state",
	"setval":  "modifies sequence state",
	"currval": "depends on sequence state",

	// Transaction ID.
	"txid_current":          "exposes transaction state",
	"txid_current_snapshot": "exposes transaction state",
}

// checkSelectStmt validates a single SELECT statement.
func checkSelectStmt(stmt *pg_query.SelectStmt) error {
	// Check SELECT ... INTO (creates a new table).
	if stmt.IntoClause != nil {
		return fmt.Errorf("SELECT INTO is not allowed: it creates a new table")
	}

	// Check WITH RECURSIVE without LIMIT.
	if stmt.WithClause != nil && stmt.WithClause.Recursive {
		if !selectHasLimit(stmt) {
			return fmt.Errorf("WITH RECURSIVE requires a LIMIT clause")
		}
	}

	// Check cross joins.
	if err := checkCrossJoins(stmt); err != nil {
		return err
	}

	// Recursively check for dangerous functions and generate_series abuse
	// in the entire statement.
	if err := checkNode(stmt); err != nil {
		return err
	}

	// Check UNION / INTERSECT / EXCEPT branches.
	if stmt.Larg != nil {
		if err := checkSelectStmt(stmt.Larg); err != nil {
			return err
		}
	}
	if stmt.Rarg != nil {
		if err := checkSelectStmt(stmt.Rarg); err != nil {
			return err
		}
	}

	return nil
}

// selectHasLimit returns true if the SELECT statement (or the outermost one
// in a set operation) has a LIMIT clause.
func selectHasLimit(stmt *pg_query.SelectStmt) bool {
	if stmt.LimitCount != nil {
		return true
	}
	// For UNION/INTERSECT/EXCEPT the LIMIT is on the outermost statement,
	// which is already checked, so we only need to check the current level.
	return false
}

// checkCrossJoins checks for explicit CROSS JOINs and implicit cross joins
// (multiple FROM items without a WHERE clause).
func checkCrossJoins(stmt *pg_query.SelectStmt) error {
	// Check for explicit CROSS JOIN in FROM clause.
	for _, from := range stmt.FromClause {
		if err := checkForCrossJoin(from); err != nil {
			return err
		}
	}

	// Check for implicit cross join: multiple FROM items without WHERE.
	if len(stmt.FromClause) > 1 && stmt.WhereClause == nil {
		return fmt.Errorf("implicit cross join detected: multiple FROM items without a WHERE clause")
	}

	return nil
}

// checkForCrossJoin recursively checks for explicit CROSS JOIN nodes.
func checkForCrossJoin(node *pg_query.Node) error {
	if node == nil {
		return nil
	}
	joinExpr := node.GetJoinExpr()
	if joinExpr != nil {
		if joinExpr.Jointype == pg_query.JoinType_JOIN_INNER && joinExpr.Quals == nil && len(joinExpr.UsingClause) == 0 {
			// Inner join without ON/USING is a cross join.
			return fmt.Errorf("cross join is not allowed")
		}
		// Recurse into sub-joins.
		if err := checkForCrossJoin(joinExpr.Larg); err != nil {
			return err
		}
		if err := checkForCrossJoin(joinExpr.Rarg); err != nil {
			return err
		}
	}
	return nil
}

// checkNode recursively inspects AST nodes for dangerous function calls
// and abusive generate_series usage.
func checkNode(node interface{}) error {
	switch n := node.(type) {
	case *pg_query.SelectStmt:
		if n == nil {
			return nil
		}
		for _, target := range n.TargetList {
			if err := checkNode(target); err != nil {
				return err
			}
		}
		for _, from := range n.FromClause {
			if err := checkNode(from); err != nil {
				return err
			}
		}
		if err := checkNode(n.WhereClause); err != nil {
			return err
		}
		if err := checkNode(n.HavingClause); err != nil {
			return err
		}
		for _, group := range n.GroupClause {
			if err := checkNode(group); err != nil {
				return err
			}
		}
		for _, sort := range n.SortClause {
			if err := checkNode(sort); err != nil {
				return err
			}
		}
		if err := checkNode(n.LimitCount); err != nil {
			return err
		}
		if err := checkNode(n.LimitOffset); err != nil {
			return err
		}
		if n.WithClause != nil {
			for _, cte := range n.WithClause.Ctes {
				if err := checkNode(cte); err != nil {
					return err
				}
			}
		}
		for _, window := range n.WindowClause {
			if err := checkNode(window); err != nil {
				return err
			}
		}
		for _, distinct := range n.DistinctClause {
			if err := checkNode(distinct); err != nil {
				return err
			}
		}
		for _, locking := range n.LockingClause {
			if err := checkNode(locking); err != nil {
				return err
			}
		}
		if err := checkNode(n.Larg); err != nil {
			return err
		}
		if err := checkNode(n.Rarg); err != nil {
			return err
		}

	case *pg_query.Node:
		if n == nil {
			return nil
		}
		return checkNodeInner(n)

	case *pg_query.FuncCall:
		if n == nil {
			return nil
		}
		return checkFuncCall(n)

	case *pg_query.CommonTableExpr:
		if n == nil {
			return nil
		}
		if n.Ctequery != nil {
			return checkNode(n.Ctequery)
		}

	case *pg_query.SubLink:
		if n == nil {
			return nil
		}
		if n.Subselect != nil {
			if err := checkNode(n.Subselect); err != nil {
				return err
			}
		}
		if n.Testexpr != nil {
			if err := checkNode(n.Testexpr); err != nil {
				return err
			}
		}

	case *pg_query.ResTarget:
		if n == nil {
			return nil
		}
		return checkNode(n.Val)

	case *pg_query.SortBy:
		if n == nil {
			return nil
		}
		return checkNode(n.Node)

	case nil:
		return nil
	}
	return nil
}

// checkNodeInner handles checking a *pg_query.Node by dispatching to the
// appropriate concrete type.
func checkNodeInner(n *pg_query.Node) error {
	if n == nil {
		return nil
	}

	// Check function calls.
	if fc := n.GetFuncCall(); fc != nil {
		return checkFuncCall(fc)
	}

	// Check subqueries.
	if sub := n.GetSubLink(); sub != nil {
		return checkNode(sub)
	}

	// Check SELECT in FROM clause (subselect / RangeSubselect).
	if rs := n.GetRangeSubselect(); rs != nil {
		if rs.Subquery != nil {
			return checkNode(rs.Subquery)
		}
	}

	// Check range function (function in FROM clause).
	if rf := n.GetRangeFunction(); rf != nil {
		for _, funcItem := range rf.Functions {
			if err := checkNode(funcItem); err != nil {
				return err
			}
		}
	}

	// Check JOIN expressions.
	if je := n.GetJoinExpr(); je != nil {
		if err := checkNode(je.Larg); err != nil {
			return err
		}
		if err := checkNode(je.Rarg); err != nil {
			return err
		}
		if err := checkNode(je.Quals); err != nil {
			return err
		}
	}

	// Check boolean expressions.
	if be := n.GetBoolExpr(); be != nil {
		for _, arg := range be.Args {
			if err := checkNode(arg); err != nil {
				return err
			}
		}
	}

	// Check CASE expressions.
	if ce := n.GetCaseExpr(); ce != nil {
		if err := checkNode(ce.Arg); err != nil {
			return err
		}
		for _, when := range ce.Args {
			if err := checkNode(when); err != nil {
				return err
			}
		}
		if err := checkNode(ce.Defresult); err != nil {
			return err
		}
	}

	if cw := n.GetCaseWhen(); cw != nil {
		if err := checkNode(cw.Expr); err != nil {
			return err
		}
		if err := checkNode(cw.Result); err != nil {
			return err
		}
	}

	// Check COALESCE and similar.
	if ce := n.GetCoalesceExpr(); ce != nil {
		for _, arg := range ce.Args {
			if err := checkNode(arg); err != nil {
				return err
			}
		}
	}

	// Check type cast (A_Expr, TypeCast, etc.).
	if tc := n.GetTypeCast(); tc != nil {
		if err := checkNode(tc.Arg); err != nil {
			return err
		}
	}

	if ae := n.GetAExpr(); ae != nil {
		if err := checkNode(ae.Lexpr); err != nil {
			return err
		}
		if err := checkNode(ae.Rexpr); err != nil {
			return err
		}
	}

	// Check ResTarget (target list entries).
	if rt := n.GetResTarget(); rt != nil {
		if err := checkNode(rt.Val); err != nil {
			return err
		}
	}

	// Check SelectStmt nested in a Node.
	if ss := n.GetSelectStmt(); ss != nil {
		return checkSelectStmt(ss)
	}

	// Check CommonTableExpr in a Node.
	if cte := n.GetCommonTableExpr(); cte != nil {
		return checkNode(cte)
	}

	// Check lists (used in RangeFunction.functions and elsewhere).
	if list := n.GetList(); list != nil {
		for _, item := range list.Items {
			if err := checkNode(item); err != nil {
				return err
			}
		}
	}

	// Check NullTest.
	if nt := n.GetNullTest(); nt != nil {
		if err := checkNode(nt.Arg); err != nil {
			return err
		}
	}

	// Check SortBy.
	if sb := n.GetSortBy(); sb != nil {
		if err := checkNode(sb.Node); err != nil {
			return err
		}
	}

	// Check Agg (WindowFunc).
	if wf := n.GetWindowFunc(); wf != nil {
		for _, arg := range wf.Args {
			if err := checkNode(arg); err != nil {
				return err
			}
		}
	}

	// Check MinMaxExpr.
	if mm := n.GetMinMaxExpr(); mm != nil {
		for _, arg := range mm.Args {
			if err := checkNode(arg); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkFuncCall checks whether a function call is allowed.
func checkFuncCall(fc *pg_query.FuncCall) error {
	name := funcName(fc)

	// Check if it's a dangerous function.
	if reason, ok := dangerousFunctions[name]; ok {
		return fmt.Errorf("function %q is not allowed: %s", name, reason)
	}

	// Check generate_series with large literal ranges.
	if name == "generate_series" {
		if err := checkGenerateSeries(fc); err != nil {
			return err
		}
	}

	// Recurse into function arguments.
	for _, arg := range fc.Args {
		if err := checkNode(arg); err != nil {
			return err
		}
	}
	// Check FILTER and OVER clauses.
	if fc.AggFilter != nil {
		if err := checkNode(fc.AggFilter); err != nil {
			return err
		}
	}
	for _, order := range fc.AggOrder {
		if err := checkNode(order); err != nil {
			return err
		}
	}

	return nil
}

// funcName extracts the function name from a FuncCall node.
// Schema-qualified names are returned as "schema.name".
func funcName(fc *pg_query.FuncCall) string {
	parts := make([]string, 0, len(fc.Funcname))
	for _, n := range fc.Funcname {
		if s := n.GetString_(); s != nil {
			parts = append(parts, strings.ToLower(s.Sval))
		}
	}
	return strings.Join(parts, ".")
}

// maxGenerateSeriesRange is the maximum allowed range for generate_series
// when both arguments are integer literals.
const maxGenerateSeriesRange = 1_000_000

// checkGenerateSeries validates generate_series calls to prevent
// creating extremely large result sets.
func checkGenerateSeries(fc *pg_query.FuncCall) error {
	if len(fc.Args) < 2 {
		return nil
	}
	start, startOk := integerLiteral(fc.Args[0])
	end, endOk := integerLiteral(fc.Args[1])
	if startOk && endOk {
		rangeSize := end - start
		if rangeSize < 0 {
			rangeSize = -rangeSize
		}
		if rangeSize > maxGenerateSeriesRange {
			return fmt.Errorf("generate_series range too large: %d (max %d)", rangeSize, maxGenerateSeriesRange)
		}
	}
	return nil
}

// integerLiteral extracts an integer value from a constant AST node.
func integerLiteral(node *pg_query.Node) (int64, bool) {
	if node == nil {
		return 0, false
	}
	if c := node.GetAConst(); c != nil {
		if iv := c.GetIval(); iv != nil {
			return int64(iv.Ival), true
		}
	}
	// Check for negative numbers: A_Expr with AEXPR_OP '-' and a single operand,
	// or a unary minus represented differently.
	if tc := node.GetTypeCast(); tc != nil {
		return integerLiteral(tc.Arg)
	}
	return 0, false
}
