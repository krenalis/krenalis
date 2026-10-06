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
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
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
		n := newNotifier(database, make(chan notification))
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

	for _, scenario := range []string{"acquire", "snapshot", "key", "cancel-key", "replay", "panic"} {
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
				case "replay", "panic":

					// Acquiring the exclusive metadata lock waits for the snapshot transaction
					// to commit, while load still waits for KMS.
					err := database.Transaction(ctx, func(tx *db.Tx) error {
						_, err := tx.Exec(ctx, "LOCK TABLE bootstrap_metadata IN ACCESS EXCLUSIVE MODE")
						if err != nil {
							return err
						}
						query := "DROP TABLE notifications"
						if scenario == "panic" {
							// More events than the channel capacity force cleanup to
							// cancel a reader that cannot finish after the first panic.
							query = `INSERT INTO notifications (version, name, payload) SELECT version, 'AddMember', '[]'::jsonb FROM generate_series(1, 32) AS version`
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
					}

					return
				}
				t.Fatal("expected bootstrap error, got nil")

			}()
			if scenario != "acquire" && scenario != "snapshot" && !validationDone.Load() {
				t.Error("expected completed key validation, got pending validation")
			}
			checkBootstrapConnectionsReleased(t, database, opts.MaxConnections)
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
		"/ratelimiter.(*Limiter).runCompactor", "/state.(*State).keep(", "/state.(*State).keepElections(",
		"/state.(*notifier).init(", "/state.(*notifier).replay(", "/analytics-go.(*client).loop(",
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
