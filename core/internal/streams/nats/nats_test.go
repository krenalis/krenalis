// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package nats

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
)

// testJetStreamMsg records acknowledgments sent to a JetStream message.
type testJetStreamMsg struct {
	jetstream.Msg
	acknowledgments atomic.Int64
}

// Ack records a JetStream message acknowledgment.
func (m *testJetStreamMsg) Ack() error {
	m.acknowledgments.Add(1)
	return nil
}

// TestDestinationsForMessageCreatesAnonymousDestination verifies that a message
// without explicit destinations receives one anonymous destination.
func TestDestinationsForMessageCreatesAnonymousDestination(t *testing.T) {
	msg := new(testJetStreamMsg)
	destinations := destinationsForMessage(msg, nil)

	if got := len(destinations); got != 1 {
		t.Fatalf("expected 1 anonymous destination, got %d", got)
	}
	if got := destinations[0].ID; got != "" {
		t.Fatalf("expected an empty anonymous destination ID, got %q", got)
	}

	destinations[0].Ack.Acknowledge()
	destinations[0].Ack.Acknowledge()
	if got := msg.acknowledgments.Load(); got != 1 {
		t.Fatalf("expected 1 message acknowledgment, got %d", got)
	}
}

// TestDestinationAcksAcknowledgeAfterEveryDestination verifies that a NATS
// message is acknowledged after every destination has completed.
func TestDestinationAcksAcknowledgeAfterEveryDestination(t *testing.T) {
	msg := new(testJetStreamMsg)
	destinations := destinationsForMessage(msg, []string{"first", "second"})

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
	if got := msg.acknowledgments.Load(); got != 0 {
		t.Fatalf("expected 0 message acknowledgments while one destination is pending, got %d", got)
	}

	destinations[1].Ack.Acknowledge()
	destinations[1].Ack.Acknowledge()
	if got := msg.acknowledgments.Load(); got != 1 {
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
	destinations := destinationsForMessage(msg, ids)

	var wg sync.WaitGroup
	for i := range destinations {
		wg.Go(func() {
			destinations[i].Ack.Acknowledge()
			destinations[i].Ack.Acknowledge()
		})
	}
	wg.Wait()

	if got := msg.acknowledgments.Load(); got != 1 {
		t.Fatalf("expected 1 concurrent message acknowledgment, got %d", got)
	}
}
