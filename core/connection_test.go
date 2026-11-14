// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	_ "github.com/krenalis/krenalis/warehouses/postgresql"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type literalErrorDatabase struct {
	err          error
	literalCalls int
	queryCalls   int
	value        any
	typ          types.Type
}

// Close releases the test connector.
func (*literalErrorDatabase) Close() error {
	return nil
}

// Columns rejects unexpected schema requests.
func (*literalErrorDatabase) Columns(context.Context, string) ([]connectors.Column, error) {
	return nil, errors.New("core: unexpected Columns call")
}

// Merge rejects unexpected writes.
func (*literalErrorDatabase) Merge(context.Context, connectors.Table, [][]any) error {
	return errors.New("core: unexpected Merge call")
}

// Query records unexpected calls and returns an error.
func (database *literalErrorDatabase) Query(context.Context, string) (connectors.Rows, []connectors.Column, error) {
	database.queryCalls++
	return nil, nil, errors.New("core: unexpected Query call")
}

// SQLLiteral records its arguments and returns the configured error.
func (database *literalErrorDatabase) SQLLiteral(value any, typ types.Type) (string, error) {
	database.literalCalls++
	database.value = value
	database.typ = typ
	return "partial", database.err
}

// ServeUI rejects unexpected UI requests.
func (*literalErrorDatabase) ServeUI(context.Context, string, json.Value, connectors.Role) (*connectors.UI, error) {
	return nil, errors.New("core: unexpected ServeUI call")
}

var literalErrorDatabaseNumber atomic.Uint64

// TestDatabaseQueriesStopOnLiteralError verifies that previews and imports
// propagate SQL literal errors without executing queries.
func TestDatabaseQueriesStopOnLiteralError(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"), postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"),
		postgres.BasicWaitStrategies())
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected container host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected container port, got %v", err)
	}
	database, err := db.Open(&db.Options{
		Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis",
	})
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}
	t.Cleanup(database.Close)
	manager, err := kms.New(t.Context(), "key:"+base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected KMS, got %v", err)
	}
	err = initdb.InitIfEmpty(t.Context(), database, manager, false)
	if err != nil {
		t.Fatalf("expected initialized schema, got %v", err)
	}

	var connector *literalErrorDatabase
	code := fmt.Sprintf("literal-error-test-%d", literalErrorDatabaseNumber.Add(1))
	connectors.RegisterDatabase(connectors.DatabaseSpec{
		Code: code, Label: "Literal error test", Categories: connectors.CategoryDatabase,
	},
		func(*connectors.DatabaseEnv) (*literalErrorDatabase, error) {
			return connector, nil
		})
	const workspace = "111111111111"
	const source = "222222222222"
	const query = "SELECT email, updated_at FROM users WHERE updated_at > ${updated_at} LIMIT ${limit}"
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.String()},
		{Name: "updated_at", Type: types.DateTime()},
	})
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		t.Fatalf("expected schema JSON, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		SELECT $1, id, 'test', 'PostgreSQL', 'Normal', '', '', '' FROM organizations`, workspace)
	if err != nil {
		t.Fatalf("expected test workspace, got %v", err)
	}
	_, err = database.Exec(t.Context(), `INSERT INTO connections
		(id, workspace, connector, role, kms_encrypted_settings_key)
		VALUES ($1, $2, $3, 'Source', '')`, source, workspace, code)
	if err != nil {
		t.Fatalf("expected source connection, got %v", err)
	}
	cursor := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name        string
		pipeline    string
		incremental bool
		cursor      time.Time
		wantValue   any
		wantType    types.Type
	}{
		{name: "preview"},
		{name: "full import", pipeline: "333333333333", cursor: cursor},
		{name: "incremental import without cursor", pipeline: "444444444444", incremental: true},
		{
			name: "incremental import with cursor", pipeline: "555555555555", incremental: true,
			cursor: cursor, wantValue: cursor, wantType: types.DateTime(),
		},
	}
	for _, test := range tests {
		if test.pipeline == "" {
			continue
		}
		_, err = database.Exec(t.Context(), `INSERT INTO pipelines
			(id, connection, target, event_type, ordering_group, delivery_endpoint, transformation_language,
			in_schema, out_schema, transformation_mapping, query, user_id_column, updated_at_column,
			incremental, matching_in, matching_out, update_on_duplicates, table_key)
			VALUES ($1, $2, 'User', '', '', '', 'JavaScript', $3, $3,
			'{"email":"email","updated_at":"updated_at"}', $4, 'email', 'updated_at', $5, '', '', false, '')`,
			test.pipeline, source, rawSchema, query, test.incremental)
		if err != nil {
			t.Fatalf("expected test pipeline, got %v", err)
		}
		_, err = database.Exec(t.Context(), `INSERT INTO pipelines_runs
			(id, pipeline, incremental, cursor, start_time, ping_time)
			VALUES ($1, $1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			test.pipeline, test.incremental, test.cursor)
		if err != nil {
			t.Fatalf("expected active pipeline run, got %v", err)
		}
	}

	st, err := state.New(t.Context(), database, manager, nil, false)
	if err != nil {
		t.Fatalf("expected initialized state, got %v", err)
	}
	t.Cleanup(func() { st.Close(context.WithoutCancel(t.Context())) })
	core := &Core{state: st, connections: connections.New(st)}
	loaded, ok := st.Connection(source)
	if !ok {
		t.Fatal("expected loaded source connection, got none")
	}
	connection := &Connection{core: core, connection: loaded}
	literalErr := errors.New("literal conversion failed")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			connector = &literalErrorDatabase{err: literalErr}
			var err error
			if test.pipeline == "" {
				_, _, _, err = connection.ExecQuery(t.Context(), query, 1)
			} else {
				loaded, ok := st.Pipeline(test.pipeline)
				if !ok {
					t.Fatal("expected loaded pipeline, got none")
				}
				pipeline := &Pipeline{core: core, pipeline: loaded, connection: connection}
				err = pipeline.importUsers(t.Context())
			}
			if connector.queryCalls != 0 {
				t.Errorf("expected no queries, got %d", connector.queryCalls)
			}
			if connector.literalCalls != 1 {
				t.Errorf("expected one literal conversion, got %d", connector.literalCalls)
			}
			if want, ok := test.wantValue.(time.Time); ok {
				got, ok := connector.value.(time.Time)
				if !ok || !got.Equal(want) {
					t.Errorf("expected literal value %v, got %v", want, connector.value)
				}
			} else if connector.value != nil {
				t.Errorf("expected nil literal value, got %#v", connector.value)
			}
			if !types.Equal(connector.typ, test.wantType) {
				t.Errorf("expected literal type %v, got %v", test.wantType, connector.typ)
			}

			if err != nil {
				if test.pipeline == "" {
					if _, ok := errors.AsType[*errors.UnavailableError](err); !ok {
						t.Fatalf("expected an API UnavailableError, got %T: %v", err, err)
					}
					if err.Error() != literalErr.Error() {
						t.Errorf("expected error %q, got %q", literalErr, err)
					}
					return
				}
				failure, ok := errors.AsType[*pipelineError](err)
				if !ok {
					t.Fatalf("expected a pipelineError, got %T: %v", err, err)
				}
				if failure.step != metrics.InputValidationStep {
					t.Errorf("expected InputValidation step, got %v", failure.step)
				}
				unavailable, ok := errors.AsType[*connections.UnavailableError](failure.err)
				if !ok {
					t.Fatalf("expected a connector UnavailableError, got %T: %v", failure.err, failure.err)
				}
				if unavailable.Err != literalErr {
					t.Errorf("expected connector error %v, got %v", literalErr, unavailable.Err)
				}
				return
			}

			t.Fatal("expected literal conversion error, got no error")

		})
	}

}

func TestDumpPreviewEventRequest(t *testing.T) {

	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	if _, err := gz.Write([]byte(`{"a":1}`)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tests := []struct {
		name            string
		body            []byte
		contentType     string
		contentEncoding string
		want            string
	}{
		{
			name:        "json",
			body:        []byte(`{"a":1}`),
			contentType: "application/json",
			want: "POST https://example.test/preview\n" +
				"Content-Type: application/json\r\n" +
				"\n" +
				"{\n  \"a\": 1\n}",
		},
		{
			name:        "ndjson",
			body:        []byte("{\"a\":1}\n{\"b\":2}"),
			contentType: "application/x-ndjson",
			want: "POST https://example.test/preview\n" +
				"Content-Type: application/x-ndjson\r\n" +
				"\n" +
				"{\n    \"a\": 1\n}\n{\n    \"b\": 2\n}",
		},
		{
			name:            "gzip-json",
			body:            gzBuf.Bytes(),
			contentType:     "application/json",
			contentEncoding: "gzip",
			want: "POST https://example.test/preview\n" +
				"Content-Encoding: gzip\r\n" +
				"Content-Type: application/json\r\n" +
				"\n" +
				"{\n  \"a\": 1\n}",
		},
		{
			name:        "binary",
			body:        []byte{0xff, 0x00, 0xfe},
			contentType: "text/plain",
			want: "POST https://example.test/preview\n" +
				"Content-Type: text/plain\r\n" +
				"\n" +
				"[A binary body of 3 bytes]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://example.test/preview", bytes.NewReader(test.body))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}
			if test.contentEncoding != "" {
				req.Header.Set("Content-Encoding", test.contentEncoding)
			}
			out, err := dumpPreviewEventRequest(req)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if string(out) != test.want {
				t.Fatalf("expected %q, got %q", test.want, string(out))
			}
		})
	}

}
