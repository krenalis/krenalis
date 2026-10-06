// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestNotificationProducer verifies that State.Transaction persists events to
// the log and delivers their encrypted payloads through PostgreSQL NOTIFY,
// including fragmented payloads, while preserving nil versus empty values.
func TestNotificationProducer(t *testing.T) {

	database, key := replicationDatabase(t)
	ch := make(chan notification)
	producer := &State{db: database}
	producer.notifications.notifier = &notifier{db: database, key: key}
	conn, err := producer.notifications.connect(t.Context())
	if err != nil {
		t.Fatalf("expected listening session, got %v", err)
	}

	defer closeNotificationConnection(conn)
	observer, err := producer.notifications.connect(t.Context())
	if err != nil {
		t.Fatalf("expected fragment observer, got %v", err)
	}

	defer closeNotificationConnection(observer)
	listenerCtx, cancelListener := context.WithCancel(t.Context())
	listenerErr := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		session := &notificationSession{conn: conn}
		for {

			raw, err := conn.Underlying().WaitForNotification(listenerCtx)
			if err != nil {
				if listenerCtx.Err() == nil {
					listenerErr <- fmt.Errorf("cannot wait for notification: %w", err)
				}
				return
			}

			n, err := session.reconstruct(listenerCtx, key, raw.Payload)
			if err != nil {
				if listenerCtx.Err() == nil {
					listenerErr <- fmt.Errorf("cannot reconstruct notification: %w", err)
				}
				return
			}

			if n != nil {
				select {
				case ch <- *n:
				case <-listenerCtx.Done():
					return
				}
			}

		}
	}()
	defer func() {
		cancelListener()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		select {
		case <-stopped:
		case <-ctx.Done():
			t.Errorf("expected stopped notification listener, got %v", ctx.Err())
		}
	}()
	noise := make([]byte, 30000)
	random := rand.New(rand.NewPCG(1, 2))
	for i := range noise {
		noise[i] = byte(random.Uint32())
	}

	function := &TransformationFunction{ID: "function", Version: "1", Language: JavaScript, Source: "return input"}
	type producerCase struct {
		name      string
		event     any
		fragments int // zero means more than two fragments
	}
	cases := []producerCase{
		{"linked-nil", CreateConnection{LinkedConnections: nil}, 1},
		{"linked-empty-fragmented", CreateConnection{LinkedConnections: []string{}, Settings: noise}, 0},
		{"settings-empty", CreateConnection{Settings: []byte{}, SettingsKey: []byte{}}, 1},
		{"create-no-transformation", CreatePipeline{}, 1},
		{"create-function-no-mapping", CreatePipeline{Transformation: Transformation{Function: function}}, 1},
		{"create-empty-mapping-and-object", CreatePipeline{Transformation: Transformation{Mapping: map[string]string{}, InPaths: []string{}, OutPaths: []string{}}, FormatSettings: json.Value(`{}`)}, 1},
		{"update-no-transformation-unset-nil", UpdatePipeline{}, 1},
		{"update-function-no-mapping-unset-empty", UpdatePipeline{Transformation: Transformation{Function: function}, PropertiesToUnset: []string{}}, 1},
		{"remove-mcp-settings", UpdateWarehouse{Workspace: stateTestWorkspaceID, MCPSettings: nil}, 1},
	}
	// These are encoded SQL-buffer lengths. Raw Base64 cannot have length
	// 1 modulo 4, so 7980 and 8000 are the nearest attainable sizes above
	// 7978 and 7998. Also exercise both boundaries after a full continuation.
	for _, boundary := range []struct{ length, fragments int }{
		{7977, 1}, {7978, 1}, {7980, 2},
		{7997, 2}, {7998, 2}, {8000, 2},
		{15957, 2}, {15958, 2}, {15960, 3},
		{15977, 3}, {15978, 3}, {15980, 3},
	} {
		cases = append(cases, producerCase{
			fmt.Sprintf("encoded-boundary-%d", boundary.length),
			notificationBoundaryEvent(t, key, noise, boundary.length),
			boundary.fragments,
		})
	}
	for index, tc := range cases {
		expectedVersion := index + 1
		passed := t.Run(tc.name, func(t *testing.T) {

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			completed := make(chan error, 1)
			transactionStopped := make(chan struct{})
			go func() {
				defer close(transactionStopped)
				completed <- producer.Transaction(ctx, func(tx *db.Tx) (any, error) { return tc.event, nil })
			}()
			defer func() {
				cancel()
				stopCtx, stopCancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer stopCancel()
				select {
				case <-transactionStopped:
				case <-stopCtx.Done():
					t.Errorf("expected stopped transaction, got %v", stopCtx.Err())
				}
			}()
			var received notification
			select {
			case received = <-ch:
			case err := <-listenerErr:
				t.Fatalf("expected delivered event, got listener error: %v", err)
			case err := <-completed:
				t.Fatalf("expected delivered event before acknowledgement, got %v", err)
			case <-ctx.Done():
				t.Fatalf("expected delivered event, got %v", ctx.Err())
			}

			ack, ok := producer.notifications.acks.LoadAndDelete(received.Version)
			if !ok {
				t.Fatalf("expected transaction acknowledgement, got none for %d", received.Version)
			}

			ackCh, ok := ack.(chan struct{})
			if !ok {
				t.Fatalf("expected acknowledgement channel, got %T", ack)
			}

			select {
			case ackCh <- struct{}{}:
			case <-ctx.Done():
				t.Fatalf("expected acknowledgement receiver, got %v", ctx.Err())
			}

			select {
			case err := <-completed:
				if err != nil {
					t.Fatalf("expected completed transaction, got %v", err)
				}
			case <-ctx.Done():
				t.Fatalf("expected completed transaction, got %v", ctx.Err())
			}

			var logged notification
			err := database.QueryRow(ctx, "SELECT version, name, payload FROM notifications WHERE version = $1", received.Version).Scan(&logged.Version, &logged.Name, &logged.Payload)
			if err != nil {
				t.Fatalf("expected persisted event, got %v", err)
			}

			name := reflect.TypeOf(tc.event).Name()
			if logged.Version != expectedVersion || received.Version != logged.Version || received.Name != name || logged.Name != name {
				t.Fatalf("expected version %d and name %s in log and NOTIFY, got %d/%s and %d/%s", expectedVersion, name, logged.Version, logged.Name, received.Version, received.Name)
			}

			fromLog := reflect.New(reflect.TypeOf(tc.event)).Interface()
			fromNotify := reflect.New(reflect.TypeOf(tc.event)).Interface()
			err = json.Unmarshal([]byte(logged.Payload), fromLog)
			if err != nil {
				t.Fatalf("expected decoded log, got %v", err)
			}

			err = json.Unmarshal([]byte(received.Payload), fromNotify)
			if err != nil {
				t.Fatalf("expected decoded NOTIFY, got %v", err)
			}

			if !reflect.DeepEqual(fromLog, fromNotify) {
				t.Fatal("expected semantically equal log and NOTIFY including nil/empty, got different events")
			}

			switch e := fromNotify.(type) {
			case *CreateConnection:
				expected := tc.event.(CreateConnection)
				if !reflect.DeepEqual(*e, expected) {
					t.Fatal("expected original connection including nil/empty values, got different values")
				}
			case *CreatePipeline:
				expected := tc.event.(CreatePipeline)
				if !reflect.DeepEqual(e.Transformation, expected.Transformation) {
					t.Fatal("expected original transformation including nil/empty values, got different transformation")
				}
				if !json.Value(e.Filter).IsNull() || (expected.FormatSettings == nil && !e.FormatSettings.IsNull()) {
					t.Fatal("expected nil raw JSON values reconstructed as JSON null, got different values")
				}
				if expected.FormatSettings != nil && string(e.FormatSettings) != string(expected.FormatSettings) {
					t.Fatalf("expected format settings %s, got %s", expected.FormatSettings, e.FormatSettings)
				}
			case *UpdatePipeline:
				expected := tc.event.(UpdatePipeline)
				if e.Transformation.Mapping != nil || !reflect.DeepEqual(e.Transformation.Function, expected.Transformation.Function) || !reflect.DeepEqual(e.PropertiesToUnset, expected.PropertiesToUnset) {
					t.Fatal("expected absent mapping and original function/unset properties including nil/empty, got different values")
				}
			case *UpdateWarehouse:
				if e.MCPSettings != nil {
					t.Fatalf("expected nil MCP settings, got %#v", e.MCPSettings)
				}
				for _, n := range []notification{logged, received} {
					org := &Organization{mu: &sync.Mutex{}, workspaces: map[string]*Workspace{}}
					ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: org}
					ws.Warehouse.mcpSettings = []byte("previous value")
					org.workspaces[ws.ID] = ws
					state := &State{mu: &sync.Mutex{}, workspaces: map[string]*Workspace{ws.ID: ws}}
					state.updateWarehouse(n)
					if state.workspaces[ws.ID].Warehouse.mcpSettings != nil {
						t.Fatal("expected removal of previous MCP settings, got retained value")
					}
				}
			}

			fragments := 0
			for {

				raw, err := observer.Underlying().WaitForNotification(ctx)
				if err != nil {
					t.Fatalf("expected actual PostgreSQL fragment, got %v", err)
				}

				fragments++
				if len(raw.Payload) >= 8000 {
					t.Fatalf("expected PostgreSQL payload below 8000 bytes, got %d", len(raw.Payload))
				}

				if !strings.HasSuffix(raw.Payload, "*") {
					if !strings.HasSuffix(raw.Payload, fmt.Sprintf("@%d", received.Version)) {
						t.Fatalf("expected final marker for version %d, got %q", received.Version, raw.Payload)
					}
					break
				}

			}

			if (tc.fragments == 0 && fragments < 3) || (tc.fragments > 0 && fragments != tc.fragments) {
				t.Fatalf("expected %d fragments (zero means more than two), got %d", tc.fragments, fragments)
			}

		})
		if !passed {
			return
		}
	}

}

// TestNotificationRollback verifies a commit failure after both the log INSERT
// and NOTIFY, with a later committed notification as the delivery barrier.
func TestNotificationRollback(t *testing.T) {

	database, key := replicationDatabase(t)
	state := &State{db: database}
	state.notifications.notifier = &notifier{db: database, key: key}
	conn, err := state.notifications.connect(t.Context())
	if err != nil {
		t.Fatalf("expected effective LISTEN, got %v", err)
	}

	defer closeNotificationConnection(conn)
	_, err = database.Exec(t.Context(), "CREATE TABLE deferred_failure (id int CONSTRAINT deferred_failure_unique UNIQUE DEFERRABLE INITIALLY DEFERRED)")
	if err != nil {
		t.Fatalf("expected deferred constraint, got %v", err)
	}

	callbackSucceeded := false
	err = state.Transaction(t.Context(), func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(t.Context(), "INSERT INTO deferred_failure VALUES (1), (1)")
		if err != nil {
			return nil, err
		}
		callbackSucceeded = true
		return AddMember{ID: stateTestMemberID}, nil
	})
	func() {
		if err != nil {
			if !callbackSucceeded || !db.IsUniqueViolation(err) || db.ErrConstraintName(err) != "deferred_failure_unique" {
				t.Fatalf("expected deferred commit constraint after successful callback, got %v (callback %t)", err, callbackSucceeded)
			}
			return
		}
		t.Fatal("expected deferred commit failure, got success")
	}()
	// db.Tx.Commit currently closes the pgx connection after pgxpool has
	// released it on this error path. Discard that idle, closed connection;
	// repairing that independent pool ownership bug is outside step 4.
	discard, err := database.Conn(t.Context())
	if err != nil {
		t.Fatalf("expected released writer connection, got %v", err)
	}

	closeNotificationConnection(discard)
	var count int
	err = database.QueryRow(t.Context(), "SELECT count(*) FROM notifications").Scan(&count)
	if err != nil {
		t.Fatalf("expected log query, got %v", err)
	}

	if count != 0 {
		t.Fatalf("expected no persisted notifications, got %d", count)
	}

	// A version-zero transaction commits without waiting for local application.
	err = state.Transaction(t.Context(), func(tx *db.Tx) (any, error) { return SeeLeader{Election: 42}, nil })
	if err != nil {
		t.Fatalf("expected committed delivery barrier, got %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	raw, err := conn.Underlying().WaitForNotification(ctx)
	if err != nil {
		t.Fatalf("expected committed barrier notification, got %v", err)
	}

	session := &notificationSession{conn: conn}
	n, err := session.reconstruct(ctx, key, raw.Payload)
	if err != nil {
		t.Fatalf("expected reconstructed barrier, got %v", err)
	}

	if n == nil || n.Version != 0 || n.Name != "SeeLeader" {
		t.Fatalf("expected only SeeLeader barrier, got %#v", n)
	}

	var barrier SeeLeader
	decodeNotification(*n, &barrier)
	if barrier.Election != 42 {
		t.Fatalf("expected barrier election 42, got %d", barrier.Election)
	}

	// The failed versioned transaction would necessarily precede this barrier.
	err = database.QueryRow(t.Context(), "SELECT count(*) FROM notifications").Scan(&count)
	if err != nil {
		t.Fatalf("expected log query after barrier, got %v", err)
	}

	if count != 0 {
		t.Fatalf("expected elections outside persisted log, got %d rows", count)
	}

}

// notificationBoundaryEvent finds a reproducible event at an exact encoded
// transport boundary, using real compression and encryption throughout.
func notificationBoundaryEvent(t *testing.T, key *cipher.Key, noise []byte, length int) CreateConnection {

	t.Helper()
	encodedLength := func(event CreateConnection) int {
		b, err := appendEncodeNotification(t.Context(), []byte("NOTIFY krenalis, '"), key, "CreateConnection", event)
		if err != nil {
			t.Fatalf("expected encoded boundary fixture, got %v", err)
		}
		return len(b)
	}
	low, high := 0, len(noise)
	for low < high {
		middle := (low + high) / 2
		if encodedLength(CreateConnection{Settings: noise[:middle]}) < length {
			low = middle + 1
		} else {
			high = middle
		}
	}

	// Compressed lengths are not strictly monotonic. Search around the crossing
	// and vary a small field to reach sizes skipped by settings alone.
	for size := max(0, low-16); size <= min(len(noise), low+16); size++ {
		for padding := range 8 {
			event := CreateConnection{Name: strings.Repeat("x", padding), Settings: noise[:size]}
			if encodedLength(event) == length {
				return event
			}
		}
	}

	t.Fatalf("expected a real encrypted payload of length %d, got no matching fixture", length)

	return CreateConnection{}
}

// replicationDatabase provides a real PostgreSQL log and real notification key.

func TestParsePayload(t *testing.T) {

	tests := []struct {
		notification string
		version      int
		name         string
		payload      string
		err          bool
	}{
		{`foo{}`, 0, `foo`, `{}`, false},
		{`foo{}@5`, 5, `foo`, `{}`, false},
		{`boo{"a":{"b":5}}@6301`, 6301, `boo`, `{"a":{"b":5}}`, false},
		{``, 0, ``, ``, true},
		{`{}`, 0, ``, ``, true},
		{`boo`, 0, ``, ``, true},
		{`boo123`, 0, ``, ``, true},
		{`boo{}0`, 0, ``, ``, true},
		{`boo{}-1`, 0, ``, ``, true},
		{`boo{} 5`, 0, ``, ``, true},
	}

	for _, test := range tests {
		version, name, payload, err := parsePayload(test.notification)
		if err != nil {
			if !test.err {
				t.Fatalf("%s: cannot parse notification: %s", test.notification, err)
			}
			continue
		}
		if test.err {
			t.Fatalf("%s: expected error, got no errors", test.notification)
		}
		if version != test.version {
			t.Fatalf("%s: expected version %d, got %d", test.notification, test.version, version)
		}
		if name != test.name {
			t.Fatalf("%s: expected name %q, got %q", test.notification, test.name, name)
		}
		if payload != test.payload {
			t.Fatalf("%s: expected payload %q, got %q", test.notification, test.payload, payload)
		}
	}

}

func replicationDatabase(t *testing.T) (*db.DB, *cipher.Key) {

	t.Helper()
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("replication"), postgres.WithUsername("replication"), postgres.WithPassword("replication"),
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

	database, err := db.Open(&db.Options{Host: host, Port: int(port.Num()), Username: "replication", Password: "replication", Database: "replication", MaxConnections: 4})
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}

	t.Cleanup(database.Close)
	_, err = database.Exec(t.Context(), "CREATE TABLE notifications (version bigint PRIMARY KEY, name text NOT NULL, payload jsonb NOT NULL)")
	if err != nil {
		t.Fatalf("expected notification log, got %v", err)
	}

	keyManager, err := kms.New(t.Context(), "key:DJ0UMRTROH4pjY/Esh3fAErsPbdYmvsnfDCZtc9K4iU")
	if err != nil {
		t.Fatalf("expected local KMS, got %v", err)
	}

	encrypted, err := keyManager.GenerateDataKeyWithoutPlaintext(t.Context(), 32)
	if err != nil {
		t.Fatalf("expected notification key, got %v", err)
	}

	c := cipher.New(keyManager)
	t.Cleanup(c.Close)

	return database, c.Key(encrypted)
}
