// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state/ratelimiter"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type loadCleanupKMS struct {
	started   chan string
	cancelled chan struct{}
	finish    chan struct{}
	finished  chan struct{}
	calls     atomic.Int32
}

// DecryptDataKey accepts exactly one call and holds it until cancellation and
// explicit release.
func (k *loadCleanupKMS) DecryptDataKey(ctx context.Context, encryptedDataKey []byte) ([]byte, error) {
	if calls := k.calls.Add(1); calls != 1 {
		return nil, fmt.Errorf("unexpected DecryptDataKey call %d", calls)
	}
	defer close(k.finished)
	k.started <- string(encryptedDataKey)
	<-ctx.Done()
	close(k.cancelled)
	<-k.finish
	return nil, ctx.Err()
}

// GenerateDataKey rejects unexpected key generation.
func (*loadCleanupKMS) GenerateDataKey(context.Context, int) ([]byte, []byte, error) {
	panic("unexpected GenerateDataKey call")
}

// GenerateDataKeyWithoutPlaintext rejects unexpected key generation.
func (*loadCleanupKMS) GenerateDataKeyWithoutPlaintext(context.Context, int) ([]byte, error) {
	panic("unexpected GenerateDataKeyWithoutPlaintext call")
}

// bootstrapKMS overrides key decryption to coordinate bootstrap test scenarios.
type bootstrapKMS struct {
	kms.Kms
	decrypt func(context.Context, []byte) ([]byte, error)
}

// DecryptDataKey delegates key decryption to the test hook.
func (k *bootstrapKMS) DecryptDataKey(ctx context.Context, key []byte) ([]byte, error) {
	return k.decrypt(ctx, key)
}

// TestLoadWaitsForPendingNotificationKeyValidationOnSnapshotFailure verifies
// that load cancels pending notification key validation, rolls back the
// snapshot, and waits for validation to finish before returning.
func TestLoadWaitsForPendingNotificationKeyValidationOnSnapshotFailure(t *testing.T) {

	const (
		testDatabase = "krenalis"
		testUser     = "krenalis"
		testPassword = "krenalis"
	)

	postgresContainer, err := postgres.Run(t.Context(),
		testimages.PostgreSQL,
		postgres.WithDatabase(testDatabase),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("expected PostgreSQL container to start, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(postgresContainer)
		if err != nil {
			t.Errorf("expected PostgreSQL container to terminate, got %v", err)
		}
	})
	host, err := postgresContainer.Host(t.Context())
	if err != nil {
		t.Fatalf("expected PostgreSQL host, got %v", err)
	}
	port, err := postgresContainer.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected PostgreSQL port, got %v", err)
	}
	database, err := db.Open(&db.Options{
		Host:           host,
		Port:           int(port.Num()),
		Username:       testUser,
		Password:       testPassword,
		Database:       testDatabase,
		MaxConnections: 2, // locker stays acquired; only the snapshot connection can be reused
	})
	if err != nil {
		t.Fatalf("expected database pool to open, got %v", err)
	}
	t.Cleanup(database.Close)
	_, err = database.Exec(t.Context(), `
		CREATE TABLE metadata (
			installation_id text,
			kms_encrypted_http_secret_key bytea,
			kms_encrypted_oauth_key bytea,
			kms_encrypted_notification_key bytea,
			kms_encrypted_api_key_pepper bytea
		);
		INSERT INTO metadata VALUES (
			'test-installation', 'http-key'::bytea, 'oauth-key'::bytea,
			'notification-key'::bytea, 'api-key-pepper'::bytea
		);
		CREATE TABLE election (number bigint, leader text);
	`)
	if err != nil {
		t.Fatalf("expected snapshot failure fixture to be created, got %v", err)
	}

	// Keep the empty election unreadable until notification key validation starts.
	locker, err := database.Conn(t.Context())
	if err != nil {
		t.Fatalf("expected lock connection, got %v", err)
	}
	defer locker.Close()
	lockTx, err := locker.Underlying().Begin(t.Context())
	if err != nil {
		t.Fatalf("expected lock transaction to begin, got %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := lockTx.Rollback(ctx)
		if err != nil {
			if !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("expected lock transaction rolled back or already closed, got %v", err)
			}
		}
	}()
	_, err = lockTx.Exec(t.Context(), "LOCK TABLE election IN ACCESS EXCLUSIVE MODE")
	if err != nil {
		t.Fatalf("expected election table to be locked, got %v", err)
	}
	conn, err := database.Conn(t.Context())
	if err != nil {
		t.Fatalf("expected snapshot connection, got %v", err)
	}
	snapshotPID := conn.Underlying().PgConn().PID()
	conn.Close()

	keyManager := &loadCleanupKMS{
		started:   make(chan string, 1),
		cancelled: make(chan struct{}),
		finish:    make(chan struct{}),
		finished:  make(chan struct{}),
	}
	state := &State{db: database, cipher: cipher.New(keyManager)}
	t.Cleanup(state.cipher.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	finishValidation := sync.OnceFunc(func() { close(keyManager.finish) })
	returned := make(chan error, 1)
	loadDone := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		finishValidation()
		select {
		case <-loadDone:
		case <-time.After(5 * time.Second):
			t.Error("expected load worker to terminate, got blocked worker")
		}
	})
	go func() {
		defer close(loadDone)
		err := state.load(ctx, nil)
		select {
		case <-keyManager.finished:
		default:
			t.Error("expected validation finished before load returned, got pending validation")
		}
		returned <- err
	}()

	select {
	case key := <-keyManager.started:
		if key != "notification-key" {
			t.Fatalf("expected notification key validation, got %q", key)
		}
	case err := <-returned:
		t.Fatalf("expected validation to start before load returned, got %v", err)
	case <-ctx.Done():
		t.Fatalf("expected validation to start, got %v", ctx.Err())
	}
	select {
	case <-keyManager.cancelled:
		t.Fatal("expected validation pending before snapshot failure, got cancellation")
	default:
	}
	err = lockTx.Rollback(ctx)
	if err != nil {
		t.Fatalf("expected election read to be released, got %v", err)
	}

	// The empty election fails only after the lock is released; cleanup cancels KMS.
	select {
	case <-keyManager.cancelled:
	case err := <-returned:
		t.Fatalf("expected load to wait for pending validation, got return with %v", err)
	case <-ctx.Done():
		t.Fatalf("expected pending validation to be cancelled, got %v", ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("expected caller context to remain active, got %v", err)
	}
	conn, err = database.Conn(ctx)
	if err != nil {
		t.Fatalf("expected snapshot connection released before validation finishes, got %v", err)
	}
	defer conn.Close()
	if pid := conn.Underlying().PgConn().PID(); pid != snapshotPID {
		t.Fatalf("expected snapshot connection %d reused after rollback, got %d", snapshotPID, pid)
	}
	if status := conn.Underlying().PgConn().TxStatus(); status != 'I' {
		t.Fatalf("expected rolled back snapshot transaction with status I, got %c", status)
	}
	select {
	case <-keyManager.finished:
		t.Fatal("expected validation still pending after rollback, got completion")
	default:
	}
	select {
	case err := <-returned:
		t.Fatalf("expected load blocked until validation finishes, got return with %v", err)
	default:
	}

	finishValidation()
	select {
	case err = <-returned:
	case <-ctx.Done():
		t.Fatalf("expected load to return after validation finishes, got %v", ctx.Err())
	}
	if calls := keyManager.calls.Load(); calls != 1 {
		t.Fatalf("expected exactly one notification key decrypt, got %d calls", calls)
	}
	if err != nil {
		if expected := "cannot load election: " + sql.ErrNoRows.Error(); err.Error() != expected {
			t.Fatalf("expected original snapshot error %q, got %v", expected, err)
		}
	}
	if err == nil {
		t.Fatal("expected original snapshot error, got nil")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("expected caller context to remain active, got %v", err)
	}

}

func TestTransformationEqual(t *testing.T) {
	fn1 := &TransformationFunction{ID: "f1", Version: "v1"}
	fn1copy := &TransformationFunction{ID: "f1", Version: "v1"}
	fn2 := &TransformationFunction{ID: "f2", Version: "v1"}

	tests := []struct {
		name  string
		t1    Transformation
		t2    Transformation
		equal bool
	}{
		{
			name:  "nil functions equal mapping",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			t2:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			equal: true,
		},
		{
			name:  "mapping differs when functions nil",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}},
			t2:    Transformation{Mapping: map[string]string{"a": "c"}},
			equal: false,
		},
		{
			name:  "functions equals",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn1copy},
			equal: true,
		},
		{
			name:  "functions differ",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn2},
			equal: false,
		},
		{
			name:  "in paths differ",
			t1:    Transformation{Function: fn1, InPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "out paths differ",
			t1:    Transformation{Function: fn1, OutPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, OutPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "everything equal",
			t1:    Transformation{Function: fn1, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.t1.Equal(tt.t2) != tt.equal {
				t.Fatalf("expected %v", tt.equal)
			}
			if tt.t2.Equal(tt.t1) != tt.equal {
				t.Fatalf("expected symmetry %v", tt.equal)
			}
		})
	}
}

type valuerStringer interface {
	driver.Valuer
	String() string
}

// TestValuerStringerConsistency verifies that String and Value agree.
func TestValuerStringerConsistency(t *testing.T) {
	tests := []struct {
		name string
		v    valuerStringer
		want string
	}{
		{"AccessKeyTypeAPI", AccessKeyTypeAPI, "API"},
		{"AccessKeyTypeMCP", AccessKeyTypeMCP, "MCP"},
		{"Normal", Normal, "Normal"},
		{"Inspection", Inspection, "Inspection"},
		{"Maintenance", Maintenance, "Maintenance"},
		{"Application", Application, "Application"},
		{"Database", Database, "Database"},
		{"File", File, "File"},
		{"FileStorage", FileStorage, "FileStorage"},
		{"MessageBroker", MessageBroker, "MessageBroker"},
		{"SDK", SDK, "SDK"},
		{"Webhook", Webhook, "Webhook"},
		{"WebhooksPerNone", WebhooksPerNone, "None"},
		{"WebhooksPerAccount", WebhooksPerAccount, "Account"},
		{"WebhooksPerConnection", WebhooksPerConnection, "Connection"},
		{"WebhooksPerConnector", WebhooksPerConnector, "Connector"},
		{"Healthy", Healthy, "Healthy"},
		{"NoRecentData", NoRecentData, "NoRecentData"},
		{"RecentError", RecentError, "RecentError"},
		{"Source", Source, "Source"},
		{"Destination", Destination, "Destination"},
		{"TargetEvent", TargetEvent, "Event"},
		{"TargetUser", TargetUser, "User"},
		{"TargetGroup", TargetGroup, "Group"},
		{"JavaScript", JavaScript, "JavaScript"},
		{"Python", Python, "Python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.v.Value()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Value() = %v, want %s", got, tt.want)
			}
			if got := tt.v.String(); got != tt.want {
				t.Fatalf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestValuerStringerInvalidValues verifies invalid value handling.
func TestValuerStringerInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		v    valuerStringer
	}{
		{"AccessKeyType", AccessKeyType(-1)},
		{"WarehouseMode", WarehouseMode(-1)},
		{"ConnectorType", ConnectorType(-1)},
		{"WebhooksPer", WebhooksPer(-1)},
		{"Health", Health(-1)},
		{"Role", Role(-1)},
		{"Target", Target(-1)},
		{"Language", Language(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.v.Value()
			if err == nil {
				t.Fatal("Value() did not return an error")
			}

			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("String() did not panic")
				}
				if fmt.Sprint(got) != err.Error() {
					t.Fatalf("panic = %v, want %s", got, err)
				}
			}()
			_ = tt.v.String()
		})
	}
}

// TestStateBootstrap verifies PostgreSQL bootstrap ordering and resource
// cleanup after operational failures and application panics.
func TestStateBootstrap(t *testing.T) {

	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("bootstrap"), postgres.WithUsername("bootstrap"), postgres.WithPassword("bootstrap"),
		postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected PostgreSQL host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected PostgreSQL port, got %v", err)
	}
	opts := db.Options{Host: host, Port: int(port.Num()), Username: "bootstrap", Password: "bootstrap", Database: "bootstrap", MaxConnections: 4}
	admin, err := db.Open(&opts)
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}
	t.Cleanup(admin.Close)

	t.Run("listen-effective", func(t *testing.T) {

		database, _ := bootstrapDatabase(t, admin, opts, "listen_effective")
		n := newNotifier(database)
		conn, err := n.connect(t.Context())
		if err != nil {
			t.Fatalf("expected listening connection, got %v", err)
		}
		defer closeNotificationConnection(conn)

		if status := conn.Underlying().PgConn().TxStatus(); status != 'I' {
			t.Fatalf("expected committed LISTEN, got transaction status %q", status)
		}
		var channel string
		err = conn.QueryRow(t.Context(), "SELECT pg_listening_channels()").Scan(&channel)
		if err != nil {
			t.Fatalf("expected registered channel, got %v", err)
		}
		if channel != "krenalis" {
			t.Fatalf("expected krenalis channel, got %q", channel)
		}

	})

	for _, notify := range []bool{false, true} {
		t.Run(fmt.Sprintf("snapshot-race-notify-%t", notify), func(t *testing.T) {

			database, keyManager := bootstrapDatabase(t, admin, opts, fmt.Sprintf("race_%t", notify))
			before := bootstrapWorkers()
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			tx, err := database.Begin(ctx)
			if err != nil {
				t.Fatalf("expected writer transaction, got %v", err)
			}
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(ctx, "LOCK TABLE members IN ACCESS EXCLUSIVE MODE")
			if err != nil {
				t.Fatalf("expected snapshot barrier, got %v", err)
			}

			entered := make(chan struct{})
			release := make(chan struct{})
			validation := &bootstrapKMS{Kms: keyManager, decrypt: func(ctx context.Context, key []byte) ([]byte, error) {
				select {
				case <-entered:
				default:
					close(entered)
				}
				select {
				case <-release:
					return keyManager.DecryptDataKey(ctx, key)
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}}
			var state *State
			var bootstrapErr error
			completed := make(chan struct{})
			go func() {
				defer close(completed)
				state, bootstrapErr = New(ctx, database, validation, nil, false)
			}()
			defer func() {
				cancel()
				<-completed
				if state != nil {
					state.Close(context.Background())
				}
			}()
			select {
			case <-entered:
			case <-completed:
				t.Fatalf("expected snapshot key validation, got %v", bootstrapErr)
			case <-ctx.Done():
				t.Fatalf("expected snapshot key validation, got %v", ctx.Err())
			}

			var listenerPID int
			err = database.QueryRow(ctx, "SELECT pid FROM pg_stat_activity WHERE datname = current_database() AND query = 'LISTEN krenalis' AND state = 'idle'").Scan(&listenerPID)
			if err != nil {
				t.Fatalf("expected dedicated listener before snapshot completion, got %v", err)
			}

			// The metadata query established the snapshot; the members lock blocks
			// the remaining snapshot queries.
			var organization string
			err = tx.QueryRow(ctx, "SELECT id FROM organizations").Scan(&organization)
			if err != nil {
				t.Fatalf("expected organization, got %v", err)
			}
			_, err = tx.Exec(ctx, "INSERT INTO members (id, organization, email) VALUES ('111111111111', $1, 'bootstrap@example.test')", organization)
			if err != nil {
				t.Fatalf("expected concurrent member insertion, got %v", err)
			}
			event := AddMember{ID: "111111111111", Organization: organization}
			if notify {

				var encryptedKey []byte
				err = tx.QueryRow(ctx, "SELECT kms_encrypted_notification_key FROM bootstrap_metadata").Scan(&encryptedKey)
				if err != nil {
					t.Fatalf("expected notification key, got %v", err)
				}
				c := cipher.New(keyManager)
				n := notifier{key: c.Key(encryptedKey)}
				_, err = n.Notify(ctx, tx, event)
				c.Close()
				if err != nil {
					t.Fatalf("expected queued notification, got %v", err)
				}

			} else {

				payload, err := json.Marshal(event)
				if err != nil {
					t.Fatalf("expected encoded event, got %v", err)
				}
				_, err = tx.Exec(ctx, "INSERT INTO notifications (version, name, payload) VALUES (1, 'AddMember', $1)", payload)
				if err != nil {
					t.Fatalf("expected log event without NOTIFY, got %v", err)
				}

			}
			err = tx.Commit(ctx)
			if err != nil {
				t.Fatalf("expected committed event during snapshot, got %v", err)
			}
			close(release)
			select {
			case <-completed:
			case <-ctx.Done():
				t.Fatalf("expected completed bootstrap, got %v", ctx.Err())
			}
			if bootstrapErr != nil {
				t.Fatalf("expected completed bootstrap, got %v", bootstrapErr)
			}
			if version := state.Version(); version != 1 {
				t.Fatalf("expected replayed version 1 before New returns, got %d", version)
			}

			// A later event is a runtime barrier after any queued duplicate of version 1.
			err = state.Transaction(ctx, func(tx *db.Tx) (any, error) {
				_, err := tx.Exec(ctx, "UPDATE organizations SET enabled = false WHERE id = $1", organization)
				return SetOrganizationStatus{ID: organization, Enabled: false}, err
			})
			if err != nil {
				t.Fatalf("expected post-replay runtime delivery, got %v", err)
			}
			state.Freeze()
			org := state.organizations[organization]
			if org == nil {
				state.Unfreeze()
				t.Fatalf("expected organization %q in bootstrapped state, got nil", organization)
			}
			count, member := org.usage.counts.Members, org.members[event.ID]
			state.Unfreeze()
			if !member || count != 2 || state.Version() != 2 {
				t.Fatalf("expected member applied once and version 2, got member=%t count=%d version=%d", member, count, state.Version())
			}
			var sameSession bool
			err = database.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_stat_activity WHERE pid = $1 AND query LIKE 'SELECT version, name, payload FROM notifications%')", listenerPID).Scan(&sameSession)
			if err != nil {
				t.Fatalf("expected runtime session lookup, got %v", err)
			}
			if !sameSession || database.PoolStats().AcquiredConns() != 1 {
				t.Fatalf("expected same dedicated session after replay and runtime delivery, got same=%t acquired=%d", sameSession, database.PoolStats().AcquiredConns())
			}
			state.Close(ctx)
			state = nil
			checkBootstrapConnectionsReleased(t, database, opts.MaxConnections)
			if got := bootstrapWorkers(); !reflect.DeepEqual(got, before) {
				t.Errorf("expected bootstrap workers %v, got %v", before, got)
			}

		})
	}

	for _, scenario := range []string{"acquire", "snapshot", "key", "cancel-key", "replay", "gap", "panic"} {
		t.Run(scenario, func(t *testing.T) {

			database, keyManager := bootstrapDatabase(t, admin, opts, "failure_"+strings.ReplaceAll(scenario, "-", "_"))
			before := bootstrapWorkers()
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			if scenario == "acquire" {
				cancel()
			}
			if scenario == "snapshot" {
				_, err := database.Exec(ctx, "DROP VIEW metadata")
				if err != nil {
					t.Fatalf("expected missing metadata fixture, got %v", err)
				}
			}
			var pendingRowsBlocker *db.Conn
			if scenario == "panic" {
				var err error
				pendingRowsBlocker, err = database.Conn(ctx)
				if err != nil {
					t.Fatalf("expected pending rows lock owner, got %v", err)
				}
				defer closeNotificationConnection(pendingRowsBlocker)
				_, err = pendingRowsBlocker.Exec(ctx, "SELECT pg_advisory_lock(45)")
				if err != nil {
					t.Fatalf("expected pending rows barrier, got %v", err)
				}
			}

			var validationDone atomic.Bool
			validation := &bootstrapKMS{Kms: keyManager, decrypt: func(ctx context.Context, key []byte) ([]byte, error) {

				defer validationDone.Store(true)
				switch scenario {
				case "key":
					return nil, errors.New("test key unavailable")
				case "cancel-key":
					cancel()
					<-ctx.Done()
					return nil, ctx.Err()
				case "replay", "gap", "panic":

					// Acquiring the exclusive metadata lock waits for the snapshot transaction
					// to commit, while load still waits for KMS.
					err := database.Transaction(ctx, func(tx *db.Tx) error {
						_, err := tx.Exec(ctx, "LOCK TABLE bootstrap_metadata IN ACCESS EXCLUSIVE MODE")
						if err != nil {
							return err
						}
						query := "DROP TABLE notifications"
						if scenario == "gap" {
							query = "INSERT INTO notifications (version, name, payload) VALUES (2, 'AddMember', '{}'::jsonb)"
						}
						if scenario == "panic" {
							// The first row flushes to the client; the following row
							// cannot drain until the test releases its advisory lock.
							query = `
								INSERT INTO notifications (version, name, payload)
								SELECT version, 'AddMember', jsonb_build_array(repeat('x', 32768)) FROM generate_series(1, 32) AS version;
								ALTER TABLE notifications RENAME TO panic_log;
								CREATE FUNCTION panic_payload(v bigint, p jsonb) RETURNS jsonb LANGUAGE plpgsql STABLE COST 1000000 AS $$
								BEGIN IF v > 1 THEN PERFORM pg_advisory_xact_lock(45); END IF; RETURN p; END $$;
								CREATE VIEW notifications AS SELECT version, name, panic_payload(version, payload) AS payload FROM panic_log;
							`
						}
						_, err = tx.Exec(ctx, query)
						return err
					})
					if err != nil {
						return nil, err
					}

				}

				return keyManager.DecryptDataKey(ctx, key)
			}}
			func() {

				if scenario == "panic" {
					defer func() {
						const expected = "invalid notification payload AddMember (version 1)"
						if got := recover(); got != expected {
							t.Errorf("expected panic %q, got %v", expected, got)
						}
						err := ctx.Err()
						if err != nil {
							t.Errorf("expected panic cleanup before bootstrap context cancellation, got %v", err)
						}
					}()
				}
				state, err := New(ctx, database, validation, nil, true)
				if state != nil {
					state.Close(context.Background())
					t.Fatal("expected failed bootstrap, got a State")
				}
				if err != nil {

					switch scenario {
					case "acquire":
						if !errors.Is(err, context.Canceled) {
							t.Errorf("expected acquisition cancellation, got %v", err)
						}
					case "snapshot":
						if _, ok := errors.AsType[*DBNotInitializedError](err); !ok {
							t.Errorf("expected DBNotInitializedError, got %v", err)
						}
					case "key":
						if !strings.Contains(err.Error(), "validating the notification key: test key unavailable") {
							t.Errorf("expected unavailable notification key, got %v", err)
						}
					case "cancel-key":
						// Snapshot query errors may report cancellation without wrapping context.Canceled.
						if !strings.Contains(err.Error(), context.Canceled.Error()) {
							t.Errorf("expected cancellation during key validation, got %v", err)
						}
					case "replay":
						if !strings.Contains(err.Error(), "cannot replay state notifications") {
							t.Errorf("expected replay DB error, got %v", err)
						}
					case "gap":
						if _, ok := errors.AsType[*replicationError](err); !ok {
							t.Errorf("expected bootstrap terminal replication error returned by New, got %v", err)
						}
					}

					return
				}
				t.Fatal("expected bootstrap error, got nil")

			}()
			if scenario != "acquire" && scenario != "snapshot" && !validationDone.Load() {
				t.Error("expected completed key validation, got pending validation")
			}
			capacity := opts.MaxConnections
			if pendingRowsBlocker != nil {
				capacity--
			}
			checkBootstrapConnectionsReleased(t, database, capacity)
			if got := bootstrapWorkers(); !reflect.DeepEqual(got, before) {
				t.Errorf("expected bootstrap workers %v, got %v", before, got)
			}

		})
	}

}

// bootstrapDatabase creates an isolated database whose first snapshot query
// fails unless LISTEN has committed and the snapshot transaction uses
// REPEATABLE READ and READ ONLY.
func bootstrapDatabase(t *testing.T, admin *db.DB, opts db.Options, name string) (*db.DB, kms.Kms) {

	t.Helper()
	_, err := admin.Exec(t.Context(), "CREATE DATABASE "+name)
	if err != nil {
		t.Fatalf("expected fixture database, got %v", err)
	}
	opts.Database = name
	database, err := db.Open(&opts)
	if err != nil {
		t.Fatalf("expected fixture pool, got %v", err)
	}
	t.Cleanup(database.Close)
	keyManager, err := kms.New(t.Context(), "key:DJ0UMRTROH4pjY/Esh3fAErsPbdYmvsnfDCZtc9K4iU")
	if err != nil {
		t.Fatalf("expected local KMS, got %v", err)
	}
	err = initdb.InitIfEmpty(t.Context(), database, keyManager, true)
	if err != nil {
		t.Fatalf("expected initialized database, got %v", err)
	}
	_, err = database.Exec(t.Context(), `
		ALTER TABLE metadata RENAME TO bootstrap_metadata;
		CREATE FUNCTION checked_metadata() RETURNS SETOF bootstrap_metadata LANGUAGE plpgsql AS $$
		BEGIN
			IF current_setting('transaction_isolation') <> 'repeatable read'
				OR current_setting('transaction_read_only') <> 'on' THEN
				RAISE EXCEPTION 'snapshot must be repeatable read and read only';
			END IF;
			IF NOT EXISTS (SELECT FROM pg_stat_activity WHERE datname = current_database()
				AND pid <> pg_backend_pid() AND query = 'LISTEN krenalis' AND state = 'idle') THEN
				RAISE EXCEPTION 'snapshot started before LISTEN completed';
			END IF;
			RETURN QUERY SELECT * FROM bootstrap_metadata;
		END $$;
		CREATE VIEW metadata AS SELECT * FROM checked_metadata();
	`)
	if err != nil {
		t.Fatalf("expected snapshot ordering guard, got %v", err)
	}

	return database, keyManager
}

// bootstrapWorkers counts only goroutines owned by the State lifecycle, so
// database pool and testcontainers workers do not affect cleanup assertions.
func bootstrapWorkers() map[string]int {

	stack := make([]byte, 1<<20)
	stack = stack[:runtime.Stack(stack, true)]
	counts := map[string]int{}
	for _, worker := range []string{
		"/cipher.newCache.func1", "/ratelimiter.(*Limiter).runRefiller", "/ratelimiter.(*Limiter).runRestorer",
		"/ratelimiter.(*Limiter).runCompactor", "/state.(*State).keepNotifications(", "/state.(*State).keepElections(",
		"/state.(*State).runNotifications(", "/state.(*State).replay(", "/analytics-go.(*client).loop(",
	} {
		counts[worker] = strings.Count(string(stack), worker)
	}

	return counts
}

// checkBootstrapConnectionsReleased acquires the full pool capacity, waiting
// for asynchronous connection cleanup without polling pool statistics.
func checkBootstrapConnectionsReleased(t *testing.T, database *db.DB, capacity int32) {

	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for range capacity {
		conn, err := database.Conn(ctx)
		if err != nil {
			t.Fatalf("expected all pool connections released, got %v", err)
		}
		defer conn.Close()
	}

}

// queryCompletionContext observes pgx unwatching a completed query. The hook
// uses a separate connection to recognize a persisted commit and can hold its
// return until application completes, before Transaction can enter waitVersion.
type queryCompletionContext struct {
	context.Context
	afterQuery func()
}

// AfterFunc preserves cancellation and adds a barrier when pgx stops watching.
func (ctx queryCompletionContext) AfterFunc(f func()) func() bool {
	stop := context.AfterFunc(ctx.Context, f)
	return func() bool {
		stopped := stop()
		if stopped {
			ctx.afterQuery()
		}
		return stopped
	}
}

// Value makes context.AfterFunc use the fixture's scheduling adapter.
func (ctx queryCompletionContext) Value(any) any {
	return nil
}

// TestStateTransactionApplied checks real commits, local dispatch, fast path,
// replay and application completed before the commit call returns.
func TestStateTransactionApplied(t *testing.T) {
	for _, mode := range []string{"fast-path", "replay", "already-applied", "cancel-after-commit"} {
		t.Run(mode, func(t *testing.T) {

			database, key := replicationDatabase(t)
			state, session := replicationState(t, database, key)
			// Align MAX(version)+1 with the fixture's applied cursor.
			logReplicationEvent(t, database, 123, "snapshot")
			_, err := database.Exec(t.Context(), "CREATE TABLE changes (code text PRIMARY KEY)")
			if err != nil {
				t.Fatalf("expected application table, got %v", err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			entered, release := make(chan struct{}), make(chan struct{})
			releaseDispatch := sync.OnceFunc(func() { close(release) })
			defer releaseDispatch()
			state.listeners = []any{func(AddConsentPurpose) { close(entered); <-release }}
			pid := session.conn.Underlying().PgConn().PID()
			if mode == "replay" {
				_, err = session.conn.Exec(ctx, "UNLISTEN krenalis")
				if err != nil {
					t.Fatalf("expected replay-only session, got %v", err)
				}
			}

			var transactionCtx context.Context = ctx
			appliedBeforeWait := false
			if mode == "already-applied" {
				transactionCtx = queryCompletionContext{Context: ctx, afterQuery: func() {

					var committed bool
					err := database.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM notifications WHERE version = 124)").Scan(&committed)
					if err != nil {
						t.Errorf("expected commit observation, got %v", err)
						return
					}
					if !committed {
						return
					}

					err = state.waitVersion(ctx, 124)
					if err != nil {
						t.Errorf("expected application before returning from COMMIT, got %v", err)
						return
					}
					appliedBeforeWait = true

				}}
			}

			observer, err := state.notifications.connect(ctx)
			if err != nil {
				t.Fatalf("expected commit observer, got %v", err)
			}
			defer closeNotificationConnection(observer)
			waiting := make(chan struct{}, 8)
			state.version.next.L = versionWaitLocker{Locker: &state.version.RWMutex, waiting: waiting}
			result := make(chan error, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				result <- state.Transaction(transactionCtx, func(tx *db.Tx) (any, error) {
					_, err := tx.Exec(ctx, "INSERT INTO changes VALUES ('committed')")
					if err != nil {
						return nil, err
					}
					return AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID, Name: "Committed"}, nil
				})
			}()
			defer func() { releaseDispatch(); cancel(); <-finished }()

			_, err = observer.Underlying().WaitForNotification(ctx)
			if err != nil {
				t.Fatalf("expected committed notification, got %v", err)
			}
			select {
			case <-waiting:
			case <-ctx.Done():
				t.Fatalf("expected post-commit version wait, got %v", ctx.Err())
			}

			if mode == "replay" {
				state.close.Add(1)
				go func() {
					defer state.close.Done()
					err := state.replay(state.close.ctx, session, 124, nil)
					if err != nil {
						t.Errorf("expected committed event replay, got %v", err)
					}
				}()
				t.Cleanup(func() { state.close.cancel(); state.close.Wait() })
			} else {
				startReplication(t, state, session)
			}
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatalf("expected started dispatch, got %v", ctx.Err())
			}
			if state.Version() != 123 {
				t.Fatalf("expected old version during dispatch, got %d", state.Version())
			}
			select {
			case err := <-result:
				t.Fatalf("expected pending transaction during dispatch, got %v", err)
			default:
			}

			if mode == "cancel-after-commit" {
				cancel()
				err = <-result
				func() {
					if err != nil {
						if !errors.Is(err, context.Canceled) {
							t.Fatalf("expected canceled local wait, got %v", err)
						}
						return
					}
					t.Fatal("expected canceled local wait, got success")
				}()
			}

			var changes, notifications int
			err = database.QueryRow(t.Context(), "SELECT (SELECT count(*) FROM changes), (SELECT count(*) FROM notifications WHERE version = 124)").Scan(&changes, &notifications)
			if err != nil {
				t.Fatalf("expected persistence query, got %v", err)
			}
			if changes != 1 || notifications != 1 {
				t.Fatalf("expected persisted change and log, got changes=%d notifications=%d", changes, notifications)
			}

			releaseDispatch()
			if mode != "cancel-after-commit" {
				err = <-result
				if err != nil {
					t.Fatalf("expected successful transaction after application, got %v", err)
				}
			}
			if mode == "already-applied" && !appliedBeforeWait {
				t.Fatal("expected application before COMMIT returned, got missing query barrier")
			}
			waitReplicationVersion(t, state, 124)
			if mode == "replay" {
				state.close.Wait()
			}
			if mode == "fast-path" {
				var query string
				err = database.QueryRow(t.Context(), "SELECT query FROM pg_stat_activity WHERE pid = $1", pid).Scan(&query)
				if err != nil {
					t.Fatalf("expected listener query history, got %v", err)
				}
				if query != "LISTEN krenalis" {
					t.Fatalf("expected fast path without replay, got %q", query)
				}
			}

		})
	}
}

// TestStateTransactionCloseAfterCommit verifies that Close interrupts a
// transaction waiting after commit without rolling back the committed change
// or notification.
func TestStateTransactionCloseAfterCommit(t *testing.T) {

	database, key := replicationDatabase(t)
	logReplicationEvent(t, database, 123, "snapshot")
	_, err := database.Exec(t.Context(), "CREATE TABLE changes (code text PRIMARY KEY)")
	if err != nil {
		t.Fatalf("expected application table, got %v", err)
	}
	observer, err := database.Conn(t.Context())
	if err != nil {
		t.Fatalf("expected separate commit observer, got %v", err)
	}
	defer observer.Close()

	state := versionWaitState(t)
	state.db = database
	state.notifications = &notifier{db: database, key: key}
	state.rateLimiter = ratelimiter.New(nil, ratelimiter.Metrics{})
	state.cipher = cipher.New(nil)
	waiting := make(chan struct{}, 8)
	state.version.next.L = versionWaitLocker{Locker: &state.version.RWMutex, waiting: waiting}
	ctx, cancel := context.WithCancel(t.Context())
	timeout, stopTimeout := context.WithTimeout(t.Context(), 10*time.Second)
	defer stopTimeout()
	result := make(chan error, 1)
	finished, closed := make(chan struct{}), make(chan struct{})
	closeState := sync.OnceFunc(func() {
		go func() { state.Close(t.Context()); close(closed) }()
	})
	defer func() {

		cancel()
		closeState()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("expected transaction worker to terminate, got blocked worker")
		}
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Error("expected Close worker to terminate, got blocked worker")
		}

	}()
	go func() {
		defer close(finished)
		result <- state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err := tx.Exec(ctx, "INSERT INTO changes VALUES ('committed')")
			if err != nil {
				return nil, err
			}
			return AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID, Name: "Committed"}, nil
		})
	}()

	// Only Cond.Wait releases this locker, after a successful Commit.
	select {
	case <-waiting:
	case err := <-result:
		t.Fatalf("expected pending post-commit transaction, got %v", err)
	case <-timeout.Done():
		t.Fatalf("expected post-commit version wait, got %v", timeout.Err())
	}
	checkPersisted := func() {
		t.Helper()
		var changes, notifications int
		err := observer.QueryRow(timeout, "SELECT (SELECT count(*) FROM changes WHERE code = 'committed'), (SELECT count(*) FROM notifications WHERE version = 124 AND name = 'AddConsentPurpose')").Scan(&changes, &notifications)
		if err != nil {
			t.Fatalf("expected persistence query, got %v", err)
		}
		if changes != 1 || notifications != 1 {
			t.Fatalf("expected persisted change and notification, got changes=%d notifications=%d", changes, notifications)
		}
	}
	checkPersisted()
	// Acquiring the underlying mutex synchronizes with Cond.Wait's observed
	// release. No replication worker can publish version 124 in this fixture.
	if version := state.Version(); version != 123 {
		t.Fatalf("expected unapplied version 124 with cursor 123, got %d", version)
	}
	select {
	case err := <-result:
		t.Fatalf("expected pending transaction before Close, got %v", err)
	default:
	}

	closeState()
	var txErr error
	select {
	case txErr = <-result:
	case <-timeout.Done():
		t.Fatalf("expected transaction interrupted by Close, got %v", timeout.Err())
	}
	if txErr != nil {
		if !errors.Is(txErr, context.Canceled) {
			t.Fatalf("expected State cancellation after commit, got %v", txErr)
		}
	}
	if txErr == nil {
		t.Fatal("expected State cancellation after commit, got success")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("expected caller context still active, got %v", err)
	}
	select {
	case <-closed:
	case <-timeout.Done():
		t.Fatalf("expected Close to complete, got %v", timeout.Err())
	}
	checkPersisted()

}

// TestStateTransactionWithoutWait verifies nil events, elections and failed
// commits while the version mutex is unavailable, so even a fast wait would
// block. The failed commit occurs after Notify under a deferred constraint.
func TestStateTransactionWithoutWait(t *testing.T) {

	database, key := replicationDatabase(t)
	_, err := database.Exec(t.Context(), "CREATE TABLE changes (code text CONSTRAINT changes_unique UNIQUE DEFERRABLE INITIALLY DEFERRED)")
	if err != nil {
		t.Fatalf("expected deferred constraint fixture, got %v", err)
	}

	expectedChanges := 0
	for _, tc := range []struct {
		name  string
		event any
	}{
		{"nil", nil},
		{"elect", ElectLeader{Number: 1, Leader: "leader"}},
		{"see", SeeLeader{Election: 1}},
		{"commit-failure", AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID}},
	} {
		t.Run(tc.name, func(t *testing.T) {

			state := versionWaitState(t)
			state.db = database
			state.notifications = &notifier{db: database, key: key}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			state.version.Lock()
			unlock := sync.OnceFunc(state.version.Unlock)
			defer unlock()
			result := make(chan error, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				result <- state.Transaction(ctx, func(tx *db.Tx) (any, error) {
					_, err := tx.Exec(ctx, "INSERT INTO changes VALUES ($1)", tc.name)
					if err != nil {
						return nil, err
					}
					if tc.name == "commit-failure" {
						_, err = tx.Exec(ctx, "INSERT INTO changes VALUES ($1)", tc.name)
						if err != nil {
							return nil, err
						}
					}
					return tc.event, nil
				})
			}()
			defer func() { unlock(); cancel(); <-finished }()

			select {
			case err = <-result:
			case <-ctx.Done():
				t.Fatalf("expected transaction without version wait, got %v", ctx.Err())
			}
			if err != nil {
				if tc.name != "commit-failure" || !db.IsUniqueViolation(err) || db.ErrConstraintName(err) != "changes_unique" {
					t.Fatalf("expected deferred commit constraint, got %v", err)
				}
				return
			}
			if tc.name == "commit-failure" {
				t.Fatal("expected failed commit, got success")
			}
			expectedChanges++

		})
	}

	var changes, notifications int
	err = database.QueryRow(t.Context(), "SELECT (SELECT count(*) FROM changes), (SELECT count(*) FROM notifications)").Scan(&changes, &notifications)
	if err != nil {
		t.Fatalf("expected persisted unversioned writes and rollback query, got %v", err)
	}
	if changes != expectedChanges || notifications != 0 {
		t.Fatalf("expected %d committed writes and no versioned log, got changes=%d notifications=%d", expectedChanges, changes, notifications)
	}

}

// callbackLockObserver reports entry into the condition's Lock method.
// The callback enters it before the waiter can reach Cond.Wait.
type callbackLockObserver struct {
	sync.Locker
	onLock func()
}

// Lock reports the attempt before acquiring the underlying mutex.
func (locker callbackLockObserver) Lock() {
	locker.onLock()
	locker.Locker.Lock()
}

// cancelOnErrContext cancels after reading Err while waitVersion holds its
// mutex, then waits for the production callback to enter the condition's Lock.
type cancelOnErrContext struct {
	context.Context
	t              *testing.T
	cancel         context.CancelFunc
	locking        chan struct{}
	returnCanceled bool
	registrations  int
}

// AfterFunc counts registrations without changing cancellation scheduling.
func (ctx *cancelOnErrContext) AfterFunc(f func()) func() bool {
	ctx.registrations++
	return context.AfterFunc(ctx.Context, f)
}

// Err cancels between the predicate check and Cond.Wait, then waits for the
// callback's lock attempt before returning nil or context.Canceled.
func (ctx *cancelOnErrContext) Err() error {

	err := ctx.Context.Err()
	if err != nil {
		return err
	}

	// The lock attempt drives the interleaving; this is only a virtual watchdog.
	guard := time.NewTimer(time.Second)
	defer guard.Stop()
	ctx.cancel()
	select {
	case <-ctx.locking:
	case <-guard.C:
		ctx.t.Fatal("expected cancellation callback to enter condition lock, got no lock attempt")
	}
	if ctx.returnCanceled {
		return ctx.Context.Err()
	}

	return nil
}

// Value hides the underlying cancel context so context uses AfterFunc above.
func (ctx *cancelOnErrContext) Value(any) any {
	return nil
}

// versionWaitLocker observes condition-lock releases using the real mutex.
// Before cancellation, these releases mark Cond.Wait registering a waiter.
type versionWaitLocker struct {
	sync.Locker
	waiting chan struct{}
}

// Unlock reports the condition-lock release before unlocking the mutex.
func (locker versionWaitLocker) Unlock() {
	locker.waiting <- struct{}{}
	locker.Locker.Unlock()
}

// TestStateCloseWaiters verifies that Close wakes independent caller contexts
// before waiting for a worker that still needs to publish an in-flight event.
func TestStateCloseWaiters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		state := versionWaitState(t)
		state.rateLimiter = ratelimiter.New(nil, ratelimiter.Metrics{})
		state.cipher = cipher.New(nil)
		state.changing = &sync.RWMutex{}
		ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: &Organization{ID: stateTestOrganizationID}, consentPurposes: map[string]*ConsentPurpose{}}
		state.workspaces = map[string]*Workspace{stateTestWorkspaceID: ws}
		release := make(chan struct{})
		releaseDispatch := sync.OnceFunc(func() { close(release) })
		defer releaseDispatch()
		state.listeners = []any{func(AddConsentPurpose) { <-release }}
		state.close.Add(1)
		go func() {
			defer state.close.Done()
			err := state.applyNotification(notification{124, "AddConsentPurpose", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a"}`}, nil)
			if err != nil {
				t.Errorf("expected completed in-flight event, got %v", err)
			}
		}()

		const count = 32
		results := make(chan error, count)
		for range count {
			go func() { results <- state.WaitVersion(t.Context(), 124) }()
		}
		synctest.Wait()
		if len(results) != 0 || state.Version() != 123 || ws.consentPurposes[stateTestConsentPurposeID] == nil {
			t.Fatalf("expected pending waits during dispatch, got results=%d version=%d", len(results), state.Version())
		}

		closed := make(chan struct{})
		go func() { state.Close(t.Context()); close(closed) }()
		synctest.Wait()
		if len(results) != count {
			t.Fatalf("expected %d waiters woken by Close, got %d", count, len(results))
		}
		for range count {
			err := <-results
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected State cancellation, got %v", err)
				}
				continue
			}
			t.Fatal("expected State cancellation before publication, got success")
		}
		select {
		case <-closed:
			t.Fatal("expected Close to wait for in-flight event, got early completion")
		default:
		}

		releaseDispatch()
		<-closed
		if state.Version() != 124 {
			t.Fatalf("expected publication before Close completes, got %d", state.Version())
		}

	})
}

// TestWaitVersionCancellationWindow observes the callback's lock attempt before
// Cond.Wait, and checks that a started callback is not joined under the mutex.
func TestWaitVersionCancellationWindow(t *testing.T) {
	for _, returnCanceled := range []bool{false, true} {
		name := "before-wait"
		if returnCanceled {
			name = "callback-started-before-return"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				state := versionWaitState(t)
				parent, cancel := context.WithCancel(t.Context())
				defer cancel()
				locking := make(chan struct{})
				state.version.next.L = callbackLockObserver{
					Locker: &state.version.RWMutex,
					onLock: sync.OnceFunc(func() { close(locking) }),
				}
				ctx := &cancelOnErrContext{Context: parent, t: t, cancel: cancel, locking: locking, returnCanceled: returnCanceled}
				err := state.waitVersion(ctx, 124)
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("expected caller cancellation, got %v", err)
					}
					synctest.Wait() // Any late Broadcast must finish safely.
					return
				}

				t.Fatal("expected caller cancellation, got success")

			})
		})
	}
}

// TestWaitVersionDeadline checks deadline expiry using virtual time.
func TestWaitVersionDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		state := versionWaitState(t)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		start := time.Now()
		err := state.waitVersion(ctx, 124)
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) != time.Second {
				t.Fatalf("expected deadline after one virtual second, got %v after %v", err, time.Since(start))
			}
			return
		}

		t.Fatal("expected deadline, got success")

	})
}

// TestWaitVersionMultipleTargets checks normal publication, insufficient
// broadcasts, and cancellation of one waiter while the others keep waiting.
func TestWaitVersionMultipleTargets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		state := versionWaitState(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		targets := []int{124, 125, 130, 125}
		results := make([]chan error, len(targets))
		for i, target := range targets {
			results[i] = make(chan error, 1)
			waitCtx := t.Context()
			if i == 3 {
				waitCtx = ctx
			}
			go func() { results[i] <- state.waitVersion(waitCtx, target) }()
		}
		synctest.Wait()

		state.version.Lock()
		state.version.next.Broadcast()
		state.version.Unlock()
		synctest.Wait()
		for i, result := range results {
			if len(result) != 0 {
				t.Fatalf("expected target %d still pending after Broadcast, got a result", targets[i])
			}
		}

		cancel()
		synctest.Wait()
		err := <-results[3]
		func() {
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected caller cancellation, got %v", err)
				}
				return
			}
			t.Fatal("expected canceled waiter, got success")
		}()
		for i := range 3 {
			if len(results[i]) != 0 {
				t.Fatalf("expected target %d pending after another waiter's cancellation, got a result", targets[i])
			}
		}

		state.version.Lock()
		state.version.current = 125
		state.version.next.Broadcast()
		state.version.Unlock()
		synctest.Wait()
		for i := range 2 {
			err := <-results[i]
			if err != nil {
				t.Fatalf("expected reached target %d, got %v", targets[i], err)
			}
		}
		if len(results[2]) != 0 {
			t.Fatal("expected target 130 pending at version 125, got a result")
		}

		state.version.Lock()
		state.version.current = 130
		state.version.next.Broadcast()
		state.version.Unlock()
		err = <-results[2]
		if err != nil {
			t.Fatalf("expected reached target 130, got %v", err)
		}

	})
}

// TestWaitVersionPrecedence checks both entry and wakeup observations: success
// wins over both cancellations, then the caller's error wins over the State's.
func TestWaitVersionPrecedence(t *testing.T) {
	for _, pending := range []bool{false, true} {
		for _, reached := range []bool{false, true} {
			for _, source := range []string{"caller", "state", "both"} {
				name := source
				if pending {
					name += "/pending"
				}
				if reached {
					name += "/reached"
				}
				t.Run(name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {

						state := versionWaitState(t)
						state.rateLimiter = ratelimiter.New(nil, ratelimiter.Metrics{})
						state.cipher = cipher.New(nil)
						ctx, cancel := context.WithTimeout(t.Context(), time.Second)
						defer cancel()
						result := make(chan error, 1)
						if pending {
							go func() { result <- state.waitVersion(ctx, 124) }()
							synctest.Wait()
						}

						state.version.Lock()
						if reached {
							state.version.current = 124
						}
						if source != "state" {
							// Wait on the timer under test, without yielding the version
							// mutex to a waiter before the deadline is observable.
							<-ctx.Done()
						}
						if source != "caller" {
							go state.Close(t.Context())
							<-state.close.ctx.Done()
						}
						state.version.next.Broadcast()
						state.version.Unlock()
						if source == "caller" {
							defer state.Close(t.Context())
						}
						if !pending {
							result <- state.waitVersion(ctx, 124)
						}

						err := <-result
						if err != nil {
							expected := error(context.DeadlineExceeded)
							if source == "state" {
								expected = context.Canceled
							}
							if reached || !errors.Is(err, expected) {
								t.Fatalf("expected reached=%t or error %v, got %v", reached, expected, err)
							}
							return
						}
						if !reached {
							t.Fatal("expected interruption before target, got success")
						}

					})
				})
			}
		}
	}
}

// TestWaitVersionReached also checks that the fast path needs no AfterFunc.
func TestWaitVersionReached(t *testing.T) {
	state := versionWaitState(t)
	// A reached version must not inspect the caller's context.
	ctx := &cancelOnErrContext{Context: t.Context(), t: t, cancel: func() { t.Fatal("expected fast path without context checks, got Err") }}
	for _, target := range []int{0, 122, 123} {
		err := state.waitVersion(ctx, target)
		if err != nil {
			t.Fatalf("expected reached version %d, got %v", target, err)
		}
		if ctx.registrations != 0 {
			t.Fatalf("expected fast path without AfterFunc, got %d registrations", ctx.registrations)
		}
	}
}

func versionWaitState(t *testing.T) *State {
	t.Helper()
	state := &State{}
	state.version.current = 123
	state.version.next = sync.Cond{L: &state.version.RWMutex}
	state.close.ctx, state.close.cancel = context.WithCancel(t.Context())
	t.Cleanup(state.close.cancel)
	return state
}
