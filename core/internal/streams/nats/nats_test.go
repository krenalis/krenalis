// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package nats

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/natsopts"

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

// testJetStreamMsg embeds jetstream.Msg so tests only need to implement the
// acknowledgment methods exercised by ackManager.
type testJetStreamMsg struct {
	jetstream.Msg
	mu         sync.Mutex
	acks       int
	inProgress int
}

func (m *testJetStreamMsg) Ack() error {
	m.mu.Lock()
	m.acks++
	m.mu.Unlock()
	return nil
}

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

func (m *testJetStreamMsg) inProgressCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inProgress
}
