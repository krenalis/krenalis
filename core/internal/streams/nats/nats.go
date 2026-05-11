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
	"github.com/krenalis/krenalis/tools/types"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

const numShards = 1

// stream implements streams.Stream.
type stream struct {
	nc *nats.Conn

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

	s := &stream{}
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

	var jetStreamUnavailableLogged bool

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
				if !jetStreamUnavailableLogged {
					slog.Warn("JetStream not enabled for this account; waiting for availability")
					jetStreamUnavailableLogged = true
				}
				continue
			case
				errors.Is(err, jetstream.ErrJetStreamNotEnabled),
				errors.Is(err, nats.ErrNoResponders):
				if !jetStreamUnavailableLogged {
					slog.Warn("JetStream not enabled; waiting for availability")
					jetStreamUnavailableLogged = true
				}
				continue
			default:
				if ctx.Err() == nil {
					if !jetStreamUnavailableLogged {
						slog.Warn("cannot update or create stream", "err", err)
						jetStreamUnavailableLogged = true
					}
					continue
				}
			}
		}
		if jetStreamUnavailableLogged {
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
	ctx, cancel := context.WithCancel(context.Background())
	consumer := &consumer{
		stream: s,
		events: make(chan streams.Event, size),
		cancel: cancel,
	}
	done := ctx.Done()
	go func() {
		ccs := make([]jetstream.ConsumeContext, 0, numShards)
		defer func() {
			// Stop the consumers.
			for _, cc := range ccs {
				cc.Stop()
			}
			// The channel is closed only after the consumers have been stopped.
			close(consumer.events)
		}()
		err := s.waitStream(ctx)
		if err != nil {
			return
		}
		for shard := range numShards {
			consumerName := "EVENTS_" + topic + "_" + strconv.Itoa(shard)
			filterSubject := "events.v1." + topic + "." + strconv.Itoa(shard)
			var cc jetstream.ConsumeContext
			bo := backoff.New(10)
			for bo.Next(ctx) {
				jsConsumer, err := s.js.jetStream.CreateOrUpdateConsumer(ctx, "EVENTS", jetstream.ConsumerConfig{
					Name:          consumerName,
					Durable:       consumerName,
					FilterSubject: filterSubject,
					AckPolicy:     jetstream.AckExplicitPolicy,
					MaxAckPending: -1,
				})
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					slog.Warn("cannot create or update a NATS consumer", "name", consumerName)
					continue
				}
				cc, err = jsConsumer.Consume(func(msg jetstream.Msg) {
					var event streams.Event
					var err error
					defer func() {
						if err != nil {
							err := msg.TermWithReason(err.Error())
							if err != nil {
								slog.Warn(fmt.Sprintf("collector: cannot ack event: %s", err))
							}
						}
					}()
					event.Attributes, err = types.Decode[map[string]any](bytes.NewReader(msg.Data()), schemas.Event)
					if err != nil {
						err = fmt.Errorf("invalid event data: %s", err)
						return
					}
					if header := msg.Headers(); header != nil {
						if destinations, ok := header["destinations"]; ok {
							event.Destinations = make([]int, len(destinations))
							for i, d := range destinations {
								id, _ := strconv.Atoi(d)
								if id <= 0 {
									err = fmt.Errorf("invalid event destination: %q", d)
									return
								}
								event.Destinations[i] = id
							}
						}
					}
					event.Ack = func() {
						if err := msg.Ack(); err != nil {
							slog.Warn(fmt.Sprintf("collector: cannot ack event: %s", err))
						}
					}
					select {
					case consumer.events <- event:
					case <-done:
						if err := msg.Nak(); err != nil {
							slog.Warn("cannot send nack for a NATS message", "error", err)
						}
						return
					}
				})
				if err != nil && ctx.Err() == nil {
					if errors.Is(err, jetstream.ErrConsumerDoesNotExist) {
						continue
					}
					slog.Warn("cannot consume messages from a NATS consumer", "consumer", consumerName)
					continue
				}
				break
			}
			if ctx.Err() != nil {
				return
			}
			ccs = append(ccs, cc)
		}
		<-ctx.Done()
	}()
	return consumer
}

// consumer implements the streams.Consumer interface.
type consumer struct {
	stream *stream
	events chan streams.Event
	cancel context.CancelFunc
}

// Close closes the consumer and closes the events channel.
func (c *consumer) Close() {
	c.cancel()
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
func (batch *batch) Publish(ctx context.Context, topics []string, event map[string]any, destinations []int) error {
	shard := shardOf(event["anonymousId"].(string))
	data, err := types.Marshal(event, schemas.Event)
	if err != nil {
		return err
	}
	for _, topic := range topics {
		var header nats.Header
		if strings.HasPrefix(topic, "connection-") {
			h := make([]string, len(destinations))
			for i, d := range destinations {
				h[i] = strconv.Itoa(d)
			}
			header = nats.Header{"destinations": h}
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

func shardOf(key string) int {
	h := fnv.New32a()
	var buf [20]byte
	n := append(buf[:0], key...)
	_, _ = h.Write(n)
	return int(h.Sum32() % uint32(numShards))
}
