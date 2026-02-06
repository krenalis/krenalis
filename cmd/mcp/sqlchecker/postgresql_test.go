// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sqlchecker

import (
	"strings"
	"testing"
)

func TestCheckPostgreSQL(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr string // empty means no error expected
	}{
		// --- Allowed queries ---
		{
			name:  "simple select",
			query: "SELECT 1",
		},
		{
			name:  "select from table",
			query: "SELECT * FROM users",
		},
		{
			name:  "select with where",
			query: "SELECT id, name FROM users WHERE age > 18",
		},
		{
			name:  "select with join",
			query: "SELECT u.id, o.total FROM users u JOIN orders o ON u.id = o.user_id",
		},
		{
			name:  "select with left join",
			query: "SELECT u.id, o.total FROM users u LEFT JOIN orders o ON u.id = o.user_id",
		},
		{
			name:  "select with aggregation",
			query: "SELECT COUNT(*), AVG(age) FROM users GROUP BY city HAVING COUNT(*) > 10",
		},
		{
			name:  "select with subquery",
			query: "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
		},
		{
			name:  "select with CTE",
			query: "WITH active AS (SELECT * FROM users WHERE active = true) SELECT * FROM active",
		},
		{
			name:  "select with UNION",
			query: "SELECT id FROM users UNION SELECT id FROM admins",
		},
		{
			name:  "select with UNION ALL",
			query: "SELECT id FROM users UNION ALL SELECT id FROM admins",
		},
		{
			name:  "select with INTERSECT",
			query: "SELECT id FROM users INTERSECT SELECT id FROM admins",
		},
		{
			name:  "select with EXCEPT",
			query: "SELECT id FROM users EXCEPT SELECT id FROM admins",
		},
		{
			name:  "select with ORDER BY and LIMIT",
			query: "SELECT * FROM users ORDER BY created_at DESC LIMIT 100",
		},
		{
			name:  "select with CASE",
			query: "SELECT CASE WHEN age > 18 THEN 'adult' ELSE 'minor' END FROM users",
		},
		{
			name:  "select with COALESCE",
			query: "SELECT COALESCE(name, 'unknown') FROM users",
		},
		{
			name:  "select with window function",
			query: "SELECT id, ROW_NUMBER() OVER (PARTITION BY city ORDER BY age) FROM users",
		},
		{
			name:  "select with DISTINCT",
			query: "SELECT DISTINCT city FROM users",
		},
		{
			name:  "select with EXISTS subquery",
			query: "SELECT * FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
		},
		{
			name:  "recursive CTE with LIMIT",
			query: "WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 100) SELECT x FROM cnt LIMIT 100",
		},
		{
			name:  "small generate_series",
			query: "SELECT * FROM generate_series(1, 1000)",
		},
		{
			name:  "generate_series with non-literal args",
			query: "SELECT * FROM generate_series(1, (SELECT max(n) FROM params))",
		},
		{
			name:  "select with type cast",
			query: "SELECT CAST(id AS TEXT) FROM users",
		},
		{
			name:  "select with multiple CTEs",
			query: "WITH a AS (SELECT 1 AS x), b AS (SELECT 2 AS y) SELECT * FROM a, b WHERE a.x = b.y",
		},

		// --- Disallowed statements ---
		{
			name:    "INSERT",
			query:   "INSERT INTO users (name) VALUES ('test')",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "UPDATE",
			query:   "UPDATE users SET name = 'test' WHERE id = 1",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "DELETE",
			query:   "DELETE FROM users WHERE id = 1",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "DROP TABLE",
			query:   "DROP TABLE users",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "CREATE TABLE",
			query:   "CREATE TABLE test (id INT)",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "ALTER TABLE",
			query:   "ALTER TABLE users ADD COLUMN email TEXT",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "TRUNCATE",
			query:   "TRUNCATE users",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "GRANT",
			query:   "GRANT SELECT ON users TO readonly",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "multiple statements with non-SELECT",
			query:   "SELECT 1; DROP TABLE users",
			wantErr: "only SELECT statements are allowed",
		},
		{
			name:    "multiple SELECT statements",
			query:   "SELECT 1; SELECT 2",
			wantErr: "", // multiple SELECTs are fine
		},

		// --- Dangerous functions ---
		{
			name:    "pg_sleep",
			query:   "SELECT pg_sleep(10)",
			wantErr: "pg_sleep",
		},
		{
			name:    "pg_sleep_for",
			query:   "SELECT pg_sleep_for('5 minutes')",
			wantErr: "pg_sleep_for",
		},
		{
			name:    "pg_terminate_backend",
			query:   "SELECT pg_terminate_backend(1234)",
			wantErr: "pg_terminate_backend",
		},
		{
			name:    "pg_cancel_backend",
			query:   "SELECT pg_cancel_backend(1234)",
			wantErr: "pg_cancel_backend",
		},
		{
			name:    "lo_import",
			query:   "SELECT lo_import('/etc/passwd')",
			wantErr: "lo_import",
		},
		{
			name:    "lo_export",
			query:   "SELECT lo_export(12345, '/tmp/out')",
			wantErr: "lo_export",
		},
		{
			name:    "dblink",
			query:   "SELECT * FROM dblink('host=evil', 'SELECT 1') AS t(id INT)",
			wantErr: "dblink",
		},
		{
			name:    "dblink_exec",
			query:   "SELECT dblink_exec('host=evil', 'DROP TABLE users')",
			wantErr: "dblink_exec",
		},
		{
			name:    "set_config",
			query:   "SELECT set_config('log_statement', 'all', false)",
			wantErr: "set_config",
		},
		{
			name:    "pg_reload_conf",
			query:   "SELECT pg_reload_conf()",
			wantErr: "pg_reload_conf",
		},
		{
			name:    "pg_advisory_lock",
			query:   "SELECT pg_advisory_lock(1)",
			wantErr: "pg_advisory_lock",
		},
		{
			name:    "nextval",
			query:   "SELECT nextval('users_id_seq')",
			wantErr: "nextval",
		},
		{
			name:    "setval",
			query:   "SELECT setval('users_id_seq', 100)",
			wantErr: "setval",
		},
		{
			name:    "currval",
			query:   "SELECT currval('users_id_seq')",
			wantErr: "currval",
		},
		{
			name:    "txid_current",
			query:   "SELECT txid_current()",
			wantErr: "txid_current",
		},
		{
			name:    "dangerous function in WHERE",
			query:   "SELECT * FROM users WHERE pg_sleep(10) IS NOT NULL",
			wantErr: "pg_sleep",
		},
		{
			name:    "dangerous function in subquery",
			query:   "SELECT * FROM users WHERE id IN (SELECT pg_sleep(1)::int)",
			wantErr: "pg_sleep",
		},
		{
			name:    "dangerous function in CTE",
			query:   "WITH evil AS (SELECT pg_sleep(10)) SELECT * FROM evil",
			wantErr: "pg_sleep",
		},

		// --- DoS patterns ---
		{
			name:    "recursive CTE without LIMIT",
			query:   "WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 1000000) SELECT x FROM cnt",
			wantErr: "WITH RECURSIVE requires a LIMIT",
		},
		{
			name:    "large generate_series",
			query:   "SELECT * FROM generate_series(1, 10000000)",
			wantErr: "generate_series range too large",
		},
		{
			name:    "negative large generate_series",
			query:   "SELECT * FROM generate_series(-5000000, 5000001)",
			wantErr: "generate_series range too large",
		},
		{
			name:    "cross join",
			query:   "SELECT * FROM users CROSS JOIN orders",
			wantErr: "cross join is not allowed",
		},
		{
			name:    "implicit cross join without WHERE",
			query:   "SELECT * FROM users, orders",
			wantErr: "implicit cross join",
		},
		{
			name:  "implicit join with WHERE is OK",
			query: "SELECT * FROM users, orders WHERE users.id = orders.user_id",
		},

		// --- SELECT INTO ---
		{
			name:    "SELECT INTO",
			query:   "SELECT * INTO new_table FROM users",
			wantErr: "SELECT INTO is not allowed",
		},

		// --- Syntax error ---
		{
			name:    "invalid SQL",
			query:   "SELECTT * FROMM users",
			wantErr: "failed to parse query",
		},

		// --- Empty query ---
		{
			name:    "empty query",
			query:   "",
			wantErr: "empty query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPostgreSQL(tt.query)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("CheckPostgreSQL(%q) = %v, want nil", tt.query, err)
				}
			} else {
				if err == nil {
					t.Errorf("CheckPostgreSQL(%q) = nil, want error containing %q", tt.query, tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckPostgreSQL(%q) = %v, want error containing %q", tt.query, err, tt.wantErr)
				}
			}
		})
	}
}
