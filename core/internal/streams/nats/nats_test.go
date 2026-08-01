// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package nats

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"

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

// TestAckManagerStopsAfterAcknowledgment verifies that acknowledging a message
// stops its heartbeats and sends its final acknowledgment only once.
func TestAckManagerStopsAfterAcknowledgment(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		manager := newAckManager(3 * time.Second)
		defer manager.Close()
		msg := &testJetStreamMsg{}
		a := manager.Track(msg)

		time.Sleep(3500 * time.Millisecond)
		synctest.Wait()
		if got := msg.inProgressCount(); got != 3 {
			t.Fatalf("expected 3 in-progress acknowledgments, got %d", got)
		}

		a.Acknowledge()
		a.Acknowledge()
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
		msg := &testJetStreamMsg{}
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
