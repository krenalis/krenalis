// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

// forbiddenTokens contains visible SQL words that cause immediate rejection
// outside opaque regions.
var forbiddenTokens = newASCIIWordSet(
	"alter",
	"begin",
	"call",
	"comment",
	"commit",
	"copy",
	"create",
	"declare",
	"delete",
	"describe",
	"drop",
	"execute",
	"explain",
	"get",
	"grant",
	"immediate",
	"insert",
	"let",
	"list",
	"ls",
	"merge",
	"put",
	"remove",
	"rename",
	"return",
	"revoke",
	"rm",
	"rollback",
	"set",
	"show",
	"truncate",
	"undrop",
	"unset",
	"update",
	"use",
)

// nonFunctionCallTokens contains words that may precede parentheses without
// starting a function call.
var nonFunctionCallTokens = newASCIIWordSet(
	"all",
	"any",
	"as",
	"cast",
	"case",
	"exists",
	"from",
	"in",
	"join",
	"over",
	"some",
	"when",
)

// allowedFunctionCalls is the initial Snowflake allowlist used by
// ValidateReadOnly.
var allowedFunctionCalls = newASCIIWordSet(
	"abs",
	"array_construct",
	"array_size",
	"approx_count_distinct",
	"avg",
	"ceil",
	"ceiling",
	"coalesce",
	"concat",
	"concat_ws",
	"count",
	"count_if",
	"current_date",
	"current_time",
	"current_timestamp",
	"dateadd",
	"date_part",
	"date_trunc",
	"datediff",
	"dense_rank",
	"extract",
	"first_value",
	"floor",
	"get",
	"get_path",
	"json_extract_path_text",
	"lag",
	"last_value",
	"lead",
	"length",
	"len",
	"listagg",
	"lower",
	"ltrim",
	"max",
	"median",
	"min",
	"mode",
	"nullif",
	"ntile",
	"object_construct",
	"percentile_cont",
	"replace",
	"rank",
	"round",
	"row_number",
	"rtrim",
	"split_part",
	"substring",
	"sum",
	"trim",
	"typeof",
	"upper",
)

// disallowedSpecialForms contains context forms rejected even without
// parentheses.
var disallowedSpecialForms = newASCIIWordSet(
	"current_account",
	"current_database",
	"current_role",
	"current_schema",
	"current_user",
	"current_warehouse",
)

// isForbiddenToken reports whether token is forbidden.
func isForbiddenToken(token string) bool {
	return forbiddenTokens.Has(token)
}

// isNonFunctionCallKeyword reports whether token may be followed by
// parentheses without being a function call.
func isNonFunctionCallKeyword(token string) bool {
	return nonFunctionCallTokens.Has(token)
}

// isAllowedFunction reports whether name is an allowed function call.
func isAllowedFunction(name string) bool {
	return allowedFunctionCalls.Has(name)
}

// isDisallowedSpecialForm reports whether name is a rejected special form.
func isDisallowedSpecialForm(name string) bool {
	return disallowedSpecialForms.Has(name)
}
