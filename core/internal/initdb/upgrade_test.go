// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

// schemaSnapshotQuery describes the objects of the current schema regardless
// of how they were created.
const schemaSnapshotQuery = `
	SELECT jsonb_build_object(
		'columns', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
				a.attnotnull, pg_get_expr(d.adbin, d.adrelid)) ORDER BY c.relname, a.attnum)
			FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
			WHERE c.relnamespace = current_schema()::regnamespace AND a.attnum > 0 AND NOT a.attisdropped),
		'constraints', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, con.conname, pg_get_constraintdef(con.oid))
				ORDER BY c.relname, con.conname)
			FROM pg_constraint con
			JOIN pg_class c ON c.oid = con.conrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'indexes', (
			SELECT jsonb_agg(pg_get_indexdef(i.indexrelid) ORDER BY c.relname)
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'enums', (
			SELECT jsonb_agg(jsonb_build_array(t.typname, e.enumlabel) ORDER BY t.typname, e.enumsortorder)
			FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			WHERE t.typnamespace = current_schema()::regnamespace),
		'views', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, pg_get_viewdef(c.oid)) ORDER BY c.relname)
			FROM pg_class c
			WHERE c.relnamespace = current_schema()::regnamespace AND c.relkind = 'v'),
		'functions', (
			SELECT jsonb_agg(pg_get_functiondef(p.oid) ORDER BY pg_get_functiondef(p.oid))
			FROM pg_proc p
			WHERE p.pronamespace = current_schema()::regnamespace)
	)::text`

// TestUpgrade verifies that upgrading a v0.43.0 database produces the schema
// of a newly initialized database, preserves and converts its data, and is
// idempotent.
func TestUpgrade(t *testing.T) {

	fresh := newInitializedTestDatabase(t)
	var expected string
	err := fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&expected)
	if err != nil {
		t.Fatal(err)
	}

	// Upgrading a newly initialized database must not change it.
	err = Upgrade(t.Context(), fresh)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	err = fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("expected unchanged schema %s, got %s", expected, got)
	}

	database := newV0_43_0TestDatabase(t)
	var pipelines, runs string
	err = database.QueryRow(t.Context(), `
		SELECT
			(SELECT jsonb_agg(to_jsonb(p) - 'filter' ORDER BY id) FROM pipelines p)::text,
			(SELECT jsonb_agg(to_jsonb(r) - '{passed_3,passed_4,passed_5,failed_3,failed_4,failed_5}'::text[] ORDER BY id)
				FROM pipelines_runs r)::text`).Scan(&pipelines, &runs)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {

		err = Upgrade(t.Context(), database)
		if err != nil {
			t.Fatal(err)
		}
		err = database.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("expected schema %s, got %s", expected, got)
		}

		assertUpgradedData(t, database, `
			SELECT jsonb_agg(to_jsonb(p) - '{filter,ordering_group,delivery_endpoint,required_consents_operator,
				required_consents_purposes}'::text[] ORDER BY id)
			FROM pipelines p`, pipelines)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_object('id', id, 'filter', filter, 'ordering_group', ordering_group,
				'delivery_endpoint', delivery_endpoint, 'required_consents_operator', required_consents_operator,
				'required_consents_purposes', required_consents_purposes) ORDER BY id)
			FROM pipelines`, `[
				{
					"id": "444444444444",
					"filter": {
						"operator": "And",
						"rules": [
							{"property": ["a"], "operator": "IsNotBetween", "values": [5, 10]},
							{"property": ["b"], "operator": "IsNull"}
						]
					},
					"ordering_group": "",
					"delivery_endpoint": "",
					"required_consents_operator": "and",
					"required_consents_purposes": []
				},
				{
					"id": "666666666666",
					"filter": null,
					"ordering_group": "events",
					"delivery_endpoint": "",
					"required_consents_operator": "and",
					"required_consents_purposes": []
				},
				{
					"id": "777777777777",
					"filter": {"operator": "Or", "rules": [{"property": ["c"], "operator": "Is", "values": ["x"]}]},
					"ordering_group": "create_event",
					"delivery_endpoint": "",
					"required_consents_operator": "and",
					"required_consents_purposes": []
				}
			]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_object('pipeline', pipeline, 'timeslot', timeslot,
				'passed', ARRAY[passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, passed_7, passed_8],
				'failed', ARRAY[failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6, failed_7, failed_8])
				ORDER BY pipeline, timeslot)
			FROM pipelines_metrics`, `[
				{
					"pipeline": "123456789ABC",
					"timeslot": 1,
					"passed": [1, 2, 3, 0, 0, 4, 5, 0, 6],
					"failed": [7, 8, 9, 0, 0, 10, 11, 0, 12]
				},
				{
					"pipeline": "234567891234",
					"timeslot": 1,
					"passed": [10, 0, 10, 0, 0, 4, 4, 10, 0],
					"failed": [0, 0, 0, 0, 0, 0, 0, 0, 0]
				},
				{
					"pipeline": "234567891234",
					"timeslot": 2,
					"passed": [10, 0, 8, 0, 0, 5, 3, 5, 0],
					"failed": [0, 0, 2, 0, 0, 1, 2, 0, 0]
				},
				{
					"pipeline": "444444444444",
					"timeslot": 1,
					"passed": [1, 2, 3, 0, 0, 4, 5, 5, 6],
					"failed": [7, 8, 9, 0, 0, 10, 11, 0, 12]
				},
				{
					"pipeline": "666666666666",
					"timeslot": 1,
					"passed": [1, 2, 3, 3, 0, 4, 5, 0, 6],
					"failed": [7, 8, 9, 0, 0, 10, 11, 0, 12]
				}
			]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(to_jsonb(r) - '{passed_3,passed_4,passed_5,passed_6,passed_7,passed_8,failed_3,failed_4,
				failed_5,failed_6,failed_7,failed_8}'::text[] ORDER BY id)
			FROM pipelines_runs r`, runs)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_object('id', id,
				'passed', ARRAY[passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, passed_7, passed_8],
				'failed', ARRAY[failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6, failed_7, failed_8]))
			FROM pipelines_runs`, `[
				{
					"id": "555555555555",
					"passed": [13, 14, 15, 0, 0, 16, 17, 17, 18],
					"failed": [19, 20, 21, 0, 0, 22, 23, 0, 24]
				}
			]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_array(message, step) ORDER BY message)
			FROM pipelines_errors`, `[
				["filter", 2],
				["finalize", 8],
				["output validation", 6],
				["transformation", 5]
			]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(to_jsonb(d))
			FROM discontinued_functions d`, `[
				{"id": "arn:aws:lambda:eu-west-1:1:function:transform.js", "organization": null,
					"discontinued_at": "2026-01-02T03:04:05"}
			]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_build_object(
				'metadata', (SELECT jsonb_build_array(requests_rate_per_minute, requests_max_capacity) FROM metadata),
				'organizations', (
					SELECT jsonb_agg(jsonb_build_array(organization_requests_rate_per_minute,
						organization_requests_max_capacity, workspace_requests_rate_per_minute,
						workspace_requests_max_capacity, workspace_events_rate_per_minute, workspace_events_max_capacity))
					FROM organizations))`, `{
				"metadata": [100, 100],
				"organizations": [[1000, 1000, 1000, 1000, 1000, 20000]]
			}`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(to_jsonb(v) ORDER BY resource)
			FROM organization_connector_references v`, `[
				{"organization": "111111111111", "connector": "dummy", "resource_type": "connection", "resource": "333333333333"},
				{"organization": "111111111111", "connector": "javascript", "resource_type": "connection", "resource": "345678912345"},
				{"organization": "111111111111", "connector": "csv", "resource_type": "pipeline", "resource": "444444444444"},
				{"organization": "111111111111", "connector": "klaviyo", "resource_type": "connection", "resource": "888888888888"},
				{"organization": "111111111111", "connector": "dummy", "resource_type": "connection", "resource": "999999999999"}
			]`)

	}

}

// assertUpgradedData verifies that query returns a single JSON value equal to
// expected.
func assertUpgradedData(t *testing.T, database *db.DB, query, expected string) {
	t.Helper()
	var equal bool
	var got string
	err := database.QueryRow(t.Context(), `SELECT q = $1::jsonb, q::text FROM (`+query+`) AS r(q)`, expected).Scan(&equal, &got)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

// newV0_43_0TestDatabase returns a test database with the schema of release
// v0.43.0 and data covering every upgrade step.
func newV0_43_0TestDatabase(t *testing.T) *db.DB {

	t.Helper()

	schema, err := os.ReadFile("testdata/schema_v0.43.0.sql")
	if err != nil {
		t.Fatal(err)
	}
	database := newTestDatabase(t)
	for query := range strings.SplitSeq(string(schema), ";\n") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err = database.Exec(t.Context(), query)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = database.Exec(t.Context(), `
		INSERT INTO organizations (id, name, enabled, members_limit, access_keys_limit, workspaces_limit,
			connectors_limit, connections_limit, pipelines_limit)
		VALUES ('111111111111', 'ACME inc', true, 10000, 1000, 1000, 1000, 10000, 10000);
		INSERT INTO workspaces (id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('222222222222', '111111111111', 'Workspace', 'postgresql', 'Normal', '\x', '\x', '\x');
		INSERT INTO connections (id, workspace, connector, role, kms_encrypted_settings_key) VALUES
			('333333333333', '222222222222', 'dummy', 'Source', '\x'),
			('345678912345', '222222222222', 'javascript', 'Source', '\x'),
			('999999999999', '222222222222', 'dummy', 'Destination', '\x'),
			('888888888888', '222222222222', 'klaviyo', 'Destination', '\x');
		INSERT INTO pipelines (id, connection, target, event_type, name, enabled, schedule_start, schedule_period,
			in_schema, out_schema, filter, transformation_mapping, transformation_id, transformation_version,
			transformation_language, transformation_source, transformation_preserve_json, transformation_in_paths,
			transformation_out_paths, query, format, path, sheet, compression, order_by, format_settings, export_mode,
			matching_in, matching_out, update_on_duplicates, table_name, table_key, user_id_column, updated_at_column,
			updated_at_format, incremental, cursor, health, properties_to_unset)
		VALUES ('444444444444', '333333333333', 'User', '', 'Import', true, 17, 5, '{"in":1}', '{"out":2}',
			'{
				"logical": "And",
				"conditions": [
					{"property": ["a"], "operator": "OpIsNotBetween", "values": [5, 10]},
					{"property": ["b"], "operator": "IsNull"}
				]
			}',
			'{"mapping":3}', 'function', 'v1', 'Python', 'source', true, '{in}', '{out}', 'SELECT 1', 'csv', '/path',
			'Sheet', 'Gzip', 'name', '{"setting":4}', 'CreateOnly', 'in', 'out', true, 'profiles', 'id', 'uid',
			'updated', 'format', true, '2026-01-02 03:04:05', 'RecentError', '{property}');
		INSERT INTO pipelines (id, connection, target, event_type, filter, transformation_language, matching_in,
			matching_out, table_key, update_on_duplicates) VALUES
			('666666666666', '999999999999', 'Event', 'send_event', NULL, 'JavaScript', '', '', '', false),
			('777777777777', '888888888888', 'Event', 'create_event',
				'{"logical":"Or","conditions":[{"property":["c"],"operator":"Is","values":["x"]}]}',
				'JavaScript', '', '', '', false);
		INSERT INTO pipelines_metrics (organization, workspace, connection, pipeline, target, timeslot,
			passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
			failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)
		VALUES
			('111111111111', '222222222222', '333333333333', '444444444444', 'User', 1,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12),
			('111111111111', '222222222222', '345678912345', '234567891234', 'User', 1,
				10, 0, 10, 4, 4, 0, 0, 0, 0, 0, 0, 0),
			('111111111111', '222222222222', '345678912345', '234567891234', 'User', 2,
				10, 0, 8, 5, 3, 0, 0, 0, 2, 1, 2, 0),
			('111111111111', '222222222222', '999999999999', '666666666666', 'Event', 1,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12),
			('111111111111', '222222222222', 'ABC123456789', '123456789ABC', 'User', 1,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12);
		INSERT INTO pipelines_runs (id, pipeline, start_time, ping_time, end_time,
			passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
			failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, error)
		VALUES ('555555555555', '444444444444', '2026-01-02 03:04:05', '2026-01-02 03:04:06', '2026-01-02 03:04:07',
			13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 'saved error');
		INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message) VALUES
			('444444444444', 1, 2, 1, 'filter'),
			('444444444444', 1, 3, 1, 'transformation'),
			('444444444444', 1, 4, 1, 'output validation'),
			('444444444444', 1, 5, 1, 'finalize');
		INSERT INTO discontinued_functions (id, discontinued_at)
		VALUES ('arn:aws:lambda:eu-west-1:1:function:transform.js', '2026-01-02 03:04:05');
		INSERT INTO notifications (version, name, payload) VALUES (1, 'EndPipelineRun', '{}')`)
	if err != nil {
		t.Fatal(err)
	}

	return database
}
