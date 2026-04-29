// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import (
	"strings"
	"testing"
)

// TestPolicyForbiddenTokens verifies the Snowflake command denylist.
func TestPolicyForbiddenTokens(t *testing.T) {
	tests := []string{
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
	}

	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			if !isForbiddenToken(token) {
				t.Fatalf("expected %q to be forbidden, got allowed", token)
			}
			if !isForbiddenToken(strings.ToUpper(token)) {
				t.Fatalf("expected %q to be forbidden, got allowed", strings.ToUpper(token))
			}
		})
	}
}

// TestPolicyAllowedFunctions verifies the initial BI-oriented allowlist.
func TestPolicyAllowedFunctions(t *testing.T) {
	tests := []string{
		"abs",
		"array_construct",
		"array_size",
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
		"date_part",
		"date_trunc",
		"extract",
		"floor",
		"get",
		"get_path",
		"json_extract_path_text",
		"length",
		"len",
		"listagg",
		"lower",
		"ltrim",
		"max",
		"min",
		"nullif",
		"object_construct",
		"replace",
		"round",
		"rtrim",
		"split_part",
		"substring",
		"sum",
		"trim",
		"typeof",
		"upper",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if !isAllowedFunction(name) {
				t.Fatalf("expected %q to be allowlisted, got rejected", name)
			}
			if !isAllowedFunction(strings.ToUpper(name)) {
				t.Fatalf("expected %q to be allowlisted, got rejected", strings.ToUpper(name))
			}
		})
	}
}

// TestPolicyRejectedFunctions verifies representative non-allowlisted calls.
func TestPolicyRejectedFunctions(t *testing.T) {
	tests := []string{
		"identifier",
		"last_query_id",
		"result_scan",
		"try_cast",
		"current_role",
		"current_user",
		"current_schema",
		"cortex_complete",
		"unknown_function",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if isAllowedFunction(name) {
				t.Fatalf("expected %q to be rejected, got allowlisted", name)
			}
		})
	}
}

// TestPolicyDisallowedSpecialForms verifies context forms without parentheses.
func TestPolicyDisallowedSpecialForms(t *testing.T) {
	tests := []string{
		"current_account",
		"current_database",
		"current_role",
		"current_schema",
		"current_user",
		"current_warehouse",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if !isDisallowedSpecialForm(name) {
				t.Fatalf("expected %q to be a disallowed special form, got allowed", name)
			}
		})
	}
}
