// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/connections/httpclient"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/prometheus"
	"github.com/meergo/meergo/tools/types"
)

// MaxQueuedEvents defines the maximum number of events allowed in the send
// queue. When this limit is reached, sender.SendEvent blocks until an event is
// removed from the queue.
//
// This value can be configured via the MEERGO_MAX_QUEUED_EVENTS_PER_DESTINATION
// environment variable. The minimum allowed value is 1. If not set, the default
// value is 50000.
var MaxQueuedEvents = 50_000

const asserts = false // enable during development for assertions
const traces = false  // set to true to trace execution flow

var tracesMu sync.Mutex

// postponeMarker is assigned to a user when an iterator postpones one of its
// events. It prevents further events from that user being consumed by another
// iterator until the postponed event is processed, preserving event order.
var postponeMarker = new(iterator)

type Application interface {

	// ID returns the ID of the connection.
	ID() int

	// Connector returns the name of the connector.
	Connector() string

	// WaitTime is a function invoked by the sender to determine how long to
	// wait before starting an iteration, in order to reduce the risk of being
	// throttled by the rate limiter when sending an event.
	//
	// The suggested wait time is based on the rate limiter's state at the time of
	// the call, but there is no guarantee that the request won't still be limited.
	WaitTime(pattern string) (time.Duration, error)

	// SendEvents is a function that sends events to applications.
	SendEvents(ctx context.Context, events connectors.Events) error
}

// maxQueueDelay is the maximum time an event can stay in the queue before being
// sent. See also the Sender.minBatchSize field.
const maxQueueDelay = 200 * time.Millisecond

// Event represents a message to be delivered to an application.
type Event struct {
	connectors.Event             // original event.
	createdAt        time.Time   // time at which the event was created.
	pipeline         int         // pipeline ID.
	user             *user       // user to whom the event belongs.
	sequence         int         // sequence number; access is synchronized via Sender.mu.
	discarded        bool        // true if DiscardEvent was called for this event.
	iterator         *iterator   // iterator that consumed the event; nil if it hasn't been consumed.
	ack              streams.Ack // event ack.
}

func (e *Event) CreatedAt() time.Time {
	return e.createdAt
}

// Sender sends events, buffering them internally for batch delivery.
// It ensures that events from the same user (i.e., with the same Anonymous ID)
// are delivered to the application in the exact order they were received.
//
// To send an event, follow these steps:
//  1. Call the CreateEvent method. The order in which this is called determines
//     delivery order.
//  2. (Optional) Transform the event and set the Event.Properties field.
//  3. Call the SendEvent method.
type Sender struct {
	connector string // application connector.
	metrics   *metrics.Collector

	prometheus struct {
		queueAvailable *prometheus.GaugeFunc
		queueWait      *prometheus.HistogramBuf
	}

	mu                 sync.Mutex
	waitTime           func(pattern string) (time.Duration, error)               // function that returns an estimate of how long to wait before calling sendEvents.
	sendEvents         func(ctx context.Context, events connectors.Events) error // function that sends the events to the application.
	queue              queue                                                     // events queue.
	users              map[string]*user                                          // users by anonymous id; protected by mu.
	releasableUsers    map[*user]struct{}                                        // users that have been iterated and are now ready to be released.
	iterator           *iterator                                                 // current iterator; protected by mu.
	available          int                                                       // number of available (non-read) records; protected by mu.
	availableSince     time.Time                                                 // when available first became > 0.
	schedule           *time.Timer                                               // timer to trigger an iterator every maxQueueDelay; protected by mu.
	minBatchSize       int                                                       // minimum number of events in the queue required to trigger a new iteration.
	rateLimiterPattern string                                                    // pattern of the rate limiter that defines how requests are throttled over time.
	sent               func(messageID string, err error)                         // function called in tests when an event is sent or discarded.

	close struct {
		closed bool                 // indicates whether the sender is closed; protected by mu.
		stop   chan context.Context // starts loop shutdown; returns immediately if the context is canceled.
		done   chan struct{}        // closed when the loop has terminated.
	}
}

// queue is a bounded FIFO-like event queue. events may contain nil holes after
// dequeue; total tracks the number of live entries.
type queue struct {
	cond   sync.Cond
	events []*Event // backing slice for queued events; accessed under the sender mutex
	total  int      // number of non-nil events in events; accessed under the sender mutex
}

// dequeue removes the event at index i and updates the queue state.
// It must be called holding the sender's mu mutex.
func (q *queue) dequeue(i int) {
	q.events[i] = nil
	q.total--
	if asserts {
		q.assertTotal(q.total)
	}
	if q.total < MaxQueuedEvents {
		q.cond.Signal()
	}
}

// enqueue appends event to the queue, blocking while it is full.
// It must be called holding the sender's mu mutex.
func (q *queue) enqueue(event *Event) {
	q.events = append(q.events, event)
	q.total++
}

// assertTotal asserts that the events are n.
//
// It must be called holding the s.mu mutex.
func (q *queue) assertTotal(n int) {
	total := 0
	for _, event := range q.events {
		if event != nil {
			total++
		}
	}
	if n != total {
		panic(fmt.Sprintf("core/events/collector/sender: expected %d queued, got %d", n, total))
	}
}

// New returns a new Sender. app is the application instance, and metrics is the
// metrics collector, or nil if no metrics are collected.
func New(app Application, metrics *metrics.Collector) *Sender {
	s := &Sender{
		connector:       app.Connector(),
		waitTime:        app.WaitTime,
		sendEvents:      app.SendEvents,
		metrics:         metrics,
		users:           make(map[string]*user),
		releasableUsers: make(map[*user]struct{}),
		schedule:        newSchedule(),
		minBatchSize:    min(10, MaxQueuedEvents),
	}
	s.queue.events = make([]*Event, 0, min(64, MaxQueuedEvents))
	s.queue.cond.L = &s.mu
	s.close.stop = make(chan context.Context)
	s.close.done = make(chan struct{})
	// Set the metrics.
	connection := strconv.Itoa(app.ID())
	s.prometheus.queueAvailable = queueAvailableMetric.Register(func() float64 {
		s.mu.Lock()
		a := s.available
		s.mu.Unlock()
		return float64(a)
	}, s.connector, connection)
	s.prometheus.queueWait = queueWaitMetric.Register(s.connector, connection)
	// Start the loop.
	go s.loop()
	return s
}

func (s *Sender) Available() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.available
}

// Close closes the Sender and unblocks any goroutine blocked in DiscardEvent or
// SendEvents. As long as the provided context is not canceled, it waits for all
// in-flight sends to complete before returning.
//
// When Close is called, no other calls to Sender methods should be in progress
// and no further calls should be made.
func (s *Sender) Close(ctx context.Context) {
	s.mu.Lock()
	if s.close.closed {
		s.mu.Unlock()
		return
	}
	s.close.closed = true
	// Unlock goroutines blocked in DiscardEvent or SendEvents.
	s.queue.cond.Broadcast()
	s.mu.Unlock()
	// Stop the loop.
	s.close.stop <- ctx
	<-s.close.done
	// Unregister the Prometheus metrics.
	s.prometheus.queueAvailable.Unregister()
	s.prometheus.queueWait.Unregister()
	return
}

// CreateEvent creates a new event with the given pipeline, type, schema,
// and original event.
//
// The returned event must be passed to SendEvent (optionally after setting
// the Properties field) or to DiscardEvent if it should be discarded.
func (s *Sender) CreateEvent(pipeline int, typ string, schema types.Type, event streams.Event) *Event {
	anonymousID, ok := event.Attributes["anonymousId"].(string)
	if !ok {
		panic("CreateEvent called with an event missing anonymousId")
	}
	s.mu.Lock()
	u, ok := s.users[anonymousID]
	if !ok {
		u = users.Get()
		u.anonymousID = anonymousID
		s.users[anonymousID] = u
	}
	sequence := u.queue.next()
	s.mu.Unlock()
	ev := &Event{
		Event: connectors.Event{
			Received: connections.ReceivedEvent(event.Attributes),
			Type: connectors.EventTypeInfo{
				ID:     typ,
				Schema: schema,
			},
			DestinationPipeline: pipeline,
		},
		createdAt: time.Now().UTC(),
		pipeline:  pipeline,
		user:      u,
		sequence:  sequence,
		ack:       event.Ack,
	}
	return ev
}

// DiscardEvent discards the given event.
// Call this, for example, if the event transformation fails.
//
// DiscardEvent must not be called if DiscardEvent or SendEvent have already
// been called with the same event.
func (s *Sender) DiscardEvent(event *Event) {
	event.discarded = true
	s.processEvent(event)
}

// SendEvent enqueues an event to be sent later, possibly in a batch.
//
// Events from the same user (i.e., with the same anonymousId) are sent in the
// order they were created.
//
// SendEvent must not be called if DiscardEvent or SendEvent have already been
// called with the same event.
func (s *Sender) SendEvent(event *Event) {
	s.processEvent(event)
}

// SetApplication replaces the application.
func (s *Sender) SetApplication(app *connections.Application) {
	s.mu.Lock()
	s.waitTime = app.WaitTime
	s.sendEvents = app.SendEvents
	s.mu.Unlock()
}

// addAvailable adds delta to availability.
//
// It must be called holding the s.mu mutex.
func (s *Sender) addAvailable(delta int) {
	if delta == 0 {
		return
	}
	if delta > 0 && s.available == 0 {
		s.availableSince = time.Now().UTC()
	}
	s.available += delta
	if s.available == 0 {
		s.availableSince = time.Time{}
	}
}

// compact compacts the events. It does nothing if s has been closed.
func (s *Sender) compact() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.close.closed {
		return
	}
	var i int
	for i < len(s.queue.events) && s.queue.events[i] == nil {
		i++
	}
	if i > 0 {
		s.queue.events = slices.Delete(s.queue.events, 0, i)
		if s.iterator != nil {
			s.iterator.index = max(0, s.iterator.index-i)
		}
		trace("Sender.compact: %d events compacted, %d available\n", i, s.available)
		if asserts {
			s._assertAvailable(s.available)
		}
	}
}

// discard discards the most recently read event with the provided error. It is
// invoked when an iterator calls Events.Discard.
func (s *Sender) discard(err error) {
	s.mu.Lock()
	i := s.iterator.index - 1
	var e *Event
	for {
		e = s.queue.events[i]
		if e != nil && e.iterator == s.iterator {
			break
		}
		i--
	}
	// Dequeue the event.
	s.queue.dequeue(i)
	// Update the user.
	u := e.user
	u.totals--
	u.consumed--
	if u.consumed == 0 {
		u.iterator = nil
		s.addAvailable(u.totals)
	}
	if u.disposable() {
		s.disposeUser(u)
	}
	// Update the iterator.
	s.iterator.numConsumed--
	if s.iterator.numConsumed == 0 {
		s.iterator.sameUser.user = nil
	}
	trace("Sender.discard: iterator %p; discard index %d, current %d; pipeline %d, message ID %q\n",
		s.iterator, i, s.iterator.index, e.pipeline, e.Received.MessageID())
	if asserts {
		s._assertAvailable(s.available)
	}
	if s.sent != nil {
		defer s.sent(e.Received.MessageID(), err)
	}
	s.mu.Unlock()
	if s.metrics != nil {
		s.metrics.FinalizeFailed(e.pipeline, 1, err.Error())
	}
}

// disposeUser releases a user instance back to the pool and removes it from the
// Sender. It must be called only after the user no longer owns any events.
// Use u.disposable() to check whether it is safe to dispose the user.
//
// It must be called holding the s.mu mutex.
func (s *Sender) disposeUser(u *user) {
	delete(s.users, u.anonymousID)
	users.Put(u)
}

// iterated marks the iteration of the current iterator as completed, allowing
// other iterators to be executed.
func (s *Sender) iterated() {
	s.mu.Lock()
	trace("Sender.iterated: iteration of iterator %p is completed\n", s.iterator)
	// Update minBatchSize based on the maximum number of events sent in the last iteration.
	// If a new higher value is observed, it is applied immediately.
	// Otherwise, minBatchSize decays slowly over time to adapt to reduced load.
	if n := s.iterator.numConsumed; n > 0 {
		if n > s.minBatchSize {
			// Immediately update to the new maximum.
			s.minBatchSize = n
			trace("Sender.iterated: minBatchSize increased to %d\n", n)
		} else {
			// Slowly decay over time toward smaller values
			decayed := int(0.9*float64(s.minBatchSize) + 0.1*float64(n))
			if decayed < s.minBatchSize {
				s.minBatchSize = decayed
				trace("Sender.iterated: minBatchSize decayed to %d\n", decayed)
			}
		}
		if s.minBatchSize > MaxQueuedEvents {
			s.minBatchSize = MaxQueuedEvents
		}
	}
	s.iterator = nil
	s.releaseUsers()
	s.scheduleIteration()
	s.mu.Unlock()
}

// loop runs the sender iterations according to a schedule.
func (s *Sender) loop() {

	// senders is the waiting group for all send operations.
	var senders sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopCtx context.Context

LOOP:
	for {
		select {
		case <-s.schedule.C:
		case stopCtx = <-s.close.stop:
			break LOOP
		}
		s.mu.Lock()
		var iter *iterator
		var pattern string
		if s.iterator == nil && s.available > 0 {
			s.releaseUsers()
			iter = newIterator(s)
			s.iterator = iter
			pattern = s.rateLimiterPattern
		}
		s.scheduleIteration()
		waitTime := s.waitTime
		s.mu.Unlock()
		if iter == nil {
			continue
		}
		if pattern != "" {
			if d, _ := waitTime(pattern); d > 0 {
				select {
				case <-time.After(d):
				case stopCtx = <-s.close.stop:
					break LOOP
				}
			}
		}
		senders.Go(func() {
			s.send(ctx, iter, pattern)
		})
	}

	// Waits for all send operations to complete.
	trace("wait for senders to terminate\n")
	defer context.AfterFunc(stopCtx, cancel)()
	senders.Wait()
	trace("senders are terminated\n")

	close(s.close.done)

}

// postpone marks the most recently read event as unread. It is invoked when an
// iterator calls Events.Postpone and must be executed with s.mu held.
func (s *Sender) postpone() {
	s.mu.Lock()
	i := s.iterator.index - 1
	var e *Event
	for {
		e = s.queue.events[i]
		if e != nil && e.iterator == s.iterator {
			break
		}
		i--
	}
	e.iterator = nil
	e.user.iterator = postponeMarker
	e.user.consumed--
	s.iterator.numConsumed--
	// Mark the user as releasable if no events were consumed during this iteration.
	if e.user.consumed == 0 {
		s.releasableUsers[e.user] = struct{}{}
	}
	trace("Sender.postpone: iterator %p; postpone index %d, current %d\n", s.iterator, i, s.iterator.index)
	if asserts {
		s._assertAvailable(s.available)
	}
	s.mu.Unlock()
}

// processEvent is invoked by Discard and SendEvent and represents the entry
// point for processing an event previously created by CreateEvent.
func (s *Sender) processEvent(event *Event) {
	u := event.user
	s.mu.Lock()
	event.user.queue.enqueue(event, func(event *Event) bool {
		for s.queue.total >= MaxQueuedEvents {
			if s.close.closed {
				return false
			}
			s.queue.cond.Wait()
		}
		if s.close.closed {
			return false
		}
		s.queue.enqueue(event)
		u.totals++
		if u.iterator == nil {
			s.addAvailable(1)
			if s.iterator == nil {
				s.scheduleIteration()
			}
		}
		return true
	})
	if u.disposable() {
		s.disposeUser(u)
	}
	if asserts {
		s._assertAvailable(s.available)
	}
	s.mu.Unlock()
}

// read reads an event from the queue. If consume is true, the event is removed;
// otherwise, subsequent calls to read will return the same event.
// consume indicates if the event should be consumed.
func (s *Sender) read(consume bool) (*Event, bool) {
	var event *Event
	s.mu.Lock()
	var i int
	for i = s.iterator.index; i < len(s.queue.events); i++ {
		e := s.queue.events[i]
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
		s.queue.events[i].iterator = s.iterator
		s.iterator.index++
		if event.user.consumed == 0 {
			event.user.iterator = s.iterator
			s.addAvailable(-event.user.totals)
		}
		event.user.consumed++
		s.iterator.numConsumed += 1
		if asserts {
			s._assertAvailable(s.available)
		}
	}
	if traces {
		if event != nil {
			messageId := event.Received.MessageID()
			anonymousId := event.Received.AnonymousID()
			if consume {
				trace("Sender.read: iterator %p read and consumed event %q of pipeline %d (anonymousId %q) at index %d (%d available)\n", s.iterator, messageId, event.pipeline, anonymousId, i, s.available)
			} else {
				trace("Sender.read: iterator %p read event %q of pipeline %d (anonymousId %q), without consuming, at index %d (%d available)\n", s.iterator, messageId, event.pipeline, anonymousId, i, s.available)
			}
		} else {
			if consume {
				trace("Sender.read: iterator %p tried to read, with consuming, at index %d, but no record available\n", s.iterator, i)
			} else {
				trace("Sender.read: iterator %p tried to read, without consuming, at index %d, but no record available\n", s.iterator, i)
			}
		}
	}
	s.mu.Unlock()
	if event == nil {
		return nil, false
	}
	return event, true
}

// releaseUsers releases the iterated users, making their events available
// again. They cannot be released while an iteration is in progress.
//
// It must be called holding the s.mu mutex.
func (s *Sender) releaseUsers() {
	if s.iterator != nil {
		panic("core/events/collector/sender: releaseUsers called while an iteration is still in progress")
	}
	if len(s.releasableUsers) == 0 {
		return
	}
	for u := range s.releasableUsers {
		u.iterator = nil
		u.consumed = 0
		s.addAvailable(u.totals)
		if u.disposable() {
			s.disposeUser(u)
		}
	}
	clear(s.releasableUsers)
}

// scheduleIteration computes when to run the next send iteration based on
// current queue availability, batching, and maxQueueDelay.
//
// It must be called holding the s.mu mutex.
func (s *Sender) scheduleIteration() {
	if s.available == 0 {
		s.schedule.Stop()
		return
	}
	// If an iteration is already in progress, do not reschedule.
	if s.iterator != nil {
		return
	}
	// If we have enough events for a batch, send immediately.
	if s.available >= s.minBatchSize {
		s.schedule.Reset(1) // TODO(marco): change 1 with 0. See issue https://github.com/krenalis/krenalis/issues/2122
		return
	}
	// Wait until maxQueueDelay has elapsed since the queue became available.
	elapsed := time.Since(s.availableSince)
	s.schedule.Reset(max(1, maxQueueDelay-elapsed)) // TODO(marco): change 1 with 0. See issue https://github.com/krenalis/krenalis/issues/2122
}

// send sends events to the application by calling the connector's SendEvents
// method.
func (s *Sender) send(ctx context.Context, iter *iterator, rateLimiterPattern string) {

	trace("Sender.send: iterator %p started\n", iter)
	if asserts {
		s.mu.Lock()
		s._assertAvailable(s.available)
		s.mu.Unlock()
	}

	var errEvents connectors.EventsError
	var errRequest error

	// Adds a "RateLimiterPattern" value to the context to receive updates about
	// the rate limiter used by the HTTP client.
	ctx = context.WithValue(ctx,
		httpclient.RateLimiterPatternContextKey,
		httpclient.RateLimiterPatternContextValue{
			Pattern: rateLimiterPattern,
			Set:     s.setRateLimiterPattern,
		})

	s.mu.Lock()
	sendEvents := s.sendEvents
	s.mu.Unlock()

	err := sendEvents(ctx, iter)
	if err != nil {
		if ee, ok := err.(connectors.EventsError); ok {
			errEvents = ee
		} else {
			errRequest = err
			trace("Sender.send: SendEvents returned an error that is not an EventsError: %s\n", err)
		}
	}

	var acks []streams.Ack

	type metricsKey struct {
		pipeline int
		err      error
	}
	var metricsCounts map[metricsKey]int

	s.mu.Lock()
	if s.iterator == iter {
		// SendEvents hasn't started the iteration; mark it as completed.
		s.mu.Unlock()
		trace("Sender.send: SendEvents of iterator %p has returned without starting an iteration, with error %#v\n", iter, err)
		s.iterated()
	} else {
		// SendEvents has completed the iteration.
		trace("Sender.send: SendEvents of iterator %p has returned, with error %#v\n", iter, err)
		if asserts {
			s._assertAvailable(s.available)
		}
		var index int
		for i := 0; i < len(s.queue.events); i++ {
			if s.queue.events[i] == nil || s.queue.events[i].iterator != iter {
				continue
			}
			user := s.queue.events[i].user
			s.releasableUsers[user] = struct{}{}
			err := errRequest
			if errEvents != nil {
				err = errEvents[index]
			}
			if s.metrics != nil {
				key := metricsKey{
					pipeline: s.queue.events[i].pipeline,
					err:      err,
				}
				if metricsCounts == nil {
					metricsCounts = map[metricsKey]int{key: 1}
				} else {
					metricsCounts[key]++
				}
			}
			if s.sent != nil {
				defer s.sent(s.queue.events[i].Received.MessageID(), err)
			}
			user.totals--
			acks = append(acks, s.queue.events[i].ack)
			s.queue.dequeue(i)
			index++
		}
		if s.iterator == nil {
			s.releaseUsers()
			s.scheduleIteration()
		}
		if asserts {
			s._assertAvailable(s.available)
			s._assertQueueTotal(s.queue.total)
		}
		s.mu.Unlock()
	}

	for _, ack := range acks {
		ack()
	}

	for key, count := range metricsCounts {
		trace("Sender.send: collect metric for iterator %p with pipeline %d, count %d, and error %#v\n", iter, key.pipeline, count, key.err)
		if key.err != nil {
			s.metrics.FinalizeFailed(key.pipeline, count, key.err.Error())
			continue
		}
		s.metrics.FinalizePassed(key.pipeline, count)
	}

	s.compact()

}

// setRateLimiterPattern updates the rate limiter pattern.
// It is called by the HTTP client when the rate limiter pattern used for the
// request differs from the one provided in the request's context.
func (s *Sender) setRateLimiterPattern(pattern string) {
	s.mu.Lock()
	s.rateLimiterPattern = pattern
	s.mu.Unlock()
}

// setSentFunc updates the sent function that is called when an event is sent or
// discarded. If nil, no function is called.
func (s *Sender) setSentFunc(f func(messageID string, err error)) {
	s.mu.Lock()
	s.sent = f
	s.mu.Unlock()
}

// _assertAvailable asserts that the available events are n.
//
// It must be called holding the s.mu mutex.
func (s *Sender) _assertAvailable(n int) {
	available := 0
	for _, user := range s.users {
		if user.iterator == nil {
			available += user.totals
		}
	}
	if n != available {
		panic(fmt.Sprintf("core/events/collector/sender: expected %d available, got %d", n, available))
	}
}

// _assertQueueTotal asserts that the queued events are n.
//
// It must be called holding the s.mu mutex.
func (s *Sender) _assertQueueTotal(n int) {
	total := 0
	for _, event := range s.queue.events {
		if event != nil {
			total++
		}
	}
	if n != total {
		panic(fmt.Sprintf("core/events/collector/sender: expected %d queued, got %d", n, total))
	}
}

// newSchedule returns a new stopped timer.
func newSchedule() *time.Timer {
	t := time.NewTimer(math.MaxInt64)
	t.Stop()
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
