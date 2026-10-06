// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/errors"

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
