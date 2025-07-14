//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package appwriter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
)

const assert = false // enable during development for assertions
const trace = false  // set to true to trace execution flow

type AcksFunc func(ids []string, err error)

// UpsertFunc is a function that updates or creates records in the app for the
// specified target.
type UpsertFunc func(ctx context.Context, target meergo.Targets, records meergo.Records) error

const minBatchSize = 1000
const maxQueueDelay = 200 * time.Millisecond

// Writer represents a writer for app records.
// It implements the connectors.Writer interface.
//
// By calling the Write method for each record to be written, the records are
// sent to the application, potentially in batches, and the acks function is
// called for confirmation. To ensure that all records are successfully sent to
// the app, the Close method must be called once all Write calls have completed.
type Writer struct {
	connector string         // app connector.
	target    meergo.Targets // target, can be TargetUser or TargetGroup
	upsert    UpsertFunc     // function that updates or creates records in the app.
	acks      AcksFunc       // ack function

	mu        sync.Mutex  // mutex for iterator, records, index, and available fields
	iterator  *iterator   // current iterator, if any; protected by mu
	records   []record    // records in the queue; protected by mu
	available int         // number of available (non-read) records; protected by mu
	timer     *time.Timer // timer to trigger an iterator every maxTimeBetweenIterations; protected by mu

	close struct {
		closed    atomic.Bool        // indicates if the writer has been closed
		ctx       context.Context    // context passes to iterators
		cancel    context.CancelFunc // function to cancel iterators executions
		completed sync.Cond          // signal the completion of the current iteration
		iterators sync.WaitGroup     // waiting group for the iterators
	}
}

// record represents a single user or group to be written and sent to the app.
type record struct {
	iterator   *iterator      // iterator that has consumed the record, if any
	id         string         // user or group identifier
	properties map[string]any // user or group properties
}

// New returns a new Writer. connector is the app's connector, target is the
// record target, upsert is the function that creates or updates records in the
// app, and acks acknowledges both successes and failures.
func New(connector string, target state.Target, upsert UpsertFunc, acks AcksFunc) *Writer {
	w := &Writer{
		connector: connector,
		target:    meergo.Targets(target),
		upsert:    upsert,
		acks:      acks,
		records:   make([]record, 0, 100),
		timer:     time.NewTimer(maxQueueDelay),
	}
	w.close.completed.L = &w.mu
	w.close.ctx, w.close.cancel = context.WithCancel(context.Background())
	// Start an iteration every maxTimeBetweenIterations.
	go func() {
		for {
			select {
			case <-w.timer.C:
				var iter *iterator
				w.mu.Lock()
				if w.iterator == nil && w.available > 0 {
					iter = newIterator(w)
					w.iterator = iter
				}
				w.timer.Reset(maxQueueDelay)
				w.mu.Unlock()
				if iter != nil {
					w.close.iterators.Add(1)
					go w.consume(iter)
				}
			case <-w.close.ctx.Done():
				return
			}
		}
	}()
	return w
}

// Close terminates the writer, ensuring that all records are processed
// before returning, unless the provided context is canceled.
// If processing all records fails, an error is returned.
func (w *Writer) Close(ctx context.Context) error {
	if w.close.closed.Swap(true) {
		return nil
	}
	stop := context.AfterFunc(ctx, w.close.cancel)
	defer stop()
	if trace {
		fmt.Print("Writer.Close: start closing down\n")
	}
	for {
		var iter *iterator
		w.mu.Lock()
		if w.iterator != nil {
			if trace {
				fmt.Printf("Writer.Close: wait for the iteration of iterator %p to complete\n", w.iterator)
			}
			w.close.completed.Wait()
		}
		if w.available > 0 {
			iter = newIterator(w)
			w.iterator = iter
			if trace {
				fmt.Printf("Writer.Close: %d records available; create new iterator %p\n", w.available, iter)
			}
		}
		w.mu.Unlock()
		if iter == nil {
			break
		}
		w.close.iterators.Add(1)
		go w.consume(iter)
	}
	if trace {
		fmt.Print("Writer.Close: wait for iterators to terminate\n")
	}
	w.close.iterators.Wait()
	if assert && ctx.Done() == nil {
		w._assertAvailable(0)
	}
	if trace {
		fmt.Print("Writer.Close: iterators are terminated; writer is now closed\n")
	}
	return nil
}

// Write writes a record with the provided identifier and properties.
// It panics if it called after w has been closed.
func (w *Writer) Write(_ context.Context, id string, properties map[string]any) bool {
	if w.close.closed.Load() {
		panic("core/connectors/appwriter: Write called on a closed writer")
	}
	var iter *iterator
	w.mu.Lock()
	w.records = append(w.records, record{id: id, properties: properties})
	w.available++
	if w.available == minBatchSize {
		w.timer.Reset(0)
	}
	if trace {
		fmt.Printf("Writer.Write: write record id=%q, properties=%p, available=%d\n", id, properties, w.available)
		if iter != nil {
			fmt.Printf("Writer.Write: start new iterator %p\n", iter)
		}
	}
	if assert {
		w._assertAvailable(w.available)
	}
	w.mu.Unlock()
	return true
}

// compact compacts the records. It does nothing if w has been closed.
func (w *Writer) compact() {
	w.mu.Lock()
	if w.close.closed.Load() {
		w.mu.Unlock()
		return
	}
	var i int
	for i < len(w.records) && w.records[i].properties == nil {
		i++
	}
	clear(w.records[:i])
	w.records = append(w.records[:0], w.records[i:]...)
	if w.iterator != nil {
		w.iterator.index = max(0, w.iterator.index-i)
	}
	if trace {
		fmt.Printf("Writer.compact: %d records compacted, %d available\n", i, w.available)
	}
	if assert {
		w._assertAvailable(w.available)
	}
	w.mu.Unlock()
}

// complete marks the iteration of the current iterator as completed, allowing
// other iterators to be executed.
func (w *Writer) complete() {
	w.mu.Lock()
	if trace {
		fmt.Printf("Writer.complete: iteration of iterator %p is completed\n", w.iterator)
	}
	w.iterator = nil
	if w.available >= minBatchSize {
		w.timer.Reset(0)
	}
	w.mu.Unlock()
	w.close.completed.Signal()
}

// postpone postpones the most recently read record, marking it as unread. It
// can only be called after a successful read operation.
func (w *Writer) postpone() {
	w.mu.Lock()
	i := w.iterator.index - 1
	for w.records[i].iterator != w.iterator {
		i--
	}
	w.records[i].iterator = nil
	w.available++
	if trace {
		fmt.Printf("Writer.postpone: iterator %p; postpone index %d, current %d\n", w.iterator, i, w.iterator.index)
	}
	if assert {
		w._assertAvailable(w.available)
	}
	w.mu.Unlock()
}

// read reads an available record and returns it. Returns false if no record is
// available. If op is not opAll, it restricts the returned record to those of
// type creation (opCreate) or update (opUpdate). consume indicates if the
// record should be consumed.
func (w *Writer) read(op op, consume bool) (meergo.Record, bool) {
	var ok bool
	var record meergo.Record
	w.mu.Lock()
	var i int
	for i = w.iterator.index; i < len(w.records); i++ {
		r := w.records[i]
		if r.iterator != nil || r.properties == nil {
			continue
		}
		ok = op == opAll || op == opUpdate && r.id != "" || op == opCreate && r.id == ""
		if ok {
			record.ID = r.id
			record.Properties = r.properties
			break
		}
	}
	w.iterator.index = i
	if ok && consume {
		w.available--
		w.records[i].iterator = w.iterator
		w.iterator.index++
		if assert {
			w._assertAvailable(w.available)
		}
	}
	if trace {
		if ok {
			if consume {
				fmt.Printf("Writer.read: iterator %p read and consumed ID %q at index %d (%d remaining)\n", w.iterator, record.ID, i, w.available)
			} else {
				fmt.Printf("Writer.read: iterator %p read ID %q, without consuming, at index %d (%d remaining)\n", w.iterator, record.ID, i, w.available)
			}
		} else {
			if consume {
				fmt.Printf("Writer.read: iterator %p tried to read, with consuming, at index %d, but no record available\n", w.iterator, i)
			} else {
				fmt.Printf("Writer.read: iterator %p tried to read, without consuming, at index %d, but no record available\n", w.iterator, i)
			}
		}
	}
	w.mu.Unlock()
	return record, ok
}

// consume consumes the available records:
//
//  1. Calls the connector's Upsert method.
//  2. Collects consumed IDs and associated errors.
//  3. Sends acknowledgments (acks).
//  4. Compacts records.
//
// It runs in its own goroutine when records are available.
func (w *Writer) consume(iter *iterator) {
	if trace {
		fmt.Printf("Writer.consume: iterator %p started\n", iter)
	}
	err := w.upsert(w.close.ctx, w.target, iter)
	errors, _ := err.(meergo.RecordsError)
	var errorOf map[error][]string
	w.mu.Lock()
	if w.iterator == iter {
		// Upsert hasn’t started the iteration; mark it as completed.
		if trace {
			fmt.Printf("Writer.consume: Upsert of iterator %p has returned without starting an iteration, with error %#v\n", iter, err)
		}
		w.iterator.index = 0
		w.iterator = nil
		w.close.completed.Signal()
	} else {
		// Upsert has completed the iteration.
		if trace {
			fmt.Printf("Writer.consume: Upsert of iterator %p has returned, with error %#v\n", iter, err)
		}
		errorOf = make(map[error][]string)
		var index int
		for i := 0; i < len(w.records); i++ {
			if w.records[i].iterator != iter {
				continue
			}
			id := w.records[i].id
			if errors != nil {
				err = errors[index]
			}
			errorOf[err] = append(errorOf[err], id)
			w.records[i] = record{}
			index++
		}
		if assert {
			w._assertAvailable(w.available)
		}
	}
	w.mu.Unlock()
	for err, ids := range errorOf {
		if trace {
			fmt.Printf("Writer.consume: send ack for iterator %p with ids %#v and error %#v\n", iter, ids, err)
		}
		w.acks(ids, err)
	}
	w.close.iterators.Done()
	w.compact()
}

// _assertAvailable asserts that the available records are n.
// It must be called holding the w.mu mutex.
func (w *Writer) _assertAvailable(n int) {
	var got int
	for _, r := range w.records {
		if r.iterator == nil && r.properties != nil {
			got++
		}
	}
	if n != got {
		panic(fmt.Sprintf("core/connectors/appwriter: expected %d available, got %d", n, got))
	}
}

type op int

const (
	opAll op = iota
	opCreate
	opUpdate
)

func (op op) String() string {
	switch op {
	case opAll:
		return "all"
	case opCreate:
		return "create"
	case opUpdate:
		return "update"
	default:
		return "unknown"
	}
}
