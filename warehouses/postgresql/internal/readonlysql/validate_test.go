// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import (
	"errors"
	"strings"
	"testing"

	"github.com/meergo/meergo/warehouses"
)

// TestValidateReadOnlyStatements verifies statement-level acceptance and
// rejection.
func TestValidateReadOnlyStatements(t *testing.T) {
	acceptTests := []struct {
		name string
		sql  string
	}{
		{name: "simple select", sql: "SELECT 1"},
		{name: "forbidden token in single quote", sql: "SELECT 'DELETE'"},
		{name: "forbidden token in dollar quote", sql: "SELECT $$DELETE$$"},
		{name: "forbidden token in quoted identifier", sql: `SELECT "DELETE" FROM t`},
		{name: "cast operator in string", sql: "SELECT '::'"},
		{name: "cast operator in dollar quote", sql: "SELECT $$::$$"},
		{name: "share identifier", sql: "SELECT share FROM t"},
		{name: "unicode prefix separated by operator", sql: `SELECT U & "foo" FROM t`},
		{name: "with select", sql: "WITH a AS (SELECT 1) SELECT * FROM a"},
	}

	for _, tt := range acceptTests {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}

	rejectTests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{name: "delete", sql: "DELETE FROM t", wantErr: "rejected: forbidden token DELETE found outside opaque region"},
		{name: "semicolon multi statement", sql: "SELECT 1; DELETE FROM t", wantErr: "rejected: semicolon found outside opaque region"},
		{name: "select into", sql: "SELECT * INTO new_table FROM t", wantErr: "rejected: forbidden token INTO found outside opaque region"},
		{name: "delete inside with", sql: "WITH a AS (DELETE FROM t RETURNING *) SELECT * FROM a", wantErr: "rejected: forbidden token DELETE found outside opaque region"},
		{name: "insert", sql: "INSERT INTO t VALUES (1)", wantErr: "rejected: forbidden token INSERT found outside opaque region"},
		{name: "update", sql: "UPDATE t SET x = 1", wantErr: "rejected: forbidden token UPDATE found outside opaque region"},
		{name: "create table", sql: "CREATE TABLE x (id int)", wantErr: "rejected: forbidden token CREATE found outside opaque region"},
		{name: "for share", sql: "SELECT * FROM t FOR SHARE", wantErr: "rejected: locking clause FOR SHARE is not allowed"},
		{name: "for key share", sql: "SELECT * FROM t FOR KEY SHARE", wantErr: "rejected: locking clause FOR KEY SHARE is not allowed"},
		{name: "for key share with comments", sql: "SELECT * FROM t FOR /*x*/ KEY /*y*/ SHARE", wantErr: "rejected: locking clause FOR KEY SHARE is not allowed"},
		{name: "builtin cast", sql: "SELECT 1::int", wantErr: "rejected: type cast operator :: is not allowed"},
		{name: "qualified custom cast", sql: "SELECT 'abc'::public.some_type", wantErr: "rejected: type cast operator :: is not allowed"},
		{name: "empty", sql: "", wantErr: "rejected: no visible SELECT token found"},
		{name: "comment only", sql: "-- comment only", wantErr: "rejected: no visible SELECT token found"},
		{name: "unterminated block comment", sql: "SELECT 1 /* unterminated", wantErr: "rejected: unterminated block comment"},
		{name: "unterminated single quoted string", sql: "SELECT 'unterminated", wantErr: "rejected: unterminated single-quoted string"},
		{name: "unterminated dollar quoted string", sql: "SELECT $$unterminated", wantErr: "rejected: unterminated dollar-quoted string"},
		{name: "unicode quoted identifier", sql: `SELECT U&"d\0061t\+000061" FROM t`, wantErr: `rejected: Unicode quoted identifier syntax U&"..." is not supported`},
		{name: "unicode quoted identifier lowercase", sql: `SELECT u&"d\0061t\+000061" FROM t`, wantErr: `rejected: Unicode quoted identifier syntax U&"..." is not supported`},
		{name: "unterminated unicode quoted identifier", sql: `SELECT U&"unterminated`, wantErr: `rejected: Unicode quoted identifier syntax U&"..." is not supported`},
	}

	for _, tt := range rejectTests {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			assertExactError(t, err, tt.wantErr)
			assertRejectedError(t, err)
			assertNoRejectedFunctionError(t, err)
		})
	}
}

// TestValidateReadOnlyFunctionsAllowed verifies allowed function calls.
func TestValidateReadOnlyFunctionsAllowed(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "lower", sql: "SELECT lower('ABC')"},
		{name: "abs", sql: "SELECT abs(-1)"},
		{name: "substring", sql: "SELECT substring('abcd', 1, 2)"},
		{name: "coalesce", sql: "SELECT coalesce(NULL, 1)"},
		{name: "count", sql: "SELECT count(*)"},
		{name: "sum", sql: "SELECT sum(connection_id) FROM meergo_events"},
		{name: "avg", sql: "SELECT avg(context_location_latitude) FROM meergo_events"},
		{name: "min max", sql: "SELECT min(received_at), max(received_at) FROM meergo_events"},
		{name: "date trunc", sql: "SELECT date_trunc('day', received_at) FROM meergo_events"},
		{name: "extract", sql: "SELECT extract(year FROM received_at) FROM meergo_events"},
		{name: "date part", sql: "SELECT date_part('month', received_at) FROM meergo_events"},
		{name: "string agg", sql: "SELECT string_agg(event, ',') FROM meergo_events"},
		{name: "array length", sql: "SELECT array_length(_identities, 1) FROM meergo_profiles_5"},
		{name: "cardinality", sql: "SELECT cardinality(preferences_categories) FROM meergo_profiles_5"},
		{name: "unnest", sql: "SELECT unnest(preferences_categories) FROM meergo_profiles_5"},
		{name: "array subquery", sql: "SELECT ARRAY(SELECT id FROM t)"},
		{name: "array literal", sql: "SELECT ARRAY[1, 2, 3]"},
		{name: "cast int", sql: "SELECT CAST(x AS int) FROM t"},
		{name: "cast text", sql: "SELECT CAST(amount AS text) FROM orders"},
		{name: "filter clause", sql: "SELECT count(*) FILTER (WHERE x > 0) FROM t"},
		{name: "filter clause with sum", sql: "SELECT sum(amount) FILTER (WHERE status = 'paid') FROM orders"},
		{name: "jsonb extract path text", sql: "SELECT jsonb_extract_path_text(properties, 'foo') FROM meergo_events"},
		{name: "jsonb array length", sql: "SELECT jsonb_array_length(properties->'items') FROM meergo_events"},
		{name: "to char", sql: "SELECT to_char(received_at, 'YYYY-MM-DD') FROM meergo_events"},
		{name: "now", sql: "SELECT now()"},
		{name: "now in where", sql: "SELECT * FROM meergo_events WHERE received_at > now() - interval '7 days'"},
		{name: "lower with spaces", sql: "SELECT lower    ('ABC')"},
		{name: "count with spaces", sql: "SELECT count (*) FROM meergo_events"},
		{name: "date trunc with spaces", sql: "SELECT date_trunc ('day', received_at) FROM meergo_events"},
		{name: "lower in string", sql: "SELECT 'lower('"},
		{name: "setval in dollar quote", sql: "SELECT $$setval(x)$$"},
		{name: "quoted lower identifier", sql: `SELECT "lower" FROM t`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}
}

// TestValidateReadOnlyFunctionsRejected verifies rejected function and built-in
// calls.
func TestValidateReadOnlyFunctionsRejected(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr string
		want    string
	}{
		{name: "nextval", sql: "SELECT nextval('seq')", wantErr: "rejected: function or built-in nextval is not allowed in read-only queries", want: "nextval"},
		{name: "setval", sql: "SELECT setval('seq', 10)", wantErr: "rejected: function or built-in setval is not allowed in read-only queries", want: "setval"},
		{name: "pg notify", sql: "SELECT pg_notify('ch', 'msg')", wantErr: "rejected: function or built-in pg_notify is not allowed in read-only queries", want: "pg_notify"},
		{name: "pg advisory lock", sql: "SELECT pg_advisory_lock(1)", wantErr: "rejected: function or built-in pg_advisory_lock is not allowed in read-only queries", want: "pg_advisory_lock"},
		{name: "resolve identities", sql: "SELECT resolve_identities(1)", wantErr: "rejected: function or built-in resolve_identities is not allowed in read-only queries", want: "resolve_identities"},
		{name: "unknown name", sql: "SELECT unknown_name(1)", wantErr: "rejected: function or built-in unknown_name is not allowed in read-only queries", want: "unknown_name"},
		{name: "mixed case unknown name", sql: "SELECT UnKnOwN_NaMe(1)", wantErr: "rejected: function or built-in unknown_name is not allowed in read-only queries", want: "unknown_name"},
		{name: "text", sql: "SELECT text(42)", wantErr: "rejected: function or built-in text is not allowed in read-only queries", want: "text"},
		{name: "current setting", sql: "SELECT current_setting('search_path')", wantErr: "rejected: function or built-in current_setting is not allowed in read-only queries", want: "current_setting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			assertExactError(t, err, tt.wantErr)
			assertFunctionNotAllowedError(t, err, tt.want)
		})
	}
}

// TestValidateReadOnlyQualifiedFunctionsRejected verifies rejection of
// qualified function calls.
func TestValidateReadOnlyQualifiedFunctionsRejected(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "pg catalog lower", sql: "SELECT pg_catalog.lower('ABC')"},
		{name: "quoted schema lower", sql: `SELECT "pg_catalog".lower('ABC')`},
		{name: "quoted schema lower with spaces", sql: `SELECT "pg_catalog" . lower('ABC')`},
		{name: "quoted schema lower with comments", sql: `SELECT "pg_catalog"/*x*/./*y*/lower('ABC')`},
		{name: "public resolve identities", sql: "SELECT public.resolve_identities(1)"},
		{name: "schema function", sql: "SELECT my_schema.my_function(1)"},
		{name: "table alias style", sql: "SELECT a.lower('ABC')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			if err == nil {
				t.Fatalf("ValidateReadOnly(%q) returned nil error", tt.sql)
			}
			if !strings.Contains(err.Error(), "qualified function call") {
				t.Fatalf("ValidateReadOnly(%q) error = %q, want qualified-function rejection", tt.sql, err.Error())
			}
			assertNoRejectedFunctionError(t, err)
		})
	}
}

// TestValidateReadOnlySpecialForms verifies accepted and rejected PostgreSQL
// special forms.
func TestValidateReadOnlySpecialForms(t *testing.T) {
	allowed := []struct {
		name string
		sql  string
	}{
		{name: "current date", sql: "SELECT CURRENT_DATE"},
		{name: "current time", sql: "SELECT CURRENT_TIME"},
		{name: "current timestamp", sql: "SELECT CURRENT_TIMESTAMP"},
		{name: "localtime", sql: "SELECT LOCALTIME"},
		{name: "localtimestamp", sql: "SELECT LOCALTIMESTAMP"},
		{name: "lowercase current date", sql: "SELECT current_date"},
		{name: "lowercase current timestamp", sql: "SELECT current_timestamp"},
		{name: "current time precision", sql: "SELECT CURRENT_TIME(0)"},
		{name: "current timestamp precision", sql: "SELECT CURRENT_TIMESTAMP ( 3 )"},
		{name: "localtime precision", sql: "SELECT LOCALTIME(2)"},
		{name: "localtimestamp precision", sql: "SELECT LOCALTIMESTAMP (6)"},
		{name: "special form in string", sql: "SELECT 'CURRENT_TIMESTAMP'"},
		{name: "special form quoted identifier", sql: `SELECT "CURRENT_TIMESTAMP"`},
		{name: "special form substring in identifier", sql: "SELECT my_current_timestamp_value FROM t"},
	}

	for _, tt := range allowed {
		t.Run("allow/"+tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}

	rejected := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{name: "current user", sql: "SELECT CURRENT_USER", wantErr: "rejected: special form CURRENT_USER is not allowed"},
		{name: "session user", sql: "SELECT SESSION_USER", wantErr: "rejected: special form SESSION_USER is not allowed"},
		{name: "user", sql: "SELECT USER", wantErr: "rejected: special form USER is not allowed"},
		{name: "current role", sql: "SELECT CURRENT_ROLE", wantErr: "rejected: special form CURRENT_ROLE is not allowed"},
		{name: "current schema", sql: "SELECT CURRENT_SCHEMA", wantErr: "rejected: special form CURRENT_SCHEMA is not allowed"},
		{name: "current catalog", sql: "SELECT CURRENT_CATALOG", wantErr: "rejected: special form CURRENT_CATALOG is not allowed"},
		{name: "malformed precision", sql: "SELECT CURRENT_TIMESTAMP(abc)", wantErr: "rejected: malformed precision for special form CURRENT_TIMESTAMP"},
		{name: "current date with parens", sql: "SELECT CURRENT_DATE()", wantErr: "rejected: special form CURRENT_DATE does not allow parentheses"},
	}

	for _, tt := range rejected {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			assertExactError(t, err, tt.wantErr)
			assertNoRejectedFunctionError(t, err)
		})
	}
}

// TestValidateReadOnlyMixedCases verifies representative combinations of
// accepted and rejected constructs.
func TestValidateReadOnlyMixedCases(t *testing.T) {
	acceptTests := []struct {
		name string
		sql  string
	}{
		{name: "current date and lower", sql: "SELECT CURRENT_DATE, lower('x')"},
		{name: "with lower and current timestamp", sql: "WITH a AS (SELECT lower('x'), CURRENT_TIMESTAMP) SELECT * FROM a"},
		{name: "date trunc and current date", sql: "SELECT date_trunc('day', received_at), CURRENT_DATE FROM meergo_events"},
	}

	for _, tt := range acceptTests {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}

	t.Run("reject/disallowed function in with", func(t *testing.T) {
		err := ValidateReadOnly("WITH a AS (SELECT nextval('seq')) SELECT * FROM a")
		assertExactError(t, err, "rejected: function or built-in nextval is not allowed in read-only queries")
		assertFunctionNotAllowedError(t, err, "nextval")
	})

	t.Run("reject/current timestamp multistatement", func(t *testing.T) {
		err := ValidateReadOnly("SELECT CURRENT_TIMESTAMP; UPDATE t SET x = 1")
		assertExactError(t, err, "rejected: semicolon found outside opaque region")
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("reject/current user plus lower", func(t *testing.T) {
		err := ValidateReadOnly("SELECT CURRENT_USER, lower('x')")
		assertExactError(t, err, "rejected: special form CURRENT_USER is not allowed")
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("reject/qualified lower plus current date", func(t *testing.T) {
		err := ValidateReadOnly("SELECT pg_catalog.lower('x'), CURRENT_DATE")
		if err == nil {
			t.Fatal("ValidateReadOnly returned nil error")
		}
		if !strings.Contains(err.Error(), "qualified function call") {
			t.Fatalf("error = %q, want qualified-function rejection", err.Error())
		}
		assertNoRejectedFunctionError(t, err)
	})
}

// TestValidateReadOnlyWhitespace verifies that all PostgreSQL whitespace
// characters (space, tab, newline, carriage return, form feed, vertical tab)
// are correctly recognised as token separators.
func TestValidateReadOnlyWhitespace(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "tab between tokens", sql: "SELECT\t1"},
		{name: "newline between tokens", sql: "SELECT\n1"},
		{name: "carriage return between tokens", sql: "SELECT\r1"},
		{name: "form feed between tokens", sql: "SELECT\f1"},
		{name: "vertical tab between tokens", sql: "SELECT\v1"},
		{name: "mixed whitespace", sql: "SELECT\t\n\r\f\v1"},
		{name: "leading whitespace", sql: "\t\n\r\f\v SELECT 1"},
		{name: "trailing whitespace", sql: "SELECT 1\t\n\r\f\v"},
		{name: "whitespace around function call", sql: "SELECT\tlower\t(\t'ABC'\t)"},
		{name: "vertical tab around function call", sql: "SELECT\vlower\v(\v'ABC'\v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}
}

// TestValidateReadOnlyIdentifierLength verifies conservative rejection of
// identifiers longer than PostgreSQL's default 63-byte limit.
func TestValidateReadOnlyIdentifierLength(t *testing.T) {
	t.Run("accept/unquoted at limit", func(t *testing.T) {
		mustAcceptSQL(t, "SELECT "+strings.Repeat("a", 63))
	})

	t.Run("reject/unquoted over limit", func(t *testing.T) {
		err := ValidateReadOnly("SELECT " + strings.Repeat("a", 64))
		assertExactError(t, err, "rejected: identifier exceeds 63 bytes")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted at limit", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "`+strings.Repeat("a", 63)+`"`)
	})

	t.Run("reject/quoted over limit", func(t *testing.T) {
		err := ValidateReadOnly(`SELECT "` + strings.Repeat("a", 64) + `"`)
		assertExactError(t, err, "rejected: identifier exceeds 63 bytes")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted utf8 counted in bytes", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "`+strings.Repeat("é", 31)+`"`)
	})

	t.Run("reject/quoted utf8 over byte limit", func(t *testing.T) {
		err := ValidateReadOnly(`SELECT "` + strings.Repeat("é", 32) + `"`)
		assertExactError(t, err, "rejected: identifier exceeds 63 bytes")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})
}

// TestValidateReadOnlyIdentifierDollarSign verifies that dollar signs are
// rejected in unquoted identifiers but remain acceptable in quoted ones.
func TestValidateReadOnlyIdentifierDollarSign(t *testing.T) {
	t.Run("reject/unquoted dollar sign", func(t *testing.T) {
		err := ValidateReadOnly("SELECT foo$bar FROM t")
		assertExactError(t, err, "rejected: dollar sign is not allowed in unquoted identifiers or outside dollar-quoted strings")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted dollar sign", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "foo$bar" FROM t`)
	})
}

func mustAcceptSQL(t *testing.T, sql string) {
	t.Helper()
	if err := ValidateReadOnly(sql); err != nil {
		t.Fatalf("ValidateReadOnly(%q) returned unexpected error: %v", sql, err)
	}
}

func assertExactError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func assertFunctionNotAllowedError(t *testing.T, err error, wantName string) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.RejectedReadOnlyQueryError, got %T (%v)", err, err)
	}
	if target.Function != wantName {
		t.Fatalf("RejectedReadOnlyQueryError.Function = %q, want %q", target.Function, wantName)
	}
}

func assertRejectedError(t *testing.T, err error) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.RejectedReadOnlyQueryError, got %T (%v)", err, err)
	}
}

func assertNoRejectedFunctionError(t *testing.T, err error) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if errors.As(err, &target) && target.Function != "" {
		t.Fatalf("unexpected rejected function: %+v", target)
	}
}
