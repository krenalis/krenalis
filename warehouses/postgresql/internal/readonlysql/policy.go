// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import "strings"

// forbiddenTokens contains visible SQL words that cause immediate rejection
// outside opaque regions.
var forbiddenTokens = newASCIIWordSet(
	"alter",
	"analyze",
	"begin",
	"call",
	"checkpoint",
	"cluster",
	"close",
	"comment",
	"commit",
	"copy",
	"create",
	"declare",
	"deallocate",
	"delete",
	"discard",
	"do",
	"drop",
	"execute",
	"fetch",
	"grant",
	"insert",
	"into",
	"listen",
	"lock",
	"merge",
	"move",
	"notify",
	"prepare",
	"refresh",
	"reindex",
	"release",
	"reset",
	"revoke",
	"rollback",
	"savepoint",
	"security",
	"set",
	"show",
	"start",
	"truncate",
	"unlisten",
	"update",
	"vacuum",
)

// nonFunctionCallTokens contains words that may precede parentheses without
// starting a function call.
var nonFunctionCallTokens = newASCIIWordSet(
	"all",
	"any",
	"as",
	"cast",
	"exists",
	"filter",
	"in",
	"row",
	"some",
)

// allowedFunctionCalls is the PostgreSQL 14 through 18 allowlist used by
// ValidateReadOnly.
//
// Names are unqualified and compared case-insensitively after normalization to
// lower case. Qualified function calls are rejected elsewhere. The list must be
// reviewed when upgrading PostgreSQL. It is intentionally limited to a
// practical BI-oriented subset rather than all safe PostgreSQL built-ins.
var allowedFunctionCalls = newASCIIWordSet(
	"abs",
	"array_agg",
	"array_length",
	"avg",
	"bool_and",
	"bool_or",
	"btrim",
	"cardinality",
	"ceil",
	"ceiling",
	"coalesce",
	"concat",
	"concat_ws",
	"count",
	"date_part",
	"date_trunc",
	"extract",
	"floor",
	"greatest",
	"json_agg",
	"jsonb_agg",
	"jsonb_array_length",
	"jsonb_extract_path",
	"jsonb_extract_path_text",
	"jsonb_object_keys",
	"jsonb_typeof",
	"least",
	"left",
	"length",
	"lower",
	"ltrim",
	"max",
	"min",
	"nullif",
	"replace",
	"right",
	"round",
	"rtrim",
	"split_part",
	"string_agg",
	"substring",
	"sum",
	"to_char",
	"unnest",
	"upper",
)

// PostgreSQL 14 through 18 special non-parenthesized forms handled separately
// from normal function calls.
//
// The checker allows only a narrow BI-oriented subset, matched
// case-insensitively as standalone SQL words. Qualified calls are handled
// elsewhere. This list must be reviewed when moving outside PostgreSQL 14
// through 18.
var allowedSpecialForms = newASCIIWordSet(
	"current_date",
	"current_time",
	"current_timestamp",
	"localtime",
	"localtimestamp",
)

// disallowedSpecialForms contains special forms that are recognized and
// rejected explicitly.
var disallowedSpecialForms = newASCIIWordSet(
	"current_catalog",
	"current_role",
	"current_schema",
	"current_user",
	"session_user",
	"user",
)

func isForbiddenToken(token string) bool {
	return forbiddenTokens.Has(token)
}

func isNonFunctionCallKeyword(token string) bool {
	return nonFunctionCallTokens.Has(token)
}

func isAllowedFunction(name string) bool {
	return allowedFunctionCalls.Has(name)
}

func isAllowedSpecialForm(name string) bool {
	return allowedSpecialForms.Has(name)
}

func isDisallowedSpecialForm(name string) bool {
	return disallowedSpecialForms.Has(name)
}

func handleSpecialForm(sql string, name scannedName) (handled bool, next int, err error) {
	if isDisallowedSpecialForm(name.normalized) {
		return true, 0, rejectSpecialFormNotAllowed(strings.ToUpper(name.token))
	}
	if !isAllowedSpecialForm(name.normalized) {
		return false, 0, nil
	}
	if !name.isFunctionCall {
		return true, name.next, nil
	}
	next, err = parseSpecialFormSuffix(sql, strings.ToUpper(name.token), name.normalized, name.next)
	return true, next, err
}

func specialFormAllowsPrecision(name string) bool {
	switch name {
	case "current_time", "current_timestamp", "localtime", "localtimestamp":
		return true
	default:
		return false
	}
}

func parseSpecialFormSuffix(sql string, upperName string, normalizedName string, start int) (int, error) {
	if !specialFormAllowsPrecision(normalizedName) {
		return 0, rejectSpecialFormDoesNotAllowParentheses(upperName)
	}

	i, err := skipIgnored(sql, start)
	if err != nil {
		return 0, err
	}
	if i >= len(sql) || sql[i] != '(' {
		return 0, rejectMalformedSpecialFormPrecision(upperName)
	}
	i++

	i, err = skipIgnored(sql, i)
	if err != nil {
		return 0, err
	}
	digitStart := i
	for i < len(sql) && isDigit(sql[i]) {
		i++
	}
	if digitStart == i {
		return 0, rejectMalformedSpecialFormPrecision(upperName)
	}

	i, err = skipIgnored(sql, i)
	if err != nil {
		return 0, err
	}
	if i >= len(sql) || sql[i] != ')' {
		return 0, rejectMalformedSpecialFormPrecision(upperName)
	}
	return i + 1, nil
}
