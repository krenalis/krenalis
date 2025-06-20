//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package sender

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

const asserts = false // enable during development for assertions
const traces = false  // set to true to trace execution flow

var tracesMu sync.Mutex

// uuidDeterministicNS defines the namespace used to generate deterministic UUIDv5 values.
var uuidDeterministicNS = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// skipMarker is assigned to a user when an iterator skips one of its events.
// It prevents further events from that user being consumed by another iterator
// until the skipped event is processed, preserving event order.
var skipMarker = new(iterator)

type Ack struct {
	Action int
	Event  string
}

// AcksFunc is a function invoked by the sender to report the result of event
// delivery.
type AcksFunc func(acks []Ack, err error)

// SendEventsFunc is a function that sends events to apps.
type SendEventsFunc func(ctx context.Context, events meergo.Events) error

// maxQueueDelay is the maximum time an event can stay in the queue before being
// sent. See also the Sender.minBatchSize field.
const maxQueueDelay = 200 * time.Millisecond

// Event represents a message to be delivered to an application.
type Event struct {
	meergo.Event           // original event.
	CreatedAt    time.Time // time at which the event was created.
	action       int       // action ID.
	user         *user     // associated user; nil if the event was discarded.
	sequence     int       // sequence number; access is synchronized via Sender.mu.
	iterator     *iterator // iterator that consumed the event; nil if it hasn't been consumed.
}

// user represents the state of a user related to event processing.
type user struct {
	iterator    *iterator // iterator holding the user's events; nil if no events are currently being consumed.
	numConsumed int       // number of consumed events (0 <= consumed <= events); 0 if iterator is nil.
	events      int       // total number of events for the user, excluding those still in the waiting queue.
	expectedSeq int       // next expected sequence number to be added to the Sender.events queue.
	nextSeq     int       // next sequence number to assign to a new created event for this user.
	waiting     []*Event  // events waiting for the one with expectedSeq to be enqueued first.
}

// Sender sends events, buffering them internally for batch delivery.
// It ensures that events from the same user (i.e., with the same AnonymousId)
// are delivered to the app in the exact order they were received.
//
// To send an event, follow these steps:
//  1. Call the CreateEvent method. The order in which this is called determines
//     delivery order.
//  2. (Optional) Transform the event and set the Event.Properties field.
//  3. Call the QueueEvent method.
type Sender struct {
	connector  string         // app connector.
	sendEvents SendEventsFunc // function that sends the events to the app.
	acks       AcksFunc       // ack function.

	mu              sync.Mutex
	events          []*Event           // events in the queue; protected by mu.
	users           map[string]*user   // users by anonymous id; protected by mu.
	releasableUsers map[*user]struct{} // users that have been iterated and are now ready to be released.
	iterator        *iterator          // current iterator; protected by mu.
	available       int                // number of available (non-read) records; protected by mu.
	index           int                // index of the oldest available event; 0 if no event is available; protected by mu.
	timer           *time.Timer        // timer to trigger an iterator every maxQueueDelay; protected by mu.
	minBatchSize    int                // minimum number of events in the queue required to trigger a new iteration.

	close struct {
		closed    atomic.Bool        // indicates if the writer has been closed.
		ctx       context.Context    // context passes to iterators.
		cancel    context.CancelFunc // function to cancel iterators executions.
		completed sync.Cond          // signal the completion of the current iteration.
		iterators sync.WaitGroup     // waiting group for the iterators.
	}
}

// New returns a new sender for the provided app. connector is app's connector,
// sendEvents is the function that sends the events to the app, and acks is the
// function that acknowledges successful delivery of events.
func New(connector string, sendEvents SendEventsFunc, acks AcksFunc) *Sender {
	s := &Sender{
		connector:       connector,
		sendEvents:      sendEvents,
		acks:            acks,
		events:          make([]*Event, 0, 1000),
		users:           make(map[string]*user, 1000),
		releasableUsers: make(map[*user]struct{}),
		timer:           newStoppedTimer(),
		minBatchSize:    10,
	}
	s.close.completed.L = &s.mu
	s.close.ctx, s.close.cancel = context.WithCancel(context.Background())
	// Start an iteration every maxQueueDelay.
	go func() {
		for {
			select {
			case <-s.timer.C:
				var iter *iterator
				s.mu.Lock()
				if s.iterator == nil && s.available > 0 {
					s.releaseUsers()
					iter = newIterator(s)
					iter.index = s.index
					s.iterator = iter
				}
				s.resetTimerLocked()
				s.mu.Unlock()
				if iter != nil {
					s.close.iterators.Add(1)
					go s.send(iter)
				}
			case <-s.close.ctx.Done():
				return
			}
		}
	}()
	return s
}

// Close terminates the sender, ensuring that all events are processed before
// returning, unless the provided context is canceled.
// If processing all events fails, an error is returned.
func (s *Sender) Close(ctx context.Context) error {
	if s.close.closed.Swap(true) {
		return nil
	}
	stop := context.AfterFunc(ctx, s.close.cancel)
	defer stop()
	trace("Sender.Close: start closing down\n")
	for {
		var iter *iterator
		s.mu.Lock()
		if s.iterator != nil {
			trace("Sender.Close: wait for the iteration of iterator %p to complete\n", s.iterator)
			s.close.completed.Wait()
		}
		if s.available > 0 {
			s.releaseUsers()
			iter = newIterator(s)
			iter.index = s.index
			s.iterator = iter
			trace("Sender.Close: %d events available; create new iterator %p\n", s.available, iter)
		}
		s.mu.Unlock()
		if iter == nil {
			break
		}
		s.close.iterators.Add(1)
		go s.send(iter)
	}
	trace("Writer.Close: wait for iterators to terminate\n")
	s.close.iterators.Wait()
	if asserts && ctx.Done() == nil {
		s._assertAvailable(0)
	}
	trace("Writer.Close: iterators are terminated; writer is now closed\n")
	return nil
}

// CreateEvent creates a new event with the given action, type, schema, and
// original event.
//
// If src is nil, the event ID is a random UUIDv7. Otherwise, the ID is a
// deterministic UUIDv5 generated from src.
//
// The returned event must be passed to the QueueEvent method, optionally after
// setting the Properties field.
func (s *Sender) CreateEvent(action int, typ string, schema types.Type, event events.Event, src rand.Source) *Event {
	var id uuid.UUID
	if src == nil {
		// Create a random UUID v7.
		id, _ = uuid.NewV7() // safe to ignore error in Go 1.24+
	} else {
		// Create a pseudo-random UUID v5.
		n := src.Uint64()
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, n)
		id = uuid.NewSHA1(uuidDeterministicNS, data)
	}
	anonymousId, ok := event["anonymousId"].(string)
	if !ok {
		panic("core/events/connector/sender: missing anonymousId")
	}
	s.mu.Lock()
	u, ok := s.users[anonymousId]
	if !ok {
		u = &user{}
		s.users[anonymousId] = u
	}
	seq := u.nextSeq
	u.nextSeq++
	s.mu.Unlock()
	ev := &Event{
		Event: meergo.Event{
			ID:     id.String(),
			Type:   typ,
			Schema: schema,
			Raw:    events.RawEvent(event),
		},
		CreatedAt: time.Now().UTC(),
		action:    action,
		user:      u,
		sequence:  seq,
	}
	return ev
}

// DiscardEvent discards the given unqueued event.
// Call this, for example, if the event transformation fails.
//
// DiscardEvent must not be called if QueueEvent has already been called with
// the same event.
func (s *Sender) DiscardEvent(event *Event) {
	s.queueOrDiscardEvent(event, true)
}

// QueueEvent queues an event to be sent later, possibly in a batch.
//
// Events from the same user (i.e., with the same anonymousId) are sent in the
// order they were created using the CreateEvent method.
func (s *Sender) QueueEvent(event *Event) {
	s.queueOrDiscardEvent(event, false)
}

// appendToReadyQueue adds the given event to the ready queue.
func (s *Sender) appendToReadyQueue(event *Event) {
	s.events = append(s.events, event)
	u := event.user
	u.events++
	if u.iterator == nil {
		s.available++
		if s.available == 1 {
			s.index = len(s.events) - 1
		}
		if s.iterator == nil {
			s.resetTimerLocked()
		}
	}
}

// appendToWaitingQueue adds the given event to the user's waiting queue.
func (s *Sender) appendToWaitingQueue(event *Event) {
	u := event.user
	index, _ := slices.BinarySearchFunc(u.waiting, event, func(a, b *Event) int {
		switch {
		case a.sequence < b.sequence:
			return +1
		case a.sequence > b.sequence:
			return -1
		default:
			return 0
		}
	})
	u.waiting = slices.Insert(u.waiting, index, event)
}

// compact compacts the events. It does nothing if s has been closed.
func (s *Sender) compact() {
	s.mu.Lock()
	if s.close.closed.Load() {
		s.mu.Unlock()
		return
	}
	var i int
	for i < len(s.events) && s.events[i] == nil {
		i++
	}
	if i > 0 {
		s.events = slices.Delete(s.events, 0, i)
		if s.iterator != nil {
			s.iterator.index = max(0, s.iterator.index-i)
		}
		s.index = max(0, s.index-i)
		trace("Sender.compact: %d events compacted, %d available\n", i, s.available)
		if asserts {
			s._assertAvailable(s.available)
		}
	}
	s.mu.Unlock()
}

// complete marks the iteration of the current iterator as completed, allowing
// other iterators to be executed.
func (s *Sender) complete() {
	s.mu.Lock()
	trace("Sender.complete: iteration of iterator %p is completed\n", s.iterator)
	// Update minBatchSize based on the maximum number of events sent in the last iteration.
	// If a new higher value is observed, it is applied immediately.
	// Otherwise, minBatchSize decays slowly over time to adapt to reduced load.
	if n := s.iterator.numConsumed; n > 0 {
		if n > s.minBatchSize {
			// Immediately update to the new maximum.
			s.minBatchSize = n
			trace("Sender.complete: minBatchSize increased to %d\n", n)
		} else {
			// Slowly decay over time toward smaller values
			decayed := int(0.9*float64(s.minBatchSize) + 0.1*float64(n))
			if decayed < s.minBatchSize {
				s.minBatchSize = decayed
				trace("Sender.complete: minBatchSize decayed to %d\n", decayed)
			}
		}
	}
	s.iterator = nil
	s.releaseUsers()
	if s.available == 0 {
		s.index = 0
	}
	s.mu.Unlock()
	s.close.completed.Signal()
}

// queueOrDiscardEvent queues the event if discard is false; otherwise, it
// discards the event.
func (s *Sender) queueOrDiscardEvent(event *Event, discard bool) {
	u := event.user
	s.mu.Lock()
	if event.sequence == u.expectedSeq {
		u.expectedSeq++
		if !discard {
			s.appendToReadyQueue(event)
		}
		// Move events from the waiting queue to the ready queue.
		for len(u.waiting) > 0 {
			last := len(u.waiting) - 1
			if u.waiting[last].sequence != u.expectedSeq {
				break
			}
			u.expectedSeq++
			// Append the event to the ready queue unless it has been discarded.
			if u.waiting[last] != nil {
				s.appendToReadyQueue(u.waiting[last])
			}
			u.waiting[last] = nil
			u.waiting = u.waiting[:last]
		}
	} else {
		s.appendToWaitingQueue(event)
	}
	if discard {
		event.user = nil
	}
	if asserts {
		s._assertAvailable(s.available)
	}
	s.mu.Unlock()
}

// read reads an event from the queue. If consume is true, the event is removed;
// otherwise, subsequent calls to read will return the same event.
func (s *Sender) read(consume bool) (*meergo.Event, bool) {
	var event *Event
	s.mu.Lock()
	var i int
	for i = s.iterator.index; i < len(s.events); i++ {
		e := s.events[i]
		if e == nil {
			continue
		}
		if e.iterator != nil {
			continue
		}
		if iter := e.user.iterator; iter != nil && iter != s.iterator {
			continue
		}
		if same := s.iterator.sameUser; same.enabled {
			if same.user == nil {
				s.iterator.sameUser.user = e.user
			} else if same.user != e.user {
				continue
			}
		}
		event = e
		break
	}
	s.iterator.index = i
	if event != nil && consume {
		s.events[i].iterator = s.iterator
		s.iterator.index++
		if event.user.numConsumed == 0 {
			event.user.iterator = s.iterator
			s.available -= event.user.events
		}
		event.user.numConsumed++
		s.iterator.numConsumed += 1
		if asserts {
			s._assertAvailable(s.available)
		}
	}
	if event != nil {
		if consume {
			trace("Sender.read: iterator %p read and consumed event %q (anonymousId %q) at index %d (%d available)\n", s.iterator, event.ID, event.Raw.AnonymousId(), i, s.available)
		} else {
			trace("Sender.read: iterator %p read event %q (anonymousId %q), without consuming, at index %d (%d available)\n", s.iterator, event.Raw.AnonymousId(), event.ID, i, s.available)
		}
	} else {
		if consume {
			trace("Sender.read: iterator %p tried to read, with consuming, at index %d, but no record available\n", s.iterator, i)
		} else {
			trace("Sender.read: iterator %p tried to read, without consuming, at index %d, but no record available\n", s.iterator, i)
		}
	}
	s.mu.Unlock()
	if event == nil {
		return nil, false
	}
	return &event.Event, true
}

// releaseUsers releases the iterated users, making their events available
// again. They cannot be released while an iteration is in progress.
//
// Must be called while holding s.mu.
func (s *Sender) releaseUsers() {
	if s.iterator != nil {
		panic("core/events/collector/sender: releaseUsers called while an iteration is still in progress")
	}
	if len(s.releasableUsers) == 0 {
		return
	}
	for user := range s.releasableUsers {
		user.iterator = nil
		user.numConsumed = 0
		s.available += user.events
	}
	clear(s.releasableUsers)
	for i := 0; i < len(s.events); i++ {
		event := s.events[i]
		if event != nil && event.iterator == nil && event.user.iterator == nil {
			s.index = i
			break
		}
	}
}

// resetTimerLocked schedules the timer so that the oldest available event is
// sent within maxQueueDelay.
// The caller must hold s.mu.
func (s *Sender) resetTimerLocked() {
	if s.available == 0 {
		s.timer.Stop()
		return
	}
	if s.iterator != nil {
		return
	}
	if s.available >= s.minBatchSize {
		s.timer.Reset(0)
		return
	}
	elapsed := time.Since(s.events[s.index].CreatedAt)
	if elapsed < maxQueueDelay {
		s.timer.Reset(maxQueueDelay - elapsed)
	} else {
		s.timer.Reset(0)
	}
}

// send sends events to the app by calling the connector's SendEvents method.
func (s *Sender) send(iter *iterator) {

	trace("Sender.send: iterator %p started\n", iter)
	if asserts {
		s.mu.Lock()
		s._assertAvailable(s.available)
		s.mu.Unlock()
	}

	var errEvents meergo.EventsError
	var errRequest error

	err := s.sendEvents(s.close.ctx, iter)
	if err != nil {
		if ee, ok := err.(meergo.EventsError); ok {
			errEvents = ee
		} else {
			errRequest = err
			trace("Sender.send: SendEvents returned an error that is not an EventsError: %s\n", err)
		}
	}

	var acksByError map[error][]Ack

	s.mu.Lock()
	if s.iterator == iter {
		// SendEvents hasn't started the iteration; mark it as completed.
		s.mu.Unlock()
		trace("Sender.send: SendEvents of iterator %p has returned without starting an iteration, with error %#v\n", iter, err)
		s.complete()
	} else {
		// SendEvents has completed the iteration.
		trace("Sender.send: SendEvents of iterator %p has returned, with error %#v\n", iter, err)
		if asserts {
			s._assertAvailable(s.available)
		}
		acksByError = make(map[error][]Ack)
		var index int
		for i := 0; i < len(s.events); i++ {
			if s.events[i] == nil || s.events[i].iterator != iter {
				continue
			}
			user := s.events[i].user
			s.releasableUsers[user] = struct{}{}
			ack := Ack{
				Action: s.events[i].action,
				Event:  s.events[i].ID,
			}
			var err error
			if errEvents != nil {
				err = errEvents[index]
			} else if errRequest != nil {
				err = errRequest
			}
			acksByError[err] = append(acksByError[err], ack)
			user.events--
			s.events[i] = nil
			index++
		}
		if s.iterator == nil {
			s.releaseUsers()
			if s.available > 0 {
				var d time.Duration
				if s.available < s.minBatchSize {
					d = maxQueueDelay
				}
				s.timer.Reset(d)
			}
		}
		if asserts {
			s._assertAvailable(s.available)
		}
	}
	s.mu.Unlock()

	for err, acks := range acksByError {
		trace("Sender.send: send ack for iterator %p with acks %#v and error %#v\n", iter, acks, err)
		s.acks(acks, err)
	}

	s.close.iterators.Done()
	s.compact()
}

// skip marks the most recently read event as unread. It is invoked when an
// iterator calls Events.Skip and must be executed with s.mu held.
func (s *Sender) skip() {
	s.mu.Lock()
	i := s.iterator.index - 1
	var e *Event
	for {
		e = s.events[i]
		if e != nil && e.iterator == s.iterator {
			break
		}
		i--
	}
	e.iterator = nil
	e.user.iterator = skipMarker
	e.user.numConsumed--
	s.iterator.numConsumed--
	// Mark the user as releasable if no events were consumed during this iteration.
	if e.user.numConsumed == 0 {
		s.releasableUsers[e.user] = struct{}{}
	}
	trace("Sender.skip: iterator %p; skip index %d, current %d\n", s.iterator, i, s.iterator.index)
	if asserts {
		s._assertAvailable(s.available)
	}
	s.mu.Unlock()
}

// _assertAvailable asserts that the available events are n.
// It must be called holding the s.mu mutex.
func (s *Sender) _assertAvailable(n int) {
	available := 0
	for _, user := range s.users {
		if user.iterator == nil {
			available += user.events
		}
	}
	if n != available {
		panic(fmt.Sprintf("core/events/collector/sender: expected %d available, got %d", n, available))
	}
}

// newStoppedTimer returns a new stopped timer.
func newStoppedTimer() *time.Timer {
	t := time.NewTimer(math.MaxInt64)
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	return t
}

// trace prints a tracing message if traces is true.
func trace(msg string, a ...any) {
	if !traces {
		return
	}
	tracesMu.Lock()
	defer tracesMu.Unlock()
	fmt.Printf(msg, a...)
}
