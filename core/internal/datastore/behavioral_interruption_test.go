// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type behavioralInterruptionObserver struct {
	pg          *pgx.Conn
	snowflake   *sql.Conn
	tag         string
	application string
}

func (o *behavioralInterruptionObserver) read(t *testing.T) behavioralInterruptionState {

	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var result behavioralInterruptionState
	if o.pg != nil {
		row := o.pg.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE wait_event='PgSleep') FROM pg_stat_activity
WHERE application_name=$1 AND state='active' AND pid<>pg_backend_pid()`, o.application)
		var active, sleeping int
		err := row.Scan(&active, &sleeping)
		if err != nil {
			t.Fatalf("expected PostgreSQL query activity, got %v", err)
		}
		if active < 0 || active > 1 || sleeping < 0 || sleeping > active {
			t.Fatalf("expected at most one active target and its sleeping subset, got %d, %d", active, sleeping)
		}
		// Wait for execution inside the UDF, beyond the extended-protocol
		// startup where a cancellation request can arrive too early.
		return behavioralInterruptionState{running: sleeping == 1, ended: active == 0}
	}

	// Tag the observer too: a target tag must never persist on this session.
	ctx = gosnowflake.WithQueryTag(ctx, "krenalis-interruption-observer")
	rows, err := o.snowflake.QueryContext(ctx, `SELECT UPPER(EXECUTION_STATUS),COALESCE(ERROR_CODE,0),
END_TIME>=START_TIME FROM TABLE(INFORMATION_SCHEMA.QUERY_HISTORY_BY_USER(RESULT_LIMIT=>1000,
INCLUDE_CLIENT_GENERATED_STATEMENT=>TRUE)) WHERE QUERY_TAG=? LIMIT 2`, o.tag)
	if err != nil {
		t.Fatalf("expected Snowflake query history, got %v", err)
	}
	count := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status, &result.code, &result.ended); err != nil {
			rows.Close()
			t.Fatalf("expected query history fields, got %v", err)
		}
		count++
		if count > 1 || len(status) > 64 || result.code < 0 || result.code > 999999 {
			rows.Close()
			t.Fatal("expected one bounded history record, got invalid history")
		}
		result.running = status == "RUNNING"
		if result.ended && status != "FAILED_WITH_ERROR" {
			rows.Close()
			t.Fatal("expected failed terminal query, got another terminal state")
		}
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		t.Fatalf("expected complete query history, got %v", err)
	}
	if closeErr != nil {
		t.Fatalf("expected query history cleanup, got %v", closeErr)
	}

	return result
}

func (o *behavioralInterruptionObserver) wait(t *testing.T, running bool) behavioralInterruptionState {

	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		status := o.read(t)
		if running && status.running || !running && status.ended {
			return status
		}
		select {
		case <-ctx.Done():
			t.Fatalf("expected observable query running=%t, got observation timeout", running)
		case <-ticker.C:
		}
	}

}

type behavioralInterruptionState struct {
	running, ended bool
	code           int64
}

// TestBehavioralCancellationPostgreSQL cancels both read paths after execution
// starts.
func TestBehavioralCancellationPostgreSQL(t *testing.T) {
	testBehavioralInterruption(t, "PostgreSQL", "cancellation")
}

// TestBehavioralCancellationSnowflake requires the isolated cancellation
// overlay.
func TestBehavioralCancellationSnowflake(t *testing.T) {
	testBehavioralInterruption(t, "Snowflake", "cancellation")
}

// TestBehavioralTimeoutPostgreSQL checks a server timeout during both read
// paths.
func TestBehavioralTimeoutPostgreSQL(t *testing.T) {
	testBehavioralInterruption(t, "PostgreSQL", "timeout")
}

// TestBehavioralTimeoutSnowflake requires the isolated timeout session overlay.
func TestBehavioralTimeoutSnowflake(t *testing.T) {
	testBehavioralInterruption(t, "Snowflake", "timeout")
}

func newBehavioralInterruptionObserver(t *testing.T, dw warehouses.Warehouse, name, tag, profile string) *behavioralInterruptionObserver {

	t.Helper()
	observer := &behavioralInterruptionObserver{tag: tag, application: tag}
	switch name {
	case "PostgreSQL":
		value, err := dw.ValidateSettings(t.Context())
		if err != nil {
			t.Fatalf("expected test database settings, got %v", err)
		}
		var settings struct {
			Host     string `json:"host"`
			Username string `json:"username"`
			Password string `json:"password"`
			Database string `json:"database"`
			Port     int    `json:"port"`
		}
		err = json.Unmarshal(value, &settings)
		if err != nil {
			t.Fatalf("expected decoded test database settings, got %v", err)
		}
		address := url.URL{Scheme: "postgres", Host: net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port)),
			User: url.UserPassword(settings.Username, settings.Password), Path: settings.Database}
		observer.pg, err = pgx.Connect(t.Context(), address.String())
		if err != nil {
			t.Fatalf("expected observer connection, got %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := observer.pg.Close(ctx)
			if err != nil {
				t.Errorf("expected observer cleanup, got %v", err)
			}
		})
		// The warehouse pool has not opened yet. Only this disposable database
		// inherits the short timeout and unique application name.
		timeout := "2s"
		if profile == "cancellation" {
			timeout = "60s"
		}
		_, err = observer.pg.Exec(t.Context(), "ALTER DATABASE "+pgx.Identifier{settings.Database}.Sanitize()+
			" SET statement_timeout='"+timeout+"'")
		if err != nil {
			t.Fatalf("expected disposable database timeout, got %v", err)
		}
		_, err = observer.pg.Exec(t.Context(), "ALTER DATABASE "+pgx.Identifier{settings.Database}.Sanitize()+
			" SET application_name='"+tag+"'")
		if err != nil {
			t.Fatalf("expected target session name, got %v", err)
		}
	case "Snowflake":
		db, err := snowflaketester.OpenDatabase(t.Context())
		if err != nil {
			t.Fatalf("expected observer pool, got %v", err)
		}
		t.Cleanup(func() {
			err := db.Close()
			if err != nil {
				t.Errorf("expected observer pool cleanup, got %v", err)
			}
		})
		observer.snowflake, err = db.Conn(t.Context())
		if err != nil {
			t.Fatalf("expected pinned observer connection, got %v", err)
		}
		t.Cleanup(func() {
			err := observer.snowflake.Close()
			if err != nil {
				t.Errorf("expected observer connection cleanup, got %v", err)
			}
		})
		// A pinned observer may take longer than the target's timeout when
		// querying history. This changes only its own session, never the pool.
		_, err = observer.snowflake.ExecContext(t.Context(), "ALTER SESSION SET STATEMENT_TIMEOUT_IN_SECONDS=60")
		if err != nil {
			t.Fatalf("expected observer safety timeout, got %v", err)
		}
	default:
		t.Fatalf("expected supported warehouse, got %s", name)
	}

	return observer
}

func testBehavioralInterruption(t *testing.T, name, profile string) {

	t.Helper()
	if testing.Short() {
		t.Skip("requires a live warehouse")
	}
	if name == "Snowflake" {
		dw := warehouses.Registered(name).New(newTestSettingsLoader(json.Value(`{}`)), nil)
		verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
		err := dw.Close()
		if err != nil {
			t.Fatalf("expected unopened warehouse cleanup, got %v", err)
		}
		if !verified {
			t.Fatal("expected verified Snowflake provider, got closed gate")
		}
	}
	dw := newJSONFilterTestWarehouse(t, name)
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	var random [16]byte
	_, err := rand.Read(random[:])
	if err != nil {
		t.Fatalf("expected random query tag, got %v", err)
	}
	tag := "krenalis-interruption/" + hex.EncodeToString(random[:])
	observer := newBehavioralInterruptionObserver(t, dw, name, tag, profile)
	switch name {
	case "PostgreSQL":
		execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events_base (kpid uuid,message_id text,type text,event text,timestamp timestamp(6))`)
		execBehavioralFixtureStatement(t, dw, `CREATE FUNCTION delay_probe(id text, seconds double precision)
RETURNS boolean LANGUAGE plpgsql VOLATILE AS $$ BEGIN PERFORM pg_sleep(seconds); RETURN true; END $$`)
	case "Snowflake":
		execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid VARCHAR(36));
CREATE TABLE krenalis_events_base (kpid VARCHAR(36),message_id VARCHAR,type VARCHAR,event VARCHAR,
 timestamp TIMESTAMP_NTZ(9))`)
		execBehavioralFixtureStatement(t, dw, `CREATE FUNCTION delay_probe(id VARCHAR, seconds FLOAT)
RETURNS BOOLEAN LANGUAGE JAVASCRIPT VOLATILE AS $$
var until = Date.now() + SECONDS * 1000; while (Date.now() < until) {} return true; $$`)
	default:
		t.Fatalf("expected supported warehouse, got %s", name)
	}
	// Warm the UDF and warehouse before timing an operation.
	execBehavioralFixtureStatement(t, dw, `SELECT delay_probe('warm',0.001)`)
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES ('"+behavioralTestID(1)+"')")
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	table := warehouses.Table{Name: "krenalis_events_base", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String()},
		{Name: "timestamp", Type: types.DateTime()},
	}}
	err = dw.Merge(t.Context(), table, [][]any{{behavioralTestID(1), "one", "track", "Delay Probe", at.Add(-time.Hour)}}, nil)
	if err != nil {
		t.Fatalf("expected real event write, got %v", err)
	}
	root := &behavioral.Rule{Count: &behavioral.Count{Scope: behavioral.Scope{Event: "Delay Probe", Grain: "registered-message",
		Start: at.Add(-24 * time.Hour), End: at}, Test: behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1}}}
	for _, operation := range []struct {
		name string
		call func(context.Context) (bool, error)
	}{
		{"membership", func(ctx context.Context) (bool, error) {
			result, err := store.QueryProfiles(ctx, root, behavioral.Bindings{}, at, behavioral.Page{Limit: 1})
			if err != nil {
				return result != nil, err
			}
			if result == nil || len(result.Profiles) != 1 || result.Profiles[0].ID != behavioralTestID(1) ||
				result.Profiles[0].Truth != behavioral.Match || result.Match != 1 || result.NoMatch != 0 ||
				result.Unknown != 0 || !result.At.Equal(at) {
				return true, fmt.Errorf("expected one matching profile and exact totals, got %+v", result)
			}
			return true, nil
		}},
		{"explanation", func(ctx context.Context) (bool, error) {
			result, err := store.ExplainProfileFilter(ctx, root, behavioral.Bindings{}, at, behavioralTestID(1))
			if err != nil {
				return result != nil, err
			}
			if result == nil || len(result.Nodes) != 1 || result.Nodes[0].Truth != behavioral.Match ||
				result.Nodes[0].Count == nil ||
				result.Nodes[0].Count.Certain != 1 || result.Nodes[0].Count.Possible != 1 || !result.At.Equal(at) {
				return true, fmt.Errorf("expected matching explanation, got %+v", result)
			}
			return true, nil
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {

			execBehavioralFixtureStatement(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS
SELECT * FROM krenalis_events_base WHERE delay_probe(message_id,30)`)
			scenario := *observer
			scenario.tag = tag + "/" + operation.name
			ctx, cancel := context.WithTimeout(gosnowflake.WithQueryTag(t.Context(), scenario.tag), 90*time.Second)
			defer cancel()
			type outcome struct {
				partial bool
				err     error
			}
			done := make(chan outcome, 1)
			finished := make(chan struct{})
			t.Cleanup(func() {
				cancel()
				select {
				case <-finished:
				case <-time.After(60 * time.Second):
					t.Error("expected interrupted worker cleanup, got timeout")
				}
			})
			started := time.Now()
			go func() {
				defer close(finished)
				partial, err := operation.call(ctx)
				done <- outcome{partial, err}
			}()
			scenario.wait(t, true)
			measured := started
			if profile == "cancellation" {
				select {
				case <-done:
					t.Fatal("expected operation still in progress before cancellation, got completed operation")
				default:
				}
				measured = time.Now()
				cancel()
			}
			var result outcome
			select {
			case result = <-done:
			case <-time.After(60 * time.Second):
				t.Fatal("expected completed interrupted operation, got client timeout")
			}
			if result.err != nil {

				unavailable, ok := errors.AsType[*WarehouseError](result.err)
				if !ok || result.partial {
					t.Fatalf("expected WarehouseError without partial data, got %T, partial=%t", result.err, result.partial)
				}
				if profile == "cancellation" {
					if !errors.Is(unavailable.Err, context.Canceled) {
						t.Fatalf("expected context cancellation cause, got %T", unavailable.Err)
					}
				} else {
					switch name {
					case "PostgreSQL":
						failure, ok := errors.AsType[*pgconn.PgError](unavailable.Err)
						if !ok || failure.Code != "57014" {
							t.Fatalf("expected PostgreSQL timeout code 57014, got %T", unavailable.Err)
						}
					case "Snowflake":
						failure, ok := errors.AsType[*gosnowflake.SnowflakeError](unavailable.Err)
						if !ok || failure.Number != 630 {
							t.Fatalf("expected Snowflake timeout 630, got %T", unavailable.Err)
						}
					default:
						t.Fatalf("expected supported warehouse, got %s", name)
					}
				}
				if profile == "timeout" && ctx.Err() != nil {
					t.Fatalf("expected server timeout before client deadline, got %v", ctx.Err())
				}
				clientElapsed := time.Since(measured)
				if profile == "timeout" {
					minimum := 2 * time.Second
					if name == "Snowflake" {
						minimum = 10 * time.Second
					}
					if clientElapsed < minimum || name == "PostgreSQL" && clientElapsed >= 20*time.Second {
						t.Fatalf("expected server timeout after at least %v and before natural completion, got %v",
							minimum, clientElapsed)
					}
				}
				terminal := scenario.wait(t, false)
				// PostgreSQL has no query-history status here. Requiring its
				// disappearance before the 30-second delay can finish excludes
				// natural completion mistaken for server-side cancellation.
				if name == "PostgreSQL" && profile == "cancellation" && time.Since(started) >= 20*time.Second {
					t.Fatal("expected PostgreSQL cancellation before natural completion, got delayed termination")
				}
				code := int64(630)
				if profile == "cancellation" {
					code = 604
				}
				if name == "Snowflake" && terminal.code != code {
					t.Fatalf("expected terminal server code %d, got %d", code, terminal.code)
				}
				t.Logf("interruption profile=%s server_code=%d client_elapsed_ms=%d total_elapsed_ms=%d", profile,
					terminal.code, clientElapsed.Milliseconds(), time.Since(started).Milliseconds())

				execBehavioralFixtureStatement(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS
SELECT * FROM krenalis_events_base`)
				complete, err := operation.call(gosnowflake.WithQueryTag(t.Context(), scenario.tag+"/recovery"))
				if err != nil {
					t.Fatalf("expected healthy follow-up on the same warehouse, got %v", err)
				}
				if !complete {
					t.Fatal("expected complete follow-up result, got nil")
				}

				return
			}

			t.Fatalf("expected interrupted operation error, got success, partial=%t", result.partial)

		})
	}

}
