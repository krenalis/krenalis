package events

import (
	"bytes"
	"context"
	"iter"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"

	"github.com/andybalholm/brotli"
)

type PendingOperation struct {
	Event   Event
	Actions []int
}

// DoneEvent represents a done event.
type DoneEvent struct {
	Action int
	ID     string
}

type OperationStore interface {

	// Done marks the specified event as completed. Once Done has been called
	// for every action associated with an event, the operation and its event
	// are permanently removed from the store.
	Done(events ...DoneEvent)

	// Pending returns an iterator to iterate over the pending operations.
	// After completing the iteration, the caller should call the returned function
	// to check for any errors that may have occurred during the iteration.
	Pending(ctx context.Context) (iter.Seq[PendingOperation], func() error)

	// Store permanently saves operations along with their associated events. Each
	// operation remains in the store until all its actions are marked as complete.
	Store(ctx context.Context, operations []PendingOperation) error
}

// PostgreStore implements the OperationStore interface storing the operations
// on the Meergo PostgreSQL server.
type PostgreStore struct {
	db *db.DB
}

// NewPostgreStore return a new PostgreStore.
func NewPostgreStore(db *db.DB) *PostgreStore {
	return &PostgreStore{db}
}

// Done marks the specified action on the given events as completed.
func (store *PostgreStore) Done(events ...DoneEvent) {
	metrics.Increment("PostgreStore.Done.calls", 1)
	metrics.Increment("PostgreStore.Done.passed_events", len(events))
	idsByAction := map[int][]string{}
	for _, event := range events {
		idsByAction[event.Action] = append(idsByAction[event.Action], event.ID)
	}
	for action, ids := range idsByAction {
		var b strings.Builder
		b.WriteString(`UPDATE event_payloads SET actions = array_remove(actions, $1) WHERE id IN ('`)
		for i, id := range ids {
			if i > 0 {
				b.WriteString(`','`)
			}
			b.WriteString(id)
		}
		b.WriteString(`')`)
		ctx := context.Background()
		_, err := store.db.Exec(ctx, b.String(), action)
		if err != nil {
			slog.Error("core/events: cannot update event operations", "err", err)
			return
		}
		_, err = store.db.Exec(ctx, "DELETE FROM event_payloads WHERE actions = '{}'")
		if err != nil {
			slog.Error("core/events: cannot delete event operations", "err", err)
			return
		}
	}
	store.setEventMetrics()
}

// Pending returns an iterator to iterate over the pending operations.
func (store *PostgreStore) Pending(ctx context.Context) (iter.Seq[PendingOperation], func() error) {
	store.setEventMetrics()
	rows, err := store.db.Query(ctx, "SELECT actions, properties FROM event_payloads WHERE actions != '{}' ORDER BY received_at")
	return func(yield func(PendingOperation) bool) {
			if err != nil {
				return
			}
			for rows.Next() {
				op := PendingOperation{}
				var properties []byte
				if err = rows.Scan(&op.Actions, &properties); err != nil {
					_ = rows.Close()
					return
				}
				r := brotli.NewReader(bytes.NewReader(properties))
				op.Event, err = types.Decode[map[string]any](r, Schema)
				if err != nil {
					continue
				}
				if !yield(op) {
					return
				}
				store.setEventMetrics()
			}
		}, func() error {
			return err
		}
}

// Store permanently saves a batch of operations in the database.
// operations must not be empty, and all operations must share the same
// connection and receivedAt properties.
func (store *PostgreStore) Store(ctx context.Context, operations []PendingOperation) error {
	metrics.Increment("PostgreStore.Store.calls", 1)
	metrics.Increment("PostgreStore.Store.number_of_operations_passed", len(operations))
	event := operations[0].Event
	connection := strconv.Itoa(event["connection"].(int))
	receivedAt := event["receivedAt"].(time.Time).Format(time.RFC3339Nano)
	var properties []any
	var insert strings.Builder
	insert.WriteString("INSERT INTO event_payloads (id, connection, received_at, actions, properties) VALUES ")
	for i, op := range operations {
		if i > 0 {
			insert.WriteByte(',')
		}
		insert.WriteString(`('`)
		insert.WriteString(op.Event["id"].(string))
		insert.WriteString(`',`)
		insert.WriteString(connection)
		insert.WriteString(`,'`)
		insert.WriteString(receivedAt)
		insert.WriteString(`','{`)
		for j, action := range op.Actions {
			if j > 0 {
				insert.WriteString(`,`)
			}
			insert.WriteString(strconv.Itoa(action))
		}
		insert.WriteString(`}',$`)
		insert.WriteString(strconv.Itoa(i + 1))
		insert.WriteByte(')')
		data, err := types.Marshal(op.Event, Schema)
		if err != nil {
			continue
		}
		var b bytes.Buffer
		writer := brotli.NewWriter(&b)
		_, _ = writer.Write(data)
		_ = writer.Close()
		properties = append(properties, b.Bytes())
	}
	insert.WriteString(" ON CONFLICT (id) DO NOTHING")
	_, err := store.db.Exec(ctx, insert.String(), properties...)
	if err != nil {
		return err
	}
	store.setEventMetrics()
	return nil
}

func (store *PostgreStore) setEventMetrics() {
	if !metrics.Enabled {
		return
	}
	var count int
	err := store.db.QueryRow(context.Background(), "SELECT count(*) FROM event_payloads").Scan(&count)
	if err != nil {
		panic(err)
	}
	metrics.Set("PostgreStore.event_payloads_count", count)
}
