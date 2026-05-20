// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package readonlysql

import (
	"errors"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/warehouses"
)

// TestValidateReadOnlyStatements verifies statement-level acceptance and
// rejection.
func TestValidateReadOnlyStatements(t *testing.T) {
	acceptTests := []struct {
		name string
		sql  string
	}{
		{name: "simple select", sql: "SELECT 1"},
		{name: "simple select trailing semicolon", sql: "SELECT 1;"},
		{name: "forbidden token in single quote", sql: "SELECT 'DELETE'"},
		{name: "forbidden token in quoted identifier", sql: `SELECT "DELETE" FROM t`},
		{name: "cast operator in string", sql: "SELECT '::'"},
		{name: "share identifier", sql: "SELECT share FROM t"},
		{name: "unicode prefix separated by operator", sql: `SELECT U & "foo" FROM t`},
		{name: "with select", sql: "WITH a AS (SELECT 1) SELECT * FROM a"},
		{name: "with select column list", sql: "WITH a(x) AS (SELECT 1) SELECT * FROM a"},
		{name: "with recursive column list", sql: "WITH RECURSIVE nums(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM nums WHERE n < 3) SELECT * FROM nums"},
		{name: "from subquery", sql: "SELECT * FROM (SELECT 1) AS t"},
		{name: "join subquery", sql: "SELECT * FROM t JOIN (SELECT 1 AS x) AS s ON TRUE"},
		{name: "lateral subquery", sql: "SELECT * FROM t CROSS JOIN LATERAL (SELECT count(*) AS n) AS x"},
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
		{name: "delete", sql: "DELETE FROM t", wantErr: "query rejected: DELETE is not allowed in read-only queries"},
		{name: "semicolon multi statement", sql: "SELECT 1; DELETE FROM t", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "semicolon with trailing whitespace", sql: "SELECT 1;   \t\n", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "semicolon with trailing line comment", sql: "SELECT 1; -- done", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "semicolon with trailing block comment", sql: "SELECT 1; /* done */", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "semicolon second statement after comment", sql: "SELECT 1; /*x*/ DELETE FROM t", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "double semicolon", sql: "SELECT 1;;", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "select into", sql: "SELECT * INTO new_table FROM t", wantErr: "query rejected: INTO is not allowed in read-only queries"},
		{name: "delete inside with", sql: "WITH a AS (DELETE FROM t RETURNING *) SELECT * FROM a", wantErr: "query rejected: DELETE is not allowed in read-only queries"},
		{name: "insert", sql: "INSERT INTO t VALUES (1)", wantErr: "query rejected: INSERT is not allowed in read-only queries"},
		{name: "update", sql: "UPDATE t SET x = 1", wantErr: "query rejected: UPDATE is not allowed in read-only queries"},
		{name: "create table", sql: "CREATE TABLE x (id int)", wantErr: "query rejected: CREATE is not allowed in read-only queries"},
		{name: "refresh materialized view", sql: "REFRESH MATERIALIZED VIEW mv", wantErr: "query rejected: REFRESH is not allowed in read-only queries"},
		{name: "refresh materialized view concurrently", sql: "REFRESH MATERIALIZED VIEW CONCURRENTLY mv", wantErr: "query rejected: REFRESH is not allowed in read-only queries"},
		{name: "for share", sql: "SELECT * FROM t FOR SHARE", wantErr: "query rejected: locking clause FOR SHARE is not allowed in read-only queries"},
		{name: "for key share", sql: "SELECT * FROM t FOR KEY SHARE", wantErr: "query rejected: locking clause FOR KEY SHARE is not allowed in read-only queries"},
		{name: "for key share with comments", sql: "SELECT * FROM t FOR /*x*/ KEY /*y*/ SHARE", wantErr: "query rejected: locking clause FOR KEY SHARE is not allowed in read-only queries"},
		{name: "builtin cast", sql: "SELECT 1::int", wantErr: "query rejected: the :: type cast syntax is not allowed in read-only queries"},
		{name: "qualified custom cast", sql: "SELECT 'abc'::public.some_type", wantErr: "query rejected: the :: type cast syntax is not allowed in read-only queries"},
		{name: "empty", sql: "", wantErr: "query rejected: a read-only SELECT query is required"},
		{name: "comment only", sql: "-- comment only", wantErr: "query rejected: a read-only SELECT query is required"},
		{name: "unterminated block comment", sql: "SELECT 1 /* unterminated", wantErr: "query rejected: unterminated block comment"},
		{name: "unterminated single quoted string", sql: "SELECT 'unterminated", wantErr: "query rejected: unterminated single-quoted string"},
		{name: "forbidden token in dollar quote", sql: "SELECT $$DELETE$$", wantErr: "query rejected: dollar-quoted strings are not allowed in read-only queries"},
		{name: "cast operator in dollar quote", sql: "SELECT $$::$$", wantErr: "query rejected: dollar-quoted strings are not allowed in read-only queries"},
		{name: "unterminated dollar quoted string", sql: "SELECT $$unterminated", wantErr: "query rejected: dollar-quoted strings are not allowed in read-only queries"},
		{name: "nul in quoted identifier", sql: "SELECT \"a\x00b\" FROM t", wantErr: "query rejected: double-quoted identifier contains NUL byte"},
		{name: "unicode quoted identifier", sql: `SELECT U&"d\0061t\+000061" FROM t`, wantErr: `query rejected: Unicode quoted identifiers are not allowed in read-only queries`},
		{name: "unicode quoted identifier lowercase", sql: `SELECT u&"d\0061t\+000061" FROM t`, wantErr: `query rejected: Unicode quoted identifiers are not allowed in read-only queries`},
		{name: "unterminated unicode quoted identifier", sql: `SELECT U&"unterminated`, wantErr: `query rejected: Unicode quoted identifiers are not allowed in read-only queries`},
		{name: "unicode escape string constant", sql: `SELECT U&'d\0061t\+000061'`, wantErr: `query rejected: Unicode escape strings are not allowed in read-only queries`},
		{name: "unicode escape string constant lowercase", sql: `SELECT u&'d\0061t\+000061'`, wantErr: `query rejected: Unicode escape strings are not allowed in read-only queries`},
		{name: "unterminated unicode escape string constant", sql: `SELECT U&'unterminated`, wantErr: `query rejected: Unicode escape strings are not allowed in read-only queries`},
		{name: "escape string constant", sql: "SELECT E'foo'", wantErr: "query rejected: escape strings are not allowed in read-only queries"},
		{name: "escape string constant lowercase", sql: "SELECT e'foo'", wantErr: "query rejected: escape strings are not allowed in read-only queries"},
		{name: "bit string constant", sql: "SELECT B'1010'", wantErr: "query rejected: bit strings are not allowed in read-only queries"},
		{name: "bit string constant lowercase", sql: "SELECT b'1010'", wantErr: "query rejected: bit strings are not allowed in read-only queries"},
		{name: "unterminated bit string constant", sql: "SELECT B'101", wantErr: "query rejected: bit strings are not allowed in read-only queries"},
		{name: "hex string constant", sql: "SELECT X'1f'", wantErr: "query rejected: hex strings are not allowed in read-only queries"},
		{name: "hex string constant lowercase", sql: "SELECT x'1f'", wantErr: "query rejected: hex strings are not allowed in read-only queries"},
		{name: "unterminated hex string constant", sql: "SELECT X'1", wantErr: "query rejected: hex strings are not allowed in read-only queries"},
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
		{name: "json extract path text", sql: "SELECT json_extract_path_text(CAST(properties AS json), 'foo') FROM meergo_events"},
		{name: "json array length", sql: "SELECT json_array_length(CAST(properties->'items' AS json)) FROM meergo_events"},
		{name: "json build object", sql: "SELECT json_build_object('event', event, 'received_at', received_at) FROM meergo_events"},
		{name: "json object agg", sql: "SELECT json_object_agg(event, received_at) FROM meergo_events"},
		{name: "json object keys", sql: "SELECT json_object_keys(CAST(properties AS json)) FROM meergo_events"},
		{name: "json typeof", sql: "SELECT json_typeof(CAST(properties AS json)) FROM meergo_events"},
		{name: "jsonb extract path text", sql: "SELECT jsonb_extract_path_text(properties, 'foo') FROM meergo_events"},
		{name: "jsonb array length", sql: "SELECT jsonb_array_length(properties->'items') FROM meergo_events"},
		{name: "jsonb build object", sql: "SELECT jsonb_build_object('event', event, 'received_at', received_at) FROM meergo_events"},
		{name: "jsonb object agg", sql: "SELECT jsonb_object_agg(event, received_at) FROM meergo_events"},
		{name: "to char", sql: "SELECT to_char(received_at, 'YYYY-MM-DD') FROM meergo_events"},
		{name: "now", sql: "SELECT now()"},
		{name: "now in where", sql: "SELECT * FROM meergo_events WHERE received_at > now() - interval '7 days'"},
		{name: "row number window", sql: "SELECT row_number() OVER (ORDER BY received_at) FROM meergo_events"},
		{name: "over clause not security validated", sql: "SELECT 1 OVER (ORDER BY x)"},
		{name: "from lateral information schema query", sql: "SELECT t.table_name, x.column_count FROM (SELECT DISTINCT table_name FROM information_schema.columns WHERE table_schema = 'public') t CROSS JOIN LATERAL (SELECT COUNT(*) AS column_count FROM information_schema.columns c WHERE c.table_schema = 'public' AND c.table_name = t.table_name) x ORDER BY x.column_count DESC, t.table_name"},
		{name: "lower with spaces", sql: "SELECT lower    ('ABC')"},
		{name: "count with spaces", sql: "SELECT count (*) FROM meergo_events"},
		{name: "date trunc with spaces", sql: "SELECT date_trunc ('day', received_at) FROM meergo_events"},
		{name: "lower in string", sql: "SELECT 'lower('"},
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
		{name: "nextval", sql: "SELECT nextval('seq')", wantErr: "query rejected: function or built-in NEXTVAL is not allowed in read-only queries", want: "nextval"},
		{name: "setval", sql: "SELECT setval('seq', 10)", wantErr: "query rejected: function or built-in SETVAL is not allowed in read-only queries", want: "setval"},
		{name: "pg notify", sql: "SELECT pg_notify('ch', 'msg')", wantErr: "query rejected: function or built-in PG_NOTIFY is not allowed in read-only queries", want: "pg_notify"},
		{name: "pg advisory lock", sql: "SELECT pg_advisory_lock(1)", wantErr: "query rejected: function or built-in PG_ADVISORY_LOCK is not allowed in read-only queries", want: "pg_advisory_lock"},
		{name: "resolve identities", sql: "SELECT resolve_identities(1)", wantErr: "query rejected: function or built-in RESOLVE_IDENTITIES is not allowed in read-only queries", want: "resolve_identities"},
		{name: "unknown name", sql: "SELECT unknown_name(1)", wantErr: "query rejected: function or built-in UNKNOWN_NAME is not allowed in read-only queries", want: "unknown_name"},
		{name: "mixed case unknown name", sql: "SELECT UnKnOwN_NaMe(1)", wantErr: "query rejected: function or built-in UNKNOWN_NAME is not allowed in read-only queries", want: "unknown_name"},
		{name: "text", sql: "SELECT text(42)", wantErr: "query rejected: function or built-in TEXT is not allowed in read-only queries", want: "text"},
		{name: "current setting", sql: "SELECT current_setting('search_path')", wantErr: "query rejected: function or built-in CURRENT_SETTING is not allowed in read-only queries", want: "current_setting"},
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

// TestValidateReadOnlyIdentifierChains verifies identifier-chain handling that
// is easy to regress when changing name normalization or dotted-name scanning.
func TestValidateReadOnlyIdentifierChains(t *testing.T) {
	t.Run("accept/qualified column reference", func(t *testing.T) {
		mustAcceptSQL(t, "SELECT e.connection_id FROM meergo_events AS e")
	})

	t.Run("accept/qualified column reference with spaces and comments", func(t *testing.T) {
		mustAcceptSQL(t, "SELECT e /*x*/ . /*y*/ connection_id FROM meergo_events AS e")
	})

	t.Run("accept/mixed case allowed function with underscore", func(t *testing.T) {
		mustAcceptSQL(t, "SELECT DaTe_TrUnC('day', received_at) FROM meergo_events")
	})

	t.Run("reject/mixed case function with digits", func(t *testing.T) {
		err := ValidateReadOnly("SELECT AbC123(1)")
		assertExactError(t, err, "query rejected: function or built-in ABC123 is not allowed in read-only queries")
		assertFunctionNotAllowedError(t, err, "abc123")
	})

	t.Run("reject/leading underscore function", func(t *testing.T) {
		err := ValidateReadOnly("SELECT _FoO(1)")
		assertExactError(t, err, "query rejected: function or built-in _FOO is not allowed in read-only queries")
		assertFunctionNotAllowedError(t, err, "_foo")
	})

	t.Run("reject/qualified mixed case exact name", func(t *testing.T) {
		err := ValidateReadOnly("SELECT Pg_Catalog.LoWeR('ABC')")
		assertExactError(t, err, "query rejected: schema-qualified function call PG_CATALOG.LOWER is not allowed in read-only queries")
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("reject/multi part qualified exact name", func(t *testing.T) {
		err := ValidateReadOnly("SELECT A.B.C(1)")
		assertExactError(t, err, "query rejected: schema-qualified function call A.B.C is not allowed in read-only queries")
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("reject/multi part qualified with spaces and comments", func(t *testing.T) {
		err := ValidateReadOnly("SELECT A /*x*/ . /*y*/ B /*z*/ . /*w*/ C(1)")
		assertExactError(t, err, "query rejected: schema-qualified function call A.B.C is not allowed in read-only queries")
		assertNoRejectedFunctionError(t, err)
	})
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
		{name: "current user", sql: "SELECT CURRENT_USER", wantErr: "query rejected: CURRENT_USER is not allowed in read-only queries"},
		{name: "session user", sql: "SELECT SESSION_USER", wantErr: "query rejected: SESSION_USER is not allowed in read-only queries"},
		{name: "user", sql: "SELECT USER", wantErr: "query rejected: USER is not allowed in read-only queries"},
		{name: "current role", sql: "SELECT CURRENT_ROLE", wantErr: "query rejected: CURRENT_ROLE is not allowed in read-only queries"},
		{name: "current schema", sql: "SELECT CURRENT_SCHEMA", wantErr: "query rejected: CURRENT_SCHEMA is not allowed in read-only queries"},
		{name: "current catalog", sql: "SELECT CURRENT_CATALOG", wantErr: "query rejected: CURRENT_CATALOG is not allowed in read-only queries"},
		{name: "malformed precision", sql: "SELECT CURRENT_TIMESTAMP(abc)", wantErr: "query rejected: invalid precision for CURRENT_TIMESTAMP in read-only queries"},
		{name: "current date with parens", sql: "SELECT CURRENT_DATE()", wantErr: "query rejected: CURRENT_DATE with parentheses is not allowed in read-only queries"},
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
		{name: "current date trailing semicolon", sql: "SELECT CURRENT_DATE;"},
	}

	for _, tt := range acceptTests {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}

	t.Run("reject/disallowed function in with", func(t *testing.T) {
		err := ValidateReadOnly("WITH a AS (SELECT nextval('seq')) SELECT * FROM a")
		assertExactError(t, err, "query rejected: function or built-in NEXTVAL is not allowed in read-only queries")
		assertFunctionNotAllowedError(t, err, "nextval")
	})

	t.Run("reject/disallowed function in recursive with column list", func(t *testing.T) {
		err := ValidateReadOnly("WITH RECURSIVE nums(n) AS (SELECT nextval('seq')) SELECT * FROM nums")
		assertExactError(t, err, "query rejected: function or built-in NEXTVAL is not allowed in read-only queries")
		assertFunctionNotAllowedError(t, err, "nextval")
	})

	t.Run("reject/current timestamp multistatement", func(t *testing.T) {
		err := ValidateReadOnly("SELECT CURRENT_TIMESTAMP; UPDATE t SET x = 1")
		assertExactError(t, err, "query rejected: multiple statements are not allowed in read-only queries")
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("reject/current user plus lower", func(t *testing.T) {
		err := ValidateReadOnly("SELECT CURRENT_USER, lower('x')")
		assertExactError(t, err, "query rejected: CURRENT_USER is not allowed in read-only queries")
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
		assertExactError(t, err, "query rejected: identifiers longer than 63 bytes are not allowed in read-only queries")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted at limit", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "`+strings.Repeat("a", 63)+`"`)
	})

	t.Run("reject/quoted over limit", func(t *testing.T) {
		err := ValidateReadOnly(`SELECT "` + strings.Repeat("a", 64) + `"`)
		assertExactError(t, err, "query rejected: identifiers longer than 63 bytes are not allowed in read-only queries")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted utf8 counted in bytes", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "`+strings.Repeat("é", 31)+`"`)
	})

	t.Run("reject/quoted utf8 over byte limit", func(t *testing.T) {
		err := ValidateReadOnly(`SELECT "` + strings.Repeat("é", 32) + `"`)
		assertExactError(t, err, "query rejected: identifiers longer than 63 bytes are not allowed in read-only queries")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})
}

// TestValidateReadOnlyIdentifierDollarSign verifies that dollar signs are
// rejected in SQL text but remain acceptable in quoted identifiers.
func TestValidateReadOnlyIdentifierDollarSign(t *testing.T) {
	t.Run("reject/unquoted dollar sign", func(t *testing.T) {
		err := ValidateReadOnly("SELECT foo$bar FROM t")
		assertExactError(t, err, "query rejected: dollar-quoted strings are not allowed in read-only queries")
		assertRejectedError(t, err)
		assertNoRejectedFunctionError(t, err)
	})

	t.Run("accept/quoted dollar sign", func(t *testing.T) {
		mustAcceptSQL(t, `SELECT "foo$bar" FROM t`)
	})
}

// TestValidateReadOnlySuccessPathAllocs verifies that representative accepted
// queries do not allocate in the validator success path.
func TestValidateReadOnlySuccessPathAllocs(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "allowed function simple", sql: "SELECT lower('x')"},
		{name: "allowed function mixed case underscore", sql: "SELECT DaTe_TrUnC('day', received_at) FROM meergo_events"},
		{name: "qualified column reference", sql: "SELECT e.connection_id FROM meergo_events AS e"},
		{name: "qualified column reference with comments", sql: "SELECT e /*x*/ . /*y*/ connection_id FROM meergo_events AS e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(1000, func() {
				if err := ValidateReadOnly(tt.sql); err != nil {
					t.Fatalf("ValidateReadOnly(%q) returned unexpected error: %v", tt.sql, err)
				}
			})
			if allocs != 0 {
				t.Fatalf("ValidateReadOnly(%q) allocated %.0f times, want 0", tt.sql, allocs)
			}
		})
	}
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
