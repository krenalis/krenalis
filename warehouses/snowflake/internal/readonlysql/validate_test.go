// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import (
	"errors"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/warehouses"
)

// TestValidateReadOnlyStatements verifies statement-level policy.
func TestValidateReadOnlyStatements(t *testing.T) {
	acceptTests := []struct {
		name string
		sql  string
	}{
		{name: "simple select", sql: "SELECT 1"},
		{name: "simple select trailing semicolon", sql: "SELECT 1;"},
		{name: "simple select trailing semicolon whitespace", sql: "SELECT 1;   \t\n"},
		{name: "simple select trailing semicolon comment", sql: "SELECT 1; -- done"},
		{name: "with select", sql: "WITH a AS (SELECT 1) SELECT * FROM a"},
		{name: "with select column list", sql: "WITH a(x) AS (SELECT 1) SELECT * FROM a"},
		{name: "with multiple ctes", sql: "WITH a AS (SELECT 1), b(x) AS (SELECT 2) SELECT * FROM a UNION ALL SELECT * FROM b"},
		{name: "from subquery", sql: "SELECT * FROM (SELECT 1) AS t"},
		{name: "join subquery", sql: "SELECT * FROM t JOIN (SELECT 1 AS x) AS s ON TRUE"},
		{name: "aggregate group order limit", sql: "SELECT event, count(*) FROM meergo_events WHERE received_at >= current_date GROUP BY event ORDER BY 2 DESC LIMIT 10"},
		{name: "qualify", sql: "SELECT customer_id, sum(amount) OVER (PARTITION BY customer_id) AS total FROM orders QUALIFY total > 100"},
		{name: "set operation", sql: "SELECT id FROM a UNION ALL SELECT id FROM b"},
		{name: "case expression", sql: "SELECT CASE WHEN type = 'track' THEN 1 ELSE 0 END FROM meergo_events"},
		{name: "forbidden token in single quote", sql: "SELECT 'DELETE'"},
		{name: "escaped quote in single quote", sql: "SELECT 'it''s DELETE'"},
		{name: "forbidden token in quoted identifier", sql: `SELECT "DELETE" FROM t`},
		{name: "escaped quote in quoted identifier", sql: `SELECT "a""b" FROM t`},
		{name: "nested block comment", sql: "SELECT 1 /* outer /* inner */ still comment */"},
		{name: "stage marker in single quote", sql: "SELECT '@stage'"},
		{name: "session variable marker in quoted identifier", sql: `SELECT "$session_table" FROM t`},
		{name: "visible dollar inside identifier", sql: "SELECT my$identifier FROM t"},
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
		{name: "empty", sql: "", wantErr: "query rejected: a read-only SELECT query is required"},
		{name: "comment only", sql: "-- comment only", wantErr: "query rejected: a read-only SELECT query is required"},
		{name: "unsupported top level", sql: "FETCH 1", wantErr: "query rejected: top-level FETCH statements are not allowed in read-only queries"},
		{name: "top level values", sql: "VALUES (1)", wantErr: "query rejected: top-level VALUES statements are not allowed in read-only queries"},
		{name: "with without main select", sql: "WITH a AS (SELECT 1)", wantErr: "query rejected: a read-only SELECT query is required"},
		{name: "with values body", sql: "WITH a AS (SELECT 1) VALUES (1)", wantErr: "query rejected: top-level VALUES statements are not allowed in read-only queries"},
		{name: "with forbidden cte name", sql: "WITH delete AS (SELECT 1) SELECT * FROM delete", wantErr: "query rejected: DELETE is not allowed in read-only queries"},
		{name: "delete", sql: "DELETE FROM t", wantErr: "query rejected: DELETE is not allowed in read-only queries"},
		{name: "insert select", sql: "INSERT INTO t SELECT 1", wantErr: "query rejected: INSERT is not allowed in read-only queries"},
		{name: "update", sql: "UPDATE t SET x = 1", wantErr: "query rejected: UPDATE is not allowed in read-only queries"},
		{name: "merge", sql: "MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN UPDATE SET x = s.x", wantErr: "query rejected: MERGE is not allowed in read-only queries"},
		{name: "create table as select", sql: "CREATE TABLE x AS SELECT 1", wantErr: "query rejected: CREATE is not allowed in read-only queries"},
		{name: "drop", sql: "DROP TABLE t", wantErr: "query rejected: DROP is not allowed in read-only queries"},
		{name: "grant", sql: "GRANT SELECT ON TABLE t TO ROLE r", wantErr: "query rejected: GRANT is not allowed in read-only queries"},
		{name: "truncate", sql: "TRUNCATE TABLE t", wantErr: "query rejected: TRUNCATE is not allowed in read-only queries"},
		{name: "copy into stage", sql: "COPY INTO @stage FROM (SELECT 1)", wantErr: "query rejected: COPY is not allowed in read-only queries"},
		{name: "put", sql: "PUT file://rows.csv @stage", wantErr: "query rejected: PUT is not allowed in read-only queries"},
		{name: "get command", sql: "GET @stage file:///tmp", wantErr: "query rejected: GET is not allowed in read-only queries"},
		{name: "list", sql: "LIST @stage", wantErr: "query rejected: LIST is not allowed in read-only queries"},
		{name: "ls", sql: "LS @stage", wantErr: "query rejected: LS is not allowed in read-only queries"},
		{name: "remove", sql: "REMOVE @stage/path", wantErr: "query rejected: REMOVE is not allowed in read-only queries"},
		{name: "rm", sql: "RM @stage/path", wantErr: "query rejected: RM is not allowed in read-only queries"},
		{name: "call", sql: "CALL proc()", wantErr: "query rejected: CALL is not allowed in read-only queries"},
		{name: "use role", sql: "USE ROLE analyst", wantErr: "query rejected: USE is not allowed in read-only queries"},
		{name: "alter session", sql: "ALTER SESSION SET QUOTED_IDENTIFIERS_IGNORE_CASE = TRUE", wantErr: "query rejected: ALTER is not allowed in read-only queries"},
		{name: "set", sql: "SET x = 1", wantErr: "query rejected: SET is not allowed in read-only queries"},
		{name: "begin", sql: "BEGIN SELECT 1; END", wantErr: "query rejected: BEGIN is not allowed in read-only queries"},
		{name: "declare", sql: "DECLARE x INT", wantErr: "query rejected: DECLARE is not allowed in read-only queries"},
		{name: "execute immediate", sql: "EXECUTE IMMEDIATE 'SELECT 1'", wantErr: "query rejected: EXECUTE is not allowed in read-only queries"},
		{name: "show", sql: "SHOW TABLES", wantErr: "query rejected: SHOW is not allowed in read-only queries"},
		{name: "describe", sql: "DESCRIBE TABLE t", wantErr: "query rejected: DESCRIBE is not allowed in read-only queries"},
		{name: "desc", sql: "DESC TABLE t", wantErr: "query rejected: DESC is not allowed in read-only queries"},
		{name: "explain", sql: "EXPLAIN SELECT 1", wantErr: "query rejected: EXPLAIN is not allowed in read-only queries"},
		{name: "semicolon multi statement", sql: "SELECT 1; DELETE FROM t", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "double semicolon", sql: "SELECT 1;;", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "semicolon second statement after comment", sql: "SELECT 1; /*x*/ DELETE FROM t", wantErr: "query rejected: multiple statements are not allowed in read-only queries"},
		{name: "unterminated block comment", sql: "SELECT 1 /* unterminated", wantErr: "query rejected: unterminated block comment"},
		{name: "unterminated single quoted string", sql: "SELECT 'unterminated", wantErr: "query rejected: unterminated single-quoted string"},
		{name: "unterminated double quoted identifier", sql: `SELECT "unterminated`, wantErr: "query rejected: unterminated double-quoted identifier"},
		{name: "unquoted non ascii identifier", sql: "SELECT идентификатор FROM t", wantErr: "query rejected: non-ASCII characters outside strings, comments, and quoted identifiers are not allowed in read-only queries"},
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

// TestValidateReadOnlyIdentifierChains verifies Snowflake identifier forms.
func TestValidateReadOnlyIdentifierChains(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "qualified column reference", sql: "SELECT e.connection_id FROM meergo_events AS e"},
		{name: "qualified column reference with comments", sql: "SELECT e /*x*/ . /*y*/ connection_id FROM meergo_events AS e"},
		{name: "unquoted identifiers", sql: "SELECT _my_identifier, MyIdentifier1, my$identifier FROM t"},
		{name: "quoted identifiers", sql: `SELECT "3rd_identifier", "my identifier", "My 'Identifier'", "$Identifier", "идентификатор" FROM t`},
		{name: "period inside quoted identifier", sql: `SELECT "my.identifier" FROM t`},
		{name: "qualified quoted object name", sql: `SELECT * FROM "My.DB"."My.Schema"."Table.1"`},
		{name: "qualified quoted object name with comments", sql: `SELECT * FROM "My.DB" /*x*/ . /*y*/ "My.Schema" . "Table.1"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
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
		{name: "coalesce", sql: "SELECT coalesce(NULL, 1)"},
		{name: "count", sql: "SELECT count(*) FROM meergo_events"},
		{name: "count if", sql: "SELECT count_if(type = 'track') FROM meergo_events"},
		{name: "approx count distinct", sql: "SELECT approx_count_distinct(customer_id) FROM orders"},
		{name: "sum", sql: "SELECT sum(connection_id) FROM meergo_events"},
		{name: "avg", sql: "SELECT avg(context_location_latitude) FROM meergo_events"},
		{name: "median", sql: "SELECT median(context_screen_width) FROM meergo_events"},
		{name: "mode", sql: "SELECT mode(context_os_name) FROM meergo_events"},
		{name: "min max", sql: "SELECT min(received_at), max(received_at) FROM meergo_events"},
		{name: "datediff", sql: "SELECT datediff('day', created_at, current_date) FROM orders"},
		{name: "dateadd", sql: "SELECT dateadd('day', 7, created_at) FROM orders"},
		{name: "date trunc", sql: "SELECT date_trunc('day', received_at) FROM meergo_events"},
		{name: "date part", sql: "SELECT date_part('month', received_at) FROM meergo_events"},
		{name: "extract", sql: "SELECT extract(year FROM received_at) FROM meergo_events"},
		{name: "listagg", sql: "SELECT listagg(event, ',') FROM meergo_events"},
		{name: "listagg within group", sql: "SELECT listagg(event, ',') WITHIN GROUP (ORDER BY event) FROM meergo_events"},
		{name: "listagg within group with comment", sql: "SELECT listagg(event, ',') WITHIN /*x*/ GROUP (ORDER BY event) FROM meergo_events"},
		{name: "percentile cont within group", sql: "SELECT percentile_cont(0.9) WITHIN GROUP (ORDER BY context_screen_width) FROM meergo_events"},
		{name: "string functions", sql: "SELECT concat_ws('-', lower(first_name), upper(last_name)) FROM meergo_profiles_5"},
		{name: "semi structured get", sql: "SELECT get(properties, 'campaign') FROM meergo_events"},
		{name: "semi structured path", sql: "SELECT get_path(properties, 'campaign.source') FROM meergo_events"},
		{name: "array size", sql: "SELECT array_size(preferences_categories) FROM meergo_profiles_5"},
		{name: "object construct", sql: "SELECT object_construct('event', event, 'type', type) FROM meergo_events"},
		{name: "typeof", sql: "SELECT typeof(properties) FROM meergo_events"},
		{name: "json extract path text", sql: "SELECT json_extract_path_text(properties, 'campaign') FROM meergo_events"},
		{name: "current date", sql: "SELECT current_date"},
		{name: "current timestamp", sql: "SELECT current_timestamp()"},
		{name: "cast", sql: "SELECT CAST(amount AS NUMBER) FROM orders"},
		{name: "window over", sql: "SELECT sum(amount) OVER (PARTITION BY customer_id) FROM orders"},
		{name: "row number window", sql: "SELECT row_number() OVER (PARTITION BY customer_id ORDER BY received_at) FROM orders"},
		{name: "rank windows", sql: "SELECT rank() OVER (ORDER BY total DESC), dense_rank() OVER (ORDER BY total DESC) FROM orders"},
		{name: "ntile window", sql: "SELECT ntile(4) OVER (ORDER BY received_at) FROM orders"},
		{name: "first last value windows", sql: "SELECT first_value(received_at) OVER (ORDER BY received_at), last_value(received_at) OVER (ORDER BY received_at) FROM orders"},
		{name: "lag lead windows", sql: "SELECT lag(amount) OVER (PARTITION BY customer_id ORDER BY received_at), lead(amount) OVER (PARTITION BY customer_id ORDER BY received_at) FROM orders"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustAcceptSQL(t, tt.sql)
		})
	}
}

// TestValidateReadOnlyFunctionsRejected verifies rejected function calls.
func TestValidateReadOnlyFunctionsRejected(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr string
		want    string
	}{
		{name: "unknown name", sql: "SELECT unknown_name(1)", wantErr: "query rejected: function or built-in UNKNOWN_NAME is not allowed in read-only queries", want: "unknown_name"},
		{name: "mixed case unknown name", sql: "SELECT UnKnOwN_NaMe(1)", wantErr: "query rejected: function or built-in UNKNOWN_NAME is not allowed in read-only queries", want: "unknown_name"},
		{name: "identifier syntax", sql: "SELECT IDENTIFIER('T') FROM t", wantErr: "query rejected: function or built-in IDENTIFIER is not allowed in read-only queries", want: "identifier"},
		{name: "result scan", sql: "SELECT RESULT_SCAN(-1)", wantErr: "query rejected: function or built-in RESULT_SCAN is not allowed in read-only queries", want: "result_scan"},
		{name: "last query id", sql: "SELECT LAST_QUERY_ID()", wantErr: "query rejected: function or built-in LAST_QUERY_ID is not allowed in read-only queries", want: "last_query_id"},
		{name: "try cast", sql: "SELECT TRY_CAST(x AS NUMBER) FROM t", wantErr: "query rejected: function or built-in TRY_CAST is not allowed in read-only queries", want: "try_cast"},
		{name: "current role", sql: "SELECT CURRENT_ROLE()", wantErr: "query rejected: function or built-in CURRENT_ROLE is not allowed in read-only queries", want: "current_role"},
		{name: "current role without parens", sql: "SELECT CURRENT_ROLE", wantErr: "query rejected: function or built-in CURRENT_ROLE is not allowed in read-only queries", want: "current_role"},
		{name: "current user", sql: "SELECT CURRENT_USER()", wantErr: "query rejected: function or built-in CURRENT_USER is not allowed in read-only queries", want: "current_user"},
		{name: "current schema without parens", sql: "SELECT CURRENT_SCHEMA", wantErr: "query rejected: function or built-in CURRENT_SCHEMA is not allowed in read-only queries", want: "current_schema"},
		{name: "external style unknown", sql: "SELECT cortex_complete(prompt) FROM t", wantErr: "query rejected: function or built-in CORTEX_COMPLETE is not allowed in read-only queries", want: "cortex_complete"},
		{name: "group call", sql: "SELECT group(1)", wantErr: "query rejected: function or built-in GROUP is not allowed in read-only queries", want: "group"},
		{name: "within group call without aggregate", sql: "SELECT within group(1)", wantErr: "query rejected: function or built-in GROUP is not allowed in read-only queries", want: "group"},
		{name: "within plus group call", sql: "SELECT within + group(1)", wantErr: "query rejected: function or built-in GROUP is not allowed in read-only queries", want: "group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			assertExactError(t, err, tt.wantErr)
			assertFunctionNotAllowedError(t, err, tt.want)
		})
	}
}

// TestValidateReadOnlyQualifiedFunctionsRejected verifies qualified calls.
func TestValidateReadOnlyQualifiedFunctionsRejected(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "schema lower", sql: "SELECT schema.lower('ABC')"},
		{name: "database schema lower", sql: "SELECT database.schema.lower('ABC')"},
		{name: "quoted schema part lower", sql: `SELECT db."schema".lower('ABC')`},
		{name: "quoted schema lower", sql: `SELECT "schema".lower('ABC')`},
		{name: "quoted schema lower with spaces", sql: `SELECT "schema" . lower('ABC')`},
		{name: "quoted schema lower with comments", sql: `SELECT "schema"/*x*/./*y*/lower('ABC')`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			if err == nil {
				t.Fatalf("expected qualified-function rejection for ValidateReadOnly(%q), got nil", tt.sql)
			}
			if !strings.Contains(err.Error(), "qualified function call") {
				t.Fatalf("expected qualified-function rejection for ValidateReadOnly(%q), got %q", tt.sql, err.Error())
			}
			assertNoRejectedFunctionError(t, err)
		})
	}
}

// TestValidateReadOnlySnowflakeLexicalRejections verifies Snowflake syntax.
func TestValidateReadOnlySnowflakeLexicalRejections(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{name: "stage reference", sql: "SELECT * FROM @stage", wantErr: "query rejected: stage references are not allowed in read-only queries"},
		{name: "stage reference after slash", sql: "SELECT * FROM @stage/path", wantErr: "query rejected: stage references are not allowed in read-only queries"},
		{name: "session variable", sql: "SELECT $session_table FROM t", wantErr: "query rejected: session variable references are not allowed in read-only queries"},
		{name: "numeric session variable", sql: "SELECT $1 FROM t", wantErr: "query rejected: session variable references are not allowed in read-only queries"},
		{name: "dollar quoted string", sql: "SELECT $$DELETE FROM t$$", wantErr: "query rejected: dollar-quoted string constants are not allowed in read-only queries"},
		{name: "unterminated dollar quoted string", sql: "SELECT $$unterminated", wantErr: "query rejected: dollar-quoted string constants are not allowed in read-only queries"},
		{name: "type cast", sql: "SELECT x::NUMBER FROM t", wantErr: "query rejected: the :: type cast syntax is not allowed in read-only queries"},
		{name: "pipe operator", sql: "SELECT 1 ->> SELECT * FROM t", wantErr: "query rejected: pipe command-result processing is not allowed in read-only queries"},
		{name: "table result scan", sql: "SELECT * FROM TABLE(RESULT_SCAN(LAST_QUERY_ID()))", wantErr: "query rejected: TABLE(...) is not allowed in read-only queries"},
		{name: "table literal", sql: "SELECT * FROM TABLE('DB.SCHEMA.TABLE')", wantErr: "query rejected: TABLE(...) is not allowed in read-only queries"},
		{name: "quoted function name", sql: `SELECT "LOWER"('ABC')`, wantErr: "query rejected: function calls with quoted identifiers are not allowed in read-only queries"},
		{name: "nul in quoted identifier", sql: "SELECT \"a\x00b\" FROM t", wantErr: "query rejected: double-quoted identifier contains NUL byte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			assertExactError(t, err, tt.wantErr)
			assertRejectedError(t, err)
			assertNoRejectedFunctionError(t, err)
		})
	}
}

// TestNewFunctionNotAllowedError verifies rejected function errors.
func TestNewFunctionNotAllowedError(t *testing.T) {
	err := newFunctionNotAllowedError("unknown_name")
	assertExactError(t, err, "query rejected: function or built-in UNKNOWN_NAME is not allowed in read-only queries")
	assertFunctionNotAllowedError(t, err, "unknown_name")
}

// TestASCIIWordSetHas verifies ASCII case-insensitive word lookup.
func TestASCIIWordSetHas(t *testing.T) {
	set := newASCIIWordSet("select", "current_timestamp")

	if !set.Has("SELECT") {
		t.Fatalf("expected set.Has(%q) to be true, got false", "SELECT")
	}
	if !set.Has("Current_Timestamp") {
		t.Fatalf("expected set.Has(%q) to be true, got false", "Current_Timestamp")
	}
	if set.Has("insert") {
		t.Fatalf("expected set.Has(%q) to be false, got true", "insert")
	}
}

// mustAcceptSQL fails if sql is rejected.
func mustAcceptSQL(t *testing.T, sql string) {
	t.Helper()
	if err := ValidateReadOnly(sql); err != nil {
		t.Fatalf("expected ValidateReadOnly(%q) to accept query, got %v", sql, err)
	}
}

// assertExactError fails if err does not match want exactly.
func assertExactError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

// assertFunctionNotAllowedError fails if err is not a rejection for wantName.
func assertFunctionNotAllowedError(t *testing.T, err error, wantName string) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.RejectedReadOnlyQueryError, got %T (%v)", err, err)
	}
	if target.Function != wantName {
		t.Fatalf("expected RejectedReadOnlyQueryError.Function %q, got %q", wantName, target.Function)
	}
}

// assertRejectedError fails if err is not a read-only query rejection.
func assertRejectedError(t *testing.T, err error) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if !errors.As(err, &target) {
		t.Fatalf("expected warehouses.RejectedReadOnlyQueryError, got %T (%v)", err, err)
	}
}

// assertNoRejectedFunctionError fails if err carries a rejected function name.
func assertNoRejectedFunctionError(t *testing.T, err error) {
	t.Helper()
	var target *warehouses.RejectedReadOnlyQueryError
	if errors.As(err, &target) && target.Function != "" {
		t.Fatalf("expected no rejected function name, got %+v", target)
	}
}
