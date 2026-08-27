// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package nats

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/natsopts"

	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestConnectRejectsAckWaitBelowMinimum verifies that Connect rejects AckWait
// values below the configured minimum.
func TestConnectRejectsAckWaitBelowMinimum(t *testing.T) {
	_, err := Connect(natsopts.Options{AckWait: natsopts.MinAckWait - time.Nanosecond})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	want := "NATS consumer AckWait must be at least 1s"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestAckManagerStopsAfterEveryDestinationAcknowledges verifies that
// heartbeats continue until every destination has completed and that the final
// acknowledgment is sent only once.
func TestAckManagerStopsAfterEveryDestinationAcknowledges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager := newAckManager(3 * time.Second)
		defer manager.Close()
		msg := new(testJetStreamMsg)
		destinations := destinationsForMessage(manager.Track(msg), []string{"first", "second"})

		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()
		if got := msg.inProgressCount(); got != 1 {
			t.Fatalf("expected 1 in-progress acknowledgment, got %d", got)
		}

		destinations[0].Ack.Acknowledge()
		destinations[0].Ack.Acknowledge()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if got := msg.ackCount(); got != 0 {
			t.Fatalf("expected no acknowledgment while one destination is pending, got %d", got)
		}
		if got := msg.inProgressCount(); got != 3 {
			t.Fatalf("expected heartbeats to continue until every destination completes, got %d", got)
		}

		destinations[1].Ack.Acknowledge()
		destinations[1].Ack.Acknowledge()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if got := msg.ackCount(); got != 1 {
			t.Fatalf("expected 1 acknowledgment, got %d", got)
		}
		if got := msg.inProgressCount(); got != 3 {
			t.Fatalf("expected in-progress acknowledgments to stop at 3, got %d", got)
		}
	})
}

// TestAckManagerStopsWithoutAcknowledging verifies that closing the manager
// stops heartbeats without acknowledging any tracked messages.
func TestAckManagerStopsWithoutAcknowledging(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager := newAckManager(3 * time.Second)
		msg := new(testJetStreamMsg)
		manager.Track(msg)

		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()
		manager.Close()
		time.Sleep(2 * time.Second)
		synctest.Wait()

		if got := msg.inProgressCount(); got != 1 {
			t.Fatalf("expected 1 in-progress acknowledgment, got %d", got)
		}
		if got := msg.ackCount(); got != 0 {
			t.Fatalf("expected no acknowledgment, got %d", got)
		}
	})
}

// TestAckManagerClearsSnapshotAfterCancellation verifies that a canceled
// heartbeat pass does not retain references to pending acknowledgments.
func TestAckManagerClearsSnapshotAfterCancellation(t *testing.T) {
	manager := newAckManager(time.Hour)
	defer manager.Close()
	manager.Track(new(testJetStreamMsg))

	// Populate the reusable snapshot buffer before canceling the next pass.
	manager.inProgress(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager.inProgress(ctx)

	for i, ack := range manager.pendingSnapshot {
		if ack != nil {
			t.Fatalf("expected acknowledgment %d to be cleared from the snapshot", i)
		}
	}
}

// TestAckStopPreventsDestinationAcknowledgment verifies that stopping the
// parent acknowledgment prevents a later destination from acknowledging the
// NATS message.
func TestAckStopPreventsDestinationAcknowledgment(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager := newAckManager(3 * time.Second)
		defer manager.Close()
		msg := new(testJetStreamMsg)
		parent := manager.Track(msg)
		destinations := destinationsForMessage(parent, nil)

		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()
		parent.Stop()
		destinations[0].Ack.Acknowledge()
		time.Sleep(2 * time.Second)
		synctest.Wait()

		if got := msg.inProgressCount(); got != 1 {
			t.Fatalf("expected 1 in-progress acknowledgment, got %d", got)
		}
		if got := msg.ackCount(); got != 0 {
			t.Fatalf("expected no acknowledgment, got %d", got)
		}
	})
}

// TestConsumerTracksFetchedMessagesWhileDecodingIsBlocked verifies that
// acknowledgment heartbeats are sent for every fetched message while the first
// message is waiting to be decoded. It also verifies that fetching resumes as
// processing makes space in the pending queue.
func TestConsumerTracksFetchedMessagesWhileDecodingIsBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		messages := make([]*testJetStreamMsg, maxMessagesPerFetch+100)
		for i := range messages {
			messages[i] = &testJetStreamMsg{data: fmt.Appendf(nil, `{
				"connectionId":"connection","anonymousId":"anonymous","messageId":"message-%d",
				"receivedAt":"2026-01-01T00:00:00Z","sentAt":"2026-01-01T00:00:00Z",
				"originalTimestamp":"2026-01-01T00:00:00Z","timestamp":"2026-01-01T00:00:00Z",
				"traits":{},"type":"identify"}`, i)}
		}
		decodeReady := make(chan struct{})
		var decodeReadyOnce sync.Once
		releaseDecode := func() {
			decodeReadyOnce.Do(func() {
				close(decodeReady)
			})
		}
		messages[0].dataReady = decodeReady

		pullConsumer := newTestPullConsumer(messages)
		wait := make(chan struct{})
		close(wait)
		s := &stream{ackWait: 3 * time.Second}
		s.js.jetStream = &testJetStream{consumer: pullConsumer}
		s.js.wait = wait
		consumer := s.Consume("test", 1)
		events, err := consumer.Events(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() {
			releaseDecode()
			pullConsumer.Close()
			consumer.Close()
			for range events {
			}
		}()

		synctest.Wait()
		fetched := pullConsumer.fetchedCount()
		if fetched != maxMessagesPerFetch {
			t.Fatalf("expected %d fetched messages while decoding is blocked, got %d", maxMessagesPerFetch, fetched)
		}
		if got := pullConsumer.fetchCount(); got != 1 {
			t.Fatalf("expected 1 fetch while decoding is blocked, got %d", got)
		}

		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()
		for i, msg := range messages {
			want := 0
			if i < fetched {
				want = 1
			}
			if got := msg.inProgressCount(); got != want {
				t.Fatalf("message %d: expected %d in-progress acknowledgments, got %d", i, want, got)
			}
		}

		releaseDecode()
		for i := range messages {
			select {
			case event, ok := <-events:
				if !ok {
					t.Fatalf("expected message %d, got a closed event channel", i)
				}
				want := fmt.Sprintf("message-%d", i)
				if got := event.Attributes["messageId"]; got != want {
					t.Fatalf("expected message ID %q, got %q", want, got)
				}
				event.Destinations[0].Ack.Acknowledge()
			case <-time.After(time.Second):
				t.Fatalf("expected message %d, got none", i)
			}
		}
		if got := pullConsumer.fetchCount(); got < 2 {
			t.Fatalf("expected at least 2 fetches, got %d", got)
		}
		for i, batchSize := range pullConsumer.fetchBatchSizes() {
			if batchSize < minFetchBatchSize {
				t.Fatalf("fetch %d: expected a batch of at least %d messages, got %d", i, minFetchBatchSize, batchSize)
			}
		}
	})
}

// TestDestinationsForMessageCreatesAnonymousDestination verifies that a message
// without explicit destinations receives one anonymous destination.
func TestDestinationsForMessageCreatesAnonymousDestination(t *testing.T) {
	msg := new(testJetStreamMsg)
	destinations := testDestinationsForMessage(t, msg, nil)

	if got := len(destinations); got != 1 {
		t.Fatalf("expected 1 anonymous destination, got %d", got)
	}
	if got := destinations[0].ID; got != "" {
		t.Fatalf("expected an empty anonymous destination ID, got %q", got)
	}

	destinations[0].Ack.Acknowledge()
	destinations[0].Ack.Acknowledge()
	if got := msg.ackCount(); got != 1 {
		t.Fatalf("expected 1 message acknowledgment, got %d", got)
	}
}

// TestDestinationAcksAcknowledgeAfterEveryDestination verifies that a NATS
// message is acknowledged after every destination has completed.
func TestDestinationAcksAcknowledgeAfterEveryDestination(t *testing.T) {
	msg := new(testJetStreamMsg)
	destinations := testDestinationsForMessage(t, msg, []string{"first", "second"})

	if got := len(destinations); got != 2 {
		t.Fatalf("expected 2 destinations, got %d", got)
	}
	if got := destinations[0].ID; got != "first" {
		t.Fatalf("expected first destination ID %q, got %q", "first", got)
	}
	if got := destinations[1].ID; got != "second" {
		t.Fatalf("expected second destination ID %q, got %q", "second", got)
	}

	destinations[0].Ack.Acknowledge()
	destinations[0].Ack.Acknowledge()
	if got := msg.ackCount(); got != 0 {
		t.Fatalf("expected 0 message acknowledgments while one destination is pending, got %d", got)
	}

	destinations[1].Ack.Acknowledge()
	destinations[1].Ack.Acknowledge()
	if got := msg.ackCount(); got != 1 {
		t.Fatalf("expected 1 message acknowledgment after every destination completed, got %d", got)
	}
}

// TestDestinationAcksAreConcurrent verifies that destination acknowledgments
// remain idempotent when destinations complete concurrently.
func TestDestinationAcksAreConcurrent(t *testing.T) {
	msg := new(testJetStreamMsg)
	ids := make([]string, 32)
	for i := range ids {
		ids[i] = fmt.Sprintf("destination%d", i)
	}
	destinations := testDestinationsForMessage(t, msg, ids)

	var wg sync.WaitGroup
	for i := range destinations {
		wg.Go(func() {
			destinations[i].Ack.Acknowledge()
			destinations[i].Ack.Acknowledge()
		})
	}
	wg.Wait()

	if got := msg.ackCount(); got != 1 {
		t.Fatalf("expected 1 concurrent message acknowledgment, got %d", got)
	}
}

// testDestinationsForMessage returns destination acknowledgments tracked by a
// manager that is closed when the test completes.
func testDestinationsForMessage(t *testing.T, msg jetstream.Msg, ids []string) []streams.Destination {
	t.Helper()
	manager := newAckManager(time.Hour)
	t.Cleanup(manager.Close)
	return destinationsForMessage(manager.Track(msg), ids)
}

// TestShardConsumerNacksBufferedMessagesWhenIterationStops verifies that
// messages fetched but not yielded are negatively acknowledged when iteration
// stops.
func TestShardConsumerNacksBufferedMessagesWhenIterationStops(t *testing.T) {
	messages := []*testJetStreamMsg{{}, {}, {}}
	pullConsumer := newTestPullConsumer(messages)
	wait := make(chan struct{})
	close(wait)
	s := &stream{ackWait: 3 * time.Second}
	s.js.jetStream = &testJetStream{consumer: pullConsumer}
	s.js.wait = wait
	c := &consumer{stream: s, topic: "test", acks: newAckManager(s.ackWait)}
	defer c.acks.Close()

	sc := newShardConsumer(c, 0)
	for message := range sc.Messages(context.Background()) {
		message.ack.Stop()
		if err := message.msg.Nak(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// The test pull consumer cannot inspect FetchContext, so release any
		// pending fetch before stopping iteration.
		pullConsumer.Close()
		break
	}

	if got := pullConsumer.fetchedCount(); got != len(messages) {
		t.Fatalf("expected %d fetched messages, got %d", len(messages), got)
	}
	for i, message := range messages {
		if got := message.nakCount(); got != 1 {
			t.Fatalf("message %d: expected 1 negative acknowledgment, got %d", i, got)
		}
	}
}

// TestShardConsumerRecreatesConsumerAfterFetchErrors verifies that the NATS
// consumer is recreated and fetching resumes when Fetch or MessageBatch reports
// that the consumer no longer exists.
func TestShardConsumerRecreatesConsumerAfterFetchErrors(t *testing.T) {
	for _, fromBatch := range []bool{false, true} {
		name := "Fetch"
		if fromBatch {
			name = "MessageBatch"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				message := new(testJetStreamMsg)
				pullConsumer := newTestPullConsumer([]*testJetStreamMsg{message})
				if fromBatch {
					pullConsumer.batchError = jetstream.ErrConsumerDoesNotExist
				} else {
					pullConsumer.fetchError = jetstream.ErrConsumerDoesNotExist
				}
				wait := make(chan struct{})
				close(wait)
				js := &testJetStream{consumer: pullConsumer}
				s := &stream{ackWait: 3 * time.Second}
				s.js.jetStream = js
				s.js.wait = wait
				c := &consumer{stream: s, topic: "test", acks: newAckManager(s.ackWait)}
				defer c.acks.Close()

				sc := newShardConsumer(c, 0)
				for fetched := range sc.Messages(context.Background()) {
					fetched.ack.Stop()
					if err := fetched.msg.Nak(); err != nil {
						t.Fatalf("expected no error, got %v", err)
					}
					pullConsumer.Close()
					break
				}

				if got := js.createCount(); got != 2 {
					t.Fatalf("expected 2 consumer creations, got %d", got)
				}
			})
		})
	}
}

// testJetStream provides the pull consumer used by consumer tests.
type testJetStream struct {
	jetstream.JetStream
	mu       sync.Mutex
	consumer jetstream.Consumer
	creates  int
}

// CreateOrUpdateConsumer returns the configured test pull consumer.
func (js *testJetStream) CreateOrUpdateConsumer(context.Context, string, jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	js.mu.Lock()
	js.creates++
	js.mu.Unlock()
	return js.consumer, nil
}

// createCount returns the number of consumer creations.
func (js *testJetStream) createCount() int {
	js.mu.Lock()
	defer js.mu.Unlock()
	return js.creates
}

// testPullConsumer returns messages in explicitly sized fetch batches.
type testPullConsumer struct {
	jetstream.Consumer
	mu         sync.Mutex // protects the fields below
	messages   []*testJetStreamMsg
	next       int
	batchSizes []int
	batchError error
	fetches    int
	fetchError error
	stop       chan struct{}
	stopOnce   sync.Once
}

// newTestPullConsumer returns a test pull consumer containing messages.
func newTestPullConsumer(messages []*testJetStreamMsg) *testPullConsumer {
	return &testPullConsumer{messages: messages, stop: make(chan struct{})}
}

// Fetch returns at most batch messages. After exhausting the configured
// messages, it waits for Close to simulate a pending fetch.
func (c *testPullConsumer) Fetch(batch int, _ ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	c.mu.Lock()
	c.fetches++
	c.batchSizes = append(c.batchSizes, batch)
	if c.fetchError != nil {
		err := c.fetchError
		c.fetchError = nil
		c.mu.Unlock()
		return nil, err
	}
	if c.batchError != nil {
		err := c.batchError
		c.batchError = nil
		c.mu.Unlock()
		messages := make(chan jetstream.Msg)
		close(messages)
		return &testMessageBatch{messages: messages, err: err}, nil
	}
	start := c.next
	end := min(start+batch, len(c.messages))
	c.next = end
	c.mu.Unlock()

	messages := make(chan jetstream.Msg, end-start)
	for _, msg := range c.messages[start:end] {
		messages <- msg
	}
	if start < end {
		close(messages)
	} else {
		go func() {
			<-c.stop
			close(messages)
		}()
	}
	return &testMessageBatch{messages: messages}, nil
}

// Close releases a fetch waiting for more test messages.
func (c *testPullConsumer) Close() {
	c.stopOnce.Do(func() {
		close(c.stop)
	})
}

// fetchedCount returns the number of messages returned by Fetch.
func (c *testPullConsumer) fetchedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.next
}

// fetchCount returns the number of calls to Fetch.
func (c *testPullConsumer) fetchCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fetches
}

// fetchBatchSizes returns the message limit requested by each call to Fetch.
func (c *testPullConsumer) fetchBatchSizes() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.batchSizes...)
}

// testMessageBatch contains the messages returned by a test fetch operation.
type testMessageBatch struct {
	messages <-chan jetstream.Msg
	err      error
}

// Messages returns the messages in the test batch.
func (b *testMessageBatch) Messages() <-chan jetstream.Msg {
	return b.messages
}

// Error returns the error reported after consuming the test batch.
func (b *testMessageBatch) Error() error {
	return b.err
}

// testJetStreamMsg embeds jetstream.Msg so tests only need to implement the
// methods exercised by consumer and acknowledgment tests.
type testJetStreamMsg struct {
	jetstream.Msg
	mu         sync.Mutex // protects acks, naks, and inProgress
	data       []byte
	dataReady  <-chan struct{}
	acks       int
	naks       int
	inProgress int
}

// Data waits until the test message is ready and returns its body.
func (m *testJetStreamMsg) Data() []byte {
	if m.dataReady != nil {
		<-m.dataReady
	}
	return m.data
}

// Headers returns no headers for the test message.
func (m *testJetStreamMsg) Headers() gonats.Header {
	return nil
}

// Ack records a final acknowledgment.
func (m *testJetStreamMsg) Ack() error {
	m.mu.Lock()
	m.acks++
	m.mu.Unlock()
	return nil
}

// Nak records a negative acknowledgment.
func (m *testJetStreamMsg) Nak() error {
	m.mu.Lock()
	m.naks++
	m.mu.Unlock()
	return nil
}

// InProgress records an acknowledgment heartbeat.
func (m *testJetStreamMsg) InProgress() error {
	m.mu.Lock()
	m.inProgress++
	m.mu.Unlock()
	return nil
}

func (m *testJetStreamMsg) ackCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.acks
}

// nakCount returns the number of negative acknowledgments.
func (m *testJetStreamMsg) nakCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.naks
}

func (m *testJetStreamMsg) inProgressCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inProgress
}
