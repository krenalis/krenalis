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
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/connections/httpclient"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/prometheus"
	"github.com/meergo/meergo/tools/types"
)

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
	CreatedAt        time.Time   // time at which the event was created.
	EnqueuedAt       time.Time   // time at which the event was enqueued.
	pipeline         int         // pipeline ID.
	user             *user       // associated user; nil if DiscardEvent was called (the event was never queued).
	sequence         int         // sequence number; access is synchronized via Sender.mu.
	iterator         *iterator   // iterator that consumed the event; nil if it hasn't been consumed.
	ack              streams.Ack // event ack.
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
// It ensures that events from the same user (i.e., with the same Anonymous ID)
// are delivered to the application in the exact order they were received.
//
// To send an event, follow these steps:
//  1. Call the CreateEvent method. The order in which this is called determines
//     delivery order.
//  2. (Optional) Transform the event and set the Event.Properties field.
//  3. Call the QueueEvent method.
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
	events             []*Event                                                  // events in the queue; protected by mu.
	users              map[string]*user                                          // users by anonymous id; protected by mu.
	releasableUsers    map[*user]struct{}                                        // users that have been iterated and are now ready to be released.
	iterator           *iterator                                                 // current iterator; protected by mu.
	available          int                                                       // number of available (non-read) records; protected by mu.
	index              int                                                       // starting index for the next iteration; all events before this index are unavailable; protected by mu.
	timer              *time.Timer                                               // timer to trigger an iterator every maxQueueDelay; protected by mu.
	minBatchSize       int                                                       // minimum number of events in the queue required to trigger a new iteration.
	rateLimiterPattern string                                                    // pattern of the rate limiter that defines how requests are throttled over time.
	sent               func(messageID string, err error)                         // function called in tests when an event is sent or discarded.

	close struct {
		closed    atomic.Bool        // indicates if the writer has been closed.
		ctx       context.Context    // context passes to iterators.
		cancel    context.CancelFunc // function to cancel iterators executions.
		completed sync.Cond          // signal the completion of the current iteration.
		iterators sync.WaitGroup     // waiting group for the iterators.
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
			case <-s.close.ctx.Done():
				return
			}
			var iter *iterator
			var pattern string
			s.mu.Lock()
			if s.iterator == nil && s.available > 0 {
				s.releaseUsers()
				iter = newIterator(s)
				iter.index = s.index
				s.iterator = iter
				pattern = s.rateLimiterPattern
			}
			s.resetTimerLocked()
			waitTime := app.WaitTime
			s.mu.Unlock()
			if iter == nil {
				continue
			}
			if pattern != "" {
				if d, _ := waitTime(pattern); d > 0 {
					select {
					case <-time.After(d):
					case <-s.close.ctx.Done():
						return
					}
				}
			}
			s.close.iterators.Add(1)
			go s.send(iter, pattern)
		}
	}()
	// Set the metrics.
	connection := strconv.Itoa(app.ID())
	s.prometheus.queueAvailable = queueAvailableMetric.Register(func() float64 {
		s.mu.Lock()
		a := s.available
		s.mu.Unlock()
		return float64(a)
	}, s.connector, connection)
	s.prometheus.queueWait = queueWaitMetric.Register(s.connector, connection)
	return s
}

func (s *Sender) Available() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.available
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
		var pattern string
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
			pattern = s.rateLimiterPattern
			trace("Sender.Close: %d events available; create new iterator %p\n", s.available, iter)
		}
		waitTime := s.waitTime
		s.mu.Unlock()
		if iter == nil {
			break
		}
		if pattern != "" {
			if d, _ := waitTime(pattern); d != 0 {
				select {
				case <-time.After(d):
				case <-s.close.ctx.Done():
				}
				if s.close.ctx.Err() != nil {
					break
				}
			}
		}
		s.close.iterators.Add(1)
		go s.send(iter, pattern)
	}
	trace("Writer.Close: wait for iterators to terminate\n")
	s.close.iterators.Wait()
	if asserts && ctx.Done() == nil {
		s._assertAvailable(0)
	}
	trace("Writer.Close: iterators are terminated; writer is now closed\n")
	s.prometheus.queueAvailable.Unregister()
	s.prometheus.queueWait.Unregister()
	return nil
}

// CreateEvent creates a new event with the given pipeline, type, schema,
// and original event.
//
// The returned event must be passed to QueueEvent (optionally after setting
// the Properties field) or to DiscardEvent if it should be discarded.
func (s *Sender) CreateEvent(pipeline int, typ string, schema types.Type, event streams.Event) *Event {
	anonymousID, ok := event.Attributes["anonymousId"].(string)
	if !ok {
		panic("core/events/connector/sender: missing anonymousId")
	}
	s.mu.Lock()
	u, ok := s.users[anonymousID]
	if !ok {
		u = &user{}
		s.users[anonymousID] = u
	}
	seq := u.nextSeq
	u.nextSeq++
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
		CreatedAt: time.Now().UTC(),
		pipeline:  pipeline,
		user:      u,
		sequence:  seq,
		ack:       event.Ack,
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

// SetApplication replaces the application.
func (s *Sender) SetApplication(app *connections.Application) {
	s.mu.Lock()
	s.waitTime = app.WaitTime
	s.sendEvents = app.SendEvents
	s.mu.Unlock()
}

// appendToReadyQueue adds the given event to the ready queue.
//
// It must be called holding the s.mu mutex.
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
//
// It must be called holding the s.mu mutex.
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

// discard discards the most recently read event with the provided error. It is
// invoked when an iterator calls Events.Discard.
func (s *Sender) discard(err error) {
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
	// update the sender.
	s.events[i] = nil
	// Update the user.
	e.user.events--
	e.user.numConsumed--
	if e.user.numConsumed == 0 {
		e.user.iterator = nil
		s.available += e.user.events
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

// postpone marks the most recently read event as unread. It is invoked when an
// iterator calls Events.Postpone and must be executed with s.mu held.
func (s *Sender) postpone() {
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
	e.user.iterator = postponeMarker
	e.user.numConsumed--
	s.iterator.numConsumed--
	// Mark the user as releasable if no events were consumed during this iteration.
	if e.user.numConsumed == 0 {
		s.releasableUsers[e.user] = struct{}{}
	}
	trace("Sender.postpone: iterator %p; postpone index %d, current %d\n", s.iterator, i, s.iterator.index)
	if asserts {
		s._assertAvailable(s.available)
	}
	s.mu.Unlock()
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
			if u.waiting[last].user != nil {
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
	} else {
		event.EnqueuedAt = time.Now().UTC()
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
//
// It must be called holding the s.mu mutex.
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

// send sends events to the application by calling the connector's SendEvents
// method.
func (s *Sender) send(iter *iterator, rateLimiterPattern string) {

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
	ctx := context.WithValue(s.close.ctx,
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
		s.complete()
	} else {
		// SendEvents has completed the iteration.
		trace("Sender.send: SendEvents of iterator %p has returned, with error %#v\n", iter, err)
		if asserts {
			s._assertAvailable(s.available)
		}
		var index int
		for i := 0; i < len(s.events); i++ {
			if s.events[i] == nil || s.events[i].iterator != iter {
				continue
			}
			user := s.events[i].user
			s.releasableUsers[user] = struct{}{}
			err := errRequest
			if errEvents != nil {
				err = errEvents[index]
			}
			if s.metrics != nil {
				key := metricsKey{
					pipeline: s.events[i].pipeline,
					err:      err,
				}
				if metricsCounts == nil {
					metricsCounts = map[metricsKey]int{key: 1}
				} else {
					metricsCounts[key]++
				}
			}
			if s.sent != nil {
				defer s.sent(s.events[i].Received.MessageID(), err)
			}
			user.events--
			acks = append(acks, s.events[i].ack)
			s.events[i] = nil
			index++
		}
		if s.iterator == nil {
			s.releaseUsers()
			s.resetTimerLocked()
		}
		if asserts {
			s._assertAvailable(s.available)
		}
		s.mu.Unlock()
	}

	s.close.iterators.Done()
	s.compact()

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
