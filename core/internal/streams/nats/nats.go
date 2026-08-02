// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package nats

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"hash/fnv"
	"iter"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/natsopts"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/base58"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

const numShards = 1

// stream implements streams.Stream.
type stream struct {
	nc *nats.Conn

	ackWait time.Duration

	mu sync.RWMutex

	// up tracks whether the stream is considered up and available.
	//
	// It combines:
	//   - an atomic boolean indicating whether the stream exists and the NATS connection is up;
	//   - a wait channel on which goroutines calling WaitUp block while the stream is down;
	//   - a timer used to close the wait channel after a short grace period since the last
	//     transition to the down state.
	up struct {
		atomic.Bool               // true if the stream exists and the NATS connection is up
		wait        chan struct{} // channel used by WaitUp callers
		timer       *time.Timer   // wakes WaitUp callers after a short delay
	}

	js struct {
		jetStream jetstream.JetStream
		stream    jetstream.Stream
		// Channel used by Stream callers to wait for the stream to become available.
		// It is closed when the stream is ready or the stream is closed.
		wait chan struct{}
		// cancel stops the goroutine running ensureEventStream.
		// If it is nil, ensureEventStream has not been called yet.
		cancel context.CancelFunc
	}

	// closed indicates whether Close has been called.
	closed bool
}

// streamOptions holds the options used by ensureEventStream when creating or
// updating the stream.
type streamOptions struct {
	replicas    int // number of replicas (0–5)
	storage     jetstream.StorageType
	compression jetstream.StoreCompression
}

// Connect establishes a NATS connection to the configured servers and returns
// corresponding stream.
//
// The NATS connection uses the configured User, Password, or Token
// authentication fields when present. If options.NKey is set, NKey
// authentication is also used, performing a challenge-response signature with
// the provided Ed25519 private key.
//
// When NKey authentication is enabled, the private key is defensively copied
// and retained only within the authentication callbacks. The copied key
// material is wiped on a best-effort basis when the NATS connection is
// definitively closed or if the initial connection attempt fails.
func Connect(options natsopts.Options) (streams.Stream, error) {

	ackWait := options.AckWait
	if ackWait == 0 {
		ackWait = natsopts.DefaultAckWait
	} else if ackWait < natsopts.MinAckWait {
		return nil, fmt.Errorf("NATS consumer AckWait must be at least %s", natsopts.MinAckWait)
	}

	s := &stream{ackWait: ackWait}
	s.js.wait = make(chan struct{})
	nKeyConnected := false

	opts := nats.Options{
		Servers:  options.Servers,
		User:     options.User,
		Password: options.Password,
		Token:    options.Token,

		// With these options enabled (provided at least one valid server URL is configured),
		// the NATS connection retries indefinitely, including on ErrNoServers and authentication failures.
		RetryOnFailedConnect: true,
		IgnoreAuthErrorAbort: true,
		AllowReconnect:       true,
		MaxReconnect:         -1,

		ReconnectWait:      nats.DefaultReconnectWait,
		ReconnectJitter:    nats.DefaultReconnectJitter,
		ReconnectJitterTLS: nats.DefaultReconnectJitterTLS,
		Timeout:            nats.DefaultTimeout,
		PingInterval:       nats.DefaultPingInterval,
		MaxPingsOut:        nats.DefaultMaxPingOut,
		SubChanLen:         nats.DefaultMaxChanLen,
		ReconnectBufSize:   nats.DefaultReconnectBufSize,
		DrainTimeout:       nats.DefaultDrainTimeout,
		FlusherTimeout:     nats.DefaultFlusherTimeout,
	}

	streamOpts := streamOptions{
		replicas:    options.Replicas,
		storage:     options.Storage,
		compression: options.Compression,
	}

	opts.ConnectedCB = func(nc *nats.Conn) {
		slog.Info("connected to NATS server")
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return
		}
		// ConnectedCB may be called before nats.Options.Connect returns,
		// so set nc now because ensureEventStream depends on it.
		s.nc = nc
		// Start ensureEventStream on first connection, unless the stream
		// has already been closed or the goroutine is already running.
		if s.js.cancel == nil {
			ctx, cancel := context.WithCancel(context.Background())
			s.js.cancel = cancel
			go s.ensureEventStream(ctx, streamOpts)
		}
		s.mu.Unlock()
		s.refreshUpState()
	}

	// ReconnectErrCB is invoked before the first successful connection,
	// on each failed connection attempt.
	var warned atomic.Bool
	opts.ReconnectErrCB = func(nc *nats.Conn, err error) {
		if warned.CompareAndSwap(false, true) {
			slog.Warn("failed to connect to NATS server; retrying", "error", err)
		}
	}

	// DisconnectedErrCB is invoked whenever a disconnection occurs.
	opts.DisconnectedErrCB = func(_ *nats.Conn, err error) {
		const msg = "disconnected from NATS server; retrying"
		if err == nil {
			slog.Info(msg)
		} else {
			slog.Info(msg, "error", err)
		}
		s.refreshUpState()
	}

	// ReconnectedCB is invoked whenever a reconnection occurs.
	opts.ReconnectedCB = func(*nats.Conn) {
		slog.Info("reconnected to NATS server")
		s.refreshUpState()
	}

	if options.NKey != nil {

		// Copy the private key bytes so future modifications to conf.NKey by caller won't affect the signer.
		pk := slices.Clone(options.NKey)
		defer func() {
			if !nKeyConnected {
				clear(pk)
			}
		}()

		pub, err := nkeys.Encode(nkeys.PrefixByteUser, pk.Public().(ed25519.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("cannot encode NATS public NKey: %w", err)
		}

		opts.Nkey = string(pub)
		opts.SignatureCB = func(nonce []byte) ([]byte, error) {
			if len(nonce) == 0 {
				return nil, errors.New("nonce cannot be empty")
			}
			return ed25519.Sign(pk, nonce), nil
		}
		opts.ClosedCB = func(_ *nats.Conn) { clear(pk) }

	}

	// Update "up" to create the wait channel.
	s.refreshUpState()

	// Connect to the NATS server.
	nc, err := opts.Connect()
	if err != nil {
		return nil, fmt.Errorf("invalid options provided for NATS initialization: %s", err)
	}
	// After Connect returns, callbacks and stream-creation goroutines may run
	// concurrently, so all access to shared fields must be mutex-protected.
	s.mu.Lock()
	if s.nc == nil {
		s.nc = nc
	}
	s.mu.Unlock()
	if options.NKey != nil {
		nKeyConnected = true
	}

	return s, nil
}

// Close closes the stream. When Close is called, no other calls to the
// stream's methods should be in progress and no other shall be made.
func (s *stream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	// If the stream is not yet created and goroutines may still be waiting,
	// close the channel to unblock them (do not set it to nil).
	if s.js.stream == nil {
		close(s.js.wait)
		// If ensureEventStream has been called, cancel it if it is still running.
		if s.js.cancel != nil {
			s.js.cancel()
		}
	}
	if s.up.wait != nil {
		close(s.up.wait)
		s.up.timer.Stop()
		s.up.timer = nil
	}
	// Drain and close the NATS connection.
	err := s.nc.Drain()
	s.nc.Close()
	s.mu.Unlock()
	slog.Info("NATS connection closed")
	return err
}

// WaitUp blocks until the stream is up and available.
// It returns false if the context is canceled, the stream is closed, or the
// stream remains down for too long.
func (s *stream) WaitUp(ctx context.Context) bool {
	if s.up.Load() {
		return true
	}
	s.mu.RLock()
	wait := s.up.wait
	s.mu.RUnlock()
	if wait == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-wait:
		return s.up.Load()
	}
}

// waitStream blocks until the stream has been created. It returns an error only
// if ctx is canceled or the stream has been closed.
func (s *stream) waitStream(ctx context.Context) error {
	select {
	case <-s.js.wait:
	case <-ctx.Done():
		return ctx.Err()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return errors.New("stream has been closed")
	}
	return nil
}

// ensureEventStream initializes the JetStream context and ensures that the
// EVENTS stream exists.
//
// It is run in its own goroutine when the first connection is established.
func (s *stream) ensureEventStream(ctx context.Context, opts streamOptions) {

	js, err := jetstream.New(s.nc)
	if err != nil {
		// jetstream.New can only fail if invalid options are provided;
		// since no options are passed here, this error is unexpected.
		panic(fmt.Sprintf("jetstream.New failed unexpectedly: %v", err))
	}

	cfg := jetstream.StreamConfig{
		Name:        "EVENTS",
		Subjects:    []string{"events.v1.>"},
		Replicas:    opts.replicas,
		Retention:   jetstream.WorkQueuePolicy,
		Storage:     opts.storage,
		Compression: opts.compression,
	}

	bo := backoff.New(10)
	var jsStream jetstream.Stream

	var lastLogMsg, lastLogErr string

	// Create the stream if it does not exist.
	// Exit the loop once the stream is created, already exists,
	// or the context has been canceled.
	for bo.Next(ctx) {
		jsStream, err = js.UpdateStream(ctx, cfg)
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			jsStream, err = js.CreateStream(ctx, cfg)
			if err == nil {
				slog.Info("EVENTS stream has been created")
			}
		}
		if err != nil {
			switch {
			case errors.Is(err, nats.ErrConnectionClosed):
				continue
			case errors.Is(err, jetstream.ErrJetStreamNotEnabledForAccount):
				msg := "JetStream not enabled for this account; waiting for availability"
				if msg != lastLogMsg {
					slog.Warn(msg)
					lastLogMsg, lastLogErr = msg, ""
				}
				continue
			case
				errors.Is(err, jetstream.ErrJetStreamNotEnabled),
				errors.Is(err, nats.ErrNoResponders):
				msg := "JetStream not enabled; waiting for availability"
				if msg != lastLogMsg {
					slog.Warn(msg)
					lastLogMsg, lastLogErr = msg, ""
				}
				continue
			default:
				if ctx.Err() == nil {
					msg := "cannot update or create stream"
					errMsg := err.Error()
					if msg != lastLogMsg || errMsg != lastLogErr {
						slog.Warn(msg, "err", err)
						lastLogMsg, lastLogErr = msg, errMsg
					}
					continue
				}
			}
		}
		if lastLogMsg != "" {
			slog.Info("JetStream became available")
		}
		break
	}

	s.mu.Lock()
	// Return immediately if the stream has already been closed.
	if s.closed {
		s.mu.Unlock()
		return
	}
	// Cancel the context to release resources.
	s.js.cancel()
	// Update the JetStream context and stream handle.
	s.js.jetStream = js
	s.js.stream = jsStream
	// Close js.wait to unblock any goroutines waiting for the stream.
	// Do not set it to nil: closing is the signal.
	close(s.js.wait)
	s.mu.Unlock()

	// Update the "up" state now that the stream is available.
	s.refreshUpState()

}

// refreshUpState updates the "up" state based on NATS connection status and
// stream availability. It does nothing if the stream is closed.
func (s *stream) refreshUpState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if s.nc != nil {
		up := s.js.stream != nil && s.nc.IsConnected()
		if !s.up.Bool.CompareAndSwap(!up, up) {
			return
		}
		if up || s.closed {
			if s.up.wait == nil {
				return
			}
			close(s.up.wait)
			s.up.wait = nil
			s.up.timer.Stop()
			s.up.timer = nil
			return
		}
	}
	s.up.wait = make(chan struct{})
	s.up.timer = time.AfterFunc(200*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.up.timer == nil {
			return
		}
		close(s.up.wait)
		s.up.wait = nil
		s.up.timer.Stop()
		s.up.timer = nil
	})
}

// Batch returns a batch publisher for the stream.
//
// It blocks until the stream has been created. It returns an error only if ctx
// is canceled or the stream has been closed.
func (s *stream) Batch(ctx context.Context) (streams.BatchPublisher, error) {
	err := s.waitStream(ctx)
	if err != nil {
		return nil, err
	}
	return &batch{stream: s, futures: make([]jetstream.PubAckFuture, 0, 1)}, nil
}

// Consume returns a buffered channel of the given size that streams events for
// the specified topic. Events belonging to the same shard are sent on the
// channel in order, ensuring per-user ordering is preserved.
func (s *stream) Consume(topic string, size int) streams.Consumer {
	return newConsumer(s, topic, size)
}

// consumer implements the streams.Consumer interface.
type consumer struct {
	stream *stream
	topic  string
	events chan streams.Event
	acks   *ackManager
	close  struct {
		cancel context.CancelFunc // stops shard consumption
		closed atomic.Bool        // indicates whether Close has been called
		sync.WaitGroup
	}
}

// newConsumer creates a consumer and starts consuming events from topic.
func newConsumer(s *stream, topic string, size int) *consumer {
	ctx, cancel := context.WithCancel(context.Background())
	c := &consumer{
		stream: s,
		topic:  topic,
		events: make(chan streams.Event, size),
		acks:   newAckManager(s.ackWait),
	}
	c.close.cancel = cancel
	for shard := range numShards {
		c.close.Go(func() {
			sc := newShardConsumer(c, shard)
			for message := range sc.Messages(ctx) {
				c.processMessage(ctx, message.msg, message.ack)
			}
		})
	}
	return c
}

// Close closes the consumer and its events channel. When Close is called, no
// calls to other consumer methods should be in progress, and no other methods
// should be called afterward.
//
// Close is idempotent; subsequent calls have no effect.
func (c *consumer) Close() {
	if c.close.closed.Swap(true) {
		return
	}
	c.close.cancel()
	c.close.Wait()
	c.acks.Close()
	close(c.events)
}

// Events returns the events channel.
//
// It blocks until the stream has been created. It returns an error only if ctx
// is canceled or the stream has been closed.
func (c *consumer) Events(ctx context.Context) (<-chan streams.Event, error) {
	err := c.stream.waitStream(ctx)
	if err != nil {
		return nil, err
	}
	return c.events, nil
}

// processMessage decodes and validates a NATS message, then delivers the
// resulting event to the consumer. If ctx is canceled, it stops acknowledgment
// tracking and negatively acknowledges the message.
func (c *consumer) processMessage(ctx context.Context, msg jetstream.Msg, eventAck *ack) {
	select {
	case <-ctx.Done():
		eventAck.Stop()
		if err := msg.Nak(); err != nil {
			slog.Warn("cannot send nack for a NATS message", "error", err)
		}
		return
	default:
	}

	var event streams.Event
	var err error
	defer func() {
		if err != nil {
			eventAck.Stop()
			termErr := msg.TermWithReason(err.Error())
			if termErr != nil {
				slog.Warn("collector: cannot terminate invalid event", "error", termErr)
			}
		}
	}()
	event.Attributes, err = types.Decode[map[string]any](bytes.NewReader(msg.Data()), schemas.Event)
	if err != nil {
		err = fmt.Errorf("invalid event data: %s", err)
		return
	}
	var destinations []string
	if header := msg.Headers(); header != nil {
		if ids, ok := header["destinations"]; ok {
			for _, id := range ids {
				if !isValidDestination(id) {
					err = fmt.Errorf("invalid event destination: %q", id)
					return
				}
			}
			destinations = ids
		}
	}
	event.Destinations = destinationsForMessage(eventAck, destinations)
	select {
	case c.events <- event:
	case <-ctx.Done():
		eventAck.Stop()
		if err := msg.Nak(); err != nil {
			slog.Warn("cannot send nack for a NATS message", "error", err)
		}
	}
}

// maxMessagesPerFetch preserves the NATS client's default pull size while
// bounding the number of messages waiting to be processed for each shard.
const maxMessagesPerFetch = jetstream.DefaultMaxMessages

// fetchedMsg represents a message fetched from a stream for one shard.
type fetchedMsg struct {
	msg jetstream.Msg
	ack *ack
}

// shardConsumer fetches and buffers messages for one shard.
type shardConsumer struct {
	consumer *consumer
	shard    int
}

// newShardConsumer returns a consumer for one shard.
func newShardConsumer(consumer *consumer, shard int) *shardConsumer {
	return &shardConsumer{consumer: consumer, shard: shard}
}

// Messages returns an iterator over messages fetched from the shard.
func (sc *shardConsumer) Messages(ctx context.Context) iter.Seq[fetchedMsg] {
	return func(yield func(fetchedMsg) bool) {
		if err := sc.consumer.stream.waitStream(ctx); err != nil {
			return
		}

		fetchCtx, cancel := context.WithCancel(ctx)
		pending := make(chan fetchedMsg, maxMessagesPerFetch)
		// spaceAvailable wakes the fetch loop when processing frees a full queue.
		spaceAvailable := make(chan struct{}, 1)
		go sc.fetch(fetchCtx, pending, spaceAvailable)
		defer func() {
			cancel()
			for message := range pending {
				message.ack.Stop()
				if err := message.msg.Nak(); err != nil {
					slog.Warn("cannot send nack for a NATS message", "error", err)
				}
			}
		}()

		for message := range pending {
			if !yield(message) {
				return
			}
			select {
			case spaceAvailable <- struct{}{}:
			default:
			}
		}
	}
}

// fetch keeps pending filled with tracked messages until fetching stops.
func (sc *shardConsumer) fetch(ctx context.Context, pending chan<- fetchedMsg, spaceAvailable <-chan struct{}) {
	defer close(pending)

	c := sc.consumer
	consumerName := "EVENTS_" + c.topic + "_" + strconv.Itoa(sc.shard)
	filterSubject := "events.v1." + c.topic + "." + strconv.Itoa(sc.shard)
	bo := backoff.New(10)

	for bo.Next(ctx) {
		jsConsumer, err := c.stream.js.jetStream.CreateOrUpdateConsumer(ctx, "EVENTS", jetstream.ConsumerConfig{
			Name:          consumerName,
			Durable:       consumerName,
			FilterSubject: filterSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       c.stream.ackWait,
			MaxDeliver:    -1,
			MaxAckPending: -1,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("cannot create or update a NATS consumer", "name", consumerName)
			continue
		}

		for ctx.Err() == nil {
			// Do not fetch until every returned message is guaranteed to fit in
			// pending without waiting for the preceding message to be processed.
			for len(pending) == cap(pending) {
				select {
				case <-spaceAvailable:
				case <-ctx.Done():
					return
				}
			}

			batchSize := cap(pending) - len(pending)
			batch, err := jsConsumer.Fetch(batchSize, jetstream.FetchContext(ctx))
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if !errors.Is(err, jetstream.ErrConsumerDoesNotExist) {
					slog.Warn("failed to fetch messages from NATS consumer", "consumer", consumerName, "error", err)
				}
				break
			}

			for msg := range batch.Messages() {
				// batchSize reserves enough queue capacity for the whole batch.
				pending <- fetchedMsg{msg: msg, ack: c.acks.Track(msg)}
			}
			if ctx.Err() != nil {
				return
			}
			if err := batch.Error(); err != nil {
				if !errors.Is(err, jetstream.ErrConsumerDoesNotExist) {
					slog.Warn("failed to fetch messages from NATS consumer", "consumer", consumerName, "error", err)
				}
				break
			}

			// A completed fetch marks the end of a sequence of transient errors.
			// TODO(marco): add a Reset method to backoff.Backoff and replace the
			// following assignment with bo.Reset().
			bo = backoff.New(10)
		}
	}
}

// batch implements the streams.Batch interface.
type batch struct {
	stream  *stream
	futures []jetstream.PubAckFuture
}

// Done publishes all buffered events.
//
// If Done returns nil, all events in the batch have been successfully
// published. If Done returns an error, no guarantees are made about whether
// or how many events have been published.
//
// After Done returns, the BatchPublisher must not be reused.
func (batch *batch) Done(ctx context.Context) error {
	// TODO(marco): future.Ok() and future.Err() creates new channels for every call. Use jetstream.WithPublishAsyncErrHandler instead.
	for _, future := range batch.futures {
		select {
		case <-future.Ok():
			// ok
		case err := <-future.Err():
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Publish adds an event to the current batch for the given topic.
// If the topic begins with "connection-", destinations contains the destination
// pipelines the event is sent to.
func (batch *batch) Publish(ctx context.Context, topics []string, event map[string]any, destinations []string) error {
	shard := shardOf(event["anonymousId"].(string))
	data, err := types.Marshal(event, schemas.Event)
	if err != nil {
		return err
	}
	for _, topic := range topics {
		var header nats.Header
		if strings.HasPrefix(topic, "connection-") {
			header = nats.Header{"destinations": slices.Clone(destinations)}
		}
		future, err := batch.stream.js.jetStream.PublishMsgAsync(&nats.Msg{
			Header:  header,
			Subject: "events.v1." + topic + "." + strconv.Itoa(shard),
			Data:    data,
		})
		if err != nil {
			return err
		}
		batch.futures = append(batch.futures, future)
	}
	return nil
}

// heartbeatsPerAckWait controls the heartbeat frequency. A JetStream
// InProgress heartbeat is sent every AckWait / heartbeatsPerAckWait.
const heartbeatsPerAckWait = 3

// ackManager tracks message acknowledgments that require periodic heartbeats.
type ackManager struct {
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	mu              sync.Mutex
	pending         map[*ack]struct{} // message acknowledgments being tracked; protected by mu
	pendingSnapshot []*ack
	ackErrors       struct {
		sync.Mutex   // protects message, occurrences, and lastLoggedAt
		message      string
		occurrences  int
		lastLoggedAt time.Time
	}
}

// newAckManager returns an acknowledgment manager that sends heartbeats at
// intervals derived from ackWait.
func newAckManager(ackWait time.Duration) *ackManager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &ackManager{
		cancel:  cancel,
		pending: make(map[*ack]struct{}),
	}
	m.wg.Go(func() {
		m.run(ctx, ackWait/heartbeatsPerAckWait)
	})
	return m
}

// Close stops sending acknowledgment heartbeats and clears pending entries.
func (m *ackManager) Close() {
	m.cancel()
	m.wg.Wait()
	m.mu.Lock()
	clear(m.pending)
	m.mu.Unlock()
}

// Remove stops tracking a message acknowledgment.
func (m *ackManager) Remove(a *ack) {
	m.mu.Lock()
	delete(m.pending, a)
	m.mu.Unlock()
}

// Track starts tracking a message until it is acknowledged or tracking is
// stopped.
func (m *ackManager) Track(msg jetstream.Msg) *ack {
	a := &ack{msg: msg, manager: m, remaining: 1}
	m.mu.Lock()
	m.pending[a] = struct{}{}
	m.mu.Unlock()
	return a
}

// inProgress sends InProgress heartbeats for all tracked acknowledgments.
func (m *ackManager) inProgress(ctx context.Context) {
	// Take a snapshot so heartbeats can be sent without holding the lock.
	m.mu.Lock()
	if len(m.pending) == 0 {
		m.mu.Unlock()
		return
	}
	pending := slices.Grow(m.pendingSnapshot[:0], len(m.pending))
	for ack := range m.pending {
		pending = append(pending, ack)
	}
	m.mu.Unlock()
	defer func() {
		// Reuse the snapshot's backing array on the next call without retaining
		// references to acknowledgments after this pass returns.
		clear(pending)
		m.pendingSnapshot = pending
	}()

	var lastErrMsg string

	// Notify NATS that each tracked message is still being processed.
	for _, ack := range pending {
		if ctx.Err() != nil {
			return
		}
		err := ack.InProgress()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgAlreadyAckd) {
				m.Remove(ack)
				continue
			}
			if errMsg := err.Error(); errMsg != lastErrMsg {
				slog.Warn("cannot notify NATS that the message is still being processed", "error", errMsg)
				lastErrMsg = errMsg
			}
		}
	}
}

// ackErrorLogInterval is the minimum interval between logs for repeated final
// acknowledgment errors.
const ackErrorLogInterval = 5 * time.Second

// ackErrorRecoveryThreshold is the number of repeated equal errors after which
// a subsequent successful acknowledgment is logged as a recovery.
const ackErrorRecoveryThreshold = 3

// logAckResult rate-limits repeated equal final acknowledgment errors and logs
// when a later acknowledgment succeeds after repeated failures.
func (m *ackManager) logAckResult(err error) {

	s := &m.ackErrors
	s.Lock()
	defer s.Unlock()

	if err == nil {
		if s.occurrences >= ackErrorRecoveryThreshold {
			slog.Info("NATS message acknowledgment delivery recovered", "previous_error", s.message, "occurrences", s.occurrences)
		}
		s.message = ""
		s.occurrences = 0
		s.lastLoggedAt = time.Time{}
		return
	}

	now := time.Now()
	message := err.Error()
	if s.occurrences == 0 || message != s.message {
		s.message = message
		s.occurrences = 1
		s.lastLoggedAt = now
		slog.Warn("failed to acknowledge NATS message", "error", err, "occurrences", 1)
		return
	}

	s.occurrences++
	if now.Sub(s.lastLoggedAt) >= ackErrorLogInterval {
		s.lastLoggedAt = now
		slog.Warn("failed to acknowledge NATS message", "error", err, "occurrences", s.occurrences)
	}

}

// run sends acknowledgment heartbeats until the context is canceled.
func (m *ackManager) run(ctx context.Context, heartbeatInterval time.Duration) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.inProgress(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// ack coordinates the destination acknowledgments of a NATS message and tracks
// it until every destination has completed or tracking is stopped.
type ack struct {
	mu        sync.Mutex // protects done, remaining, and destinationAck.done
	done      bool       // whether tracking has finished; protected by mu
	msg       jetstream.Msg
	manager   *ackManager
	remaining int // number of destinations that have not acknowledged the message; protected by mu
}

// destinationAck acknowledges one destination of a NATS message.
type destinationAck struct {
	parent *ack
	done   bool // whether the destination has acknowledged the message; protected by parent.mu
}

// Acknowledge marks the destination as complete. The last destination stops
// tracking the message and sends its acknowledgment to NATS.
func (d *destinationAck) Acknowledge() {
	a := d.parent
	a.mu.Lock()
	if d.done || a.done {
		a.mu.Unlock()
		return
	}
	d.done = true
	a.remaining--
	if a.remaining != 0 {
		a.mu.Unlock()
		return
	}
	a.done = true
	a.mu.Unlock()

	a.manager.Remove(a)
	a.manager.logAckResult(a.msg.Ack())
}

// InProgress tells NATS that the message is still being processed.
func (a *ack) InProgress() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.done {
		return nil
	}
	err := a.msg.InProgress()
	if err != nil {
		if errors.Is(err, jetstream.ErrMsgAlreadyAckd) {
			a.done = true
		}
		return err
	}
	return nil
}

// Stop stops tracking the message without acknowledging it.
func (a *ack) Stop() {
	if a.finish() {
		a.manager.Remove(a)
	}
}

// finish marks tracking as finished and reports whether it was finished by
// this call.
func (a *ack) finish() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.done {
		return false
	}
	a.done = true
	return true
}

// destinationsForMessage returns the destinations and their acknowledgments for
// a NATS message. A message without explicit destinations has one destination
// whose ID is empty.
func destinationsForMessage(parent *ack, ids []string) []streams.Destination {
	switch n := len(ids); n {
	case 0:
		return []streams.Destination{{Ack: &destinationAck{parent: parent}}}
	default:
		parent.mu.Lock()
		parent.remaining = n
		parent.mu.Unlock()
		destinations := make([]streams.Destination, n)
		for i := range destinations {
			destinations[i].ID = ids[i]
			destinations[i].Ack = &destinationAck{parent: parent}
		}
		return destinations
	}
}

// isValidDestination reports whether s is a valid destination identifier.
func isValidDestination(s string) bool {
	return len(s) == 12 && base58.IsValid(s)
}

func shardOf(key string) int {
	h := fnv.New32a()
	var buf [20]byte
	n := append(buf[:0], key...)
	_, _ = h.Write(n)
	return int(h.Sum32() % uint32(numShards))
}
