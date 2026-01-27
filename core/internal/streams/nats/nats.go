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
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/core/natsopts"
	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/types"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

const numShards = 1

// connection implements the stream.Connection interface.
type connection struct {
	nc *nats.Conn

	mu sync.RWMutex

	// up tracks whether the connection is considered up and the stream exists.
	//
	// It combines:
	//   - an atomic boolean indicating whether the stream exists and the connection is up;
	//   - a wait channel on which goroutines calling WaitUp block while the connection is down;
	//   - a timer used to close the wait channel after a short grace period since the last
	//     transition to the down state.
	up struct {
		atomic.Bool               // true if the stream exists and the connection is up
		wait        chan struct{} // channel used by WaitUp callers
		timer       *time.Timer   // wakes WaitUp callers after a short delay
	}

	js struct {
		jetStream jetstream.JetStream
		stream    jetstream.Stream
		// Channel used by Stream callers to wait for the stream to become available.
		// It is closed when the stream is ready or the connection is closed.
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

// Connect creates and opens a connection to the configured NATS servers.
//
// The connection uses the configured User/Password/Token authentication fields
// when present. If conf.NKey is set, the connection also uses NKey
// authentication with a challenge–response signature based on the provided
// Ed25519 private key.
//
// When NKey authentication is used, the private key is defensively copied and
// kept only inside the authentication callbacks. The copied key material is
// wiped on a best-effort basis when the connection is definitively closed, or
// if the initial connection attempt fails.
func Connect(options natsopts.Options) (conn streams.Connection, err error) {

	c := &connection{}
	c.js.wait = make(chan struct{})

	opts := nats.Options{
		Servers:  options.Servers,
		User:     options.User,
		Password: options.Password,
		Token:    options.Token,

		// With these options enabled (provided at least one valid server URL is configured),
		// the NATS client retries indefinitely, including on ErrNoServers and authentication failures.
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
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		// ConnectedCB may be called before nats.Options.Connect returns,
		// so set nc now because ensureEventStream depends on it.
		c.nc = nc
		// Start ensureEventStream on first connection, unless the connection
		// has already been closed or the goroutine is already running.
		if c.js.cancel == nil {
			ctx, cancel := context.WithCancel(context.Background())
			c.js.cancel = cancel
			go c.ensureEventStream(ctx, streamOpts)
		}
		c.mu.Unlock()
		c.refreshUpState()
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
	opts.DisconnectedErrCB = func(*nats.Conn, error) {
		const msg = "disconnected from NATS server; retrying"
		if err == nil {
			slog.Info(msg)
		} else {
			slog.Info(msg, "error", err)
		}
		c.refreshUpState()
	}

	// ReconnectedCB is invoked whenever a reconnection occurs.
	opts.ReconnectedCB = func(*nats.Conn) {
		slog.Info("reconnected to NATS server")
		c.refreshUpState()
	}

	if options.NKey != nil {

		// Copy the private key bytes so future modifications to conf.NKey by caller won't affect the signer.
		pk := slices.Clone(options.NKey)

		// destroy wipes key material best-effort (in-place overwrite).
		destroy := func() {
			for i := range pk {
				pk[i] = 0
			}
		}
		defer func() {
			if conn == nil {
				destroy()
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
		opts.ClosedCB = func(_ *nats.Conn) { destroy() }

	}

	// Update "up" to create the wait channel.
	c.refreshUpState()

	// Connect to the NATS server.
	nc, err := opts.Connect()
	if err != nil {
		return nil, fmt.Errorf("invalid options provided for NATS initialization: %s", err)
	}
	// After Connect returns, callbacks and stream-creation goroutines may run
	// concurrently, so all access to shared fields must be mutex-protected.
	c.mu.Lock()
	if c.nc == nil {
		c.nc = nc
	}
	c.mu.Unlock()

	return c, nil
}

// Close closes the connection. When Close is called, no other calls to
// connection's methods should be in progress and no other shall be made.
func (c *connection) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	// If the stream is not yet created and goroutines may still be waiting,
	// close the channel to unblock them (do not set it to nil).
	if c.js.stream == nil {
		close(c.js.wait)
		// If ensureEventStream has been called, cancel it if it is still running.
		if c.js.cancel != nil {
			c.js.cancel()
		}
	}
	if c.up.wait != nil {
		close(c.up.wait)
		c.up.timer.Stop()
		c.up.timer = nil
	}
	// Drain and close the NATS connection.
	err := c.nc.Drain()
	c.nc.Close()
	c.mu.Unlock()
	slog.Info("NATS connection closed")
	return err
}

// Stream returns the stream. It waits until the stream has been created.
// It returns an error only if ctx is canceled or if c has been closed.
func (c *connection) Stream(ctx context.Context) (streams.Stream, error) {
	select {
	case <-c.js.wait:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return nil, errors.New("connection has been closed")
	}
	return &stream{c}, nil
}

// WaitUp blocks until the connection is up and the stream is available.
// It returns false if the context is canceled, the connection is closed,
// or the connection remains down for too long.
func (c *connection) WaitUp(ctx context.Context) bool {
	if c.up.Load() {
		return true
	}
	c.mu.RLock()
	wait := c.up.wait
	c.mu.RUnlock()
	if wait == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-wait:
		return c.up.Load()
	}
}

// ensureEventStream initializes the JetStream context and ensures that the
// EVENTS stream exists.
//
// It is run in its own goroutine when the first connection is established.
func (c *connection) ensureEventStream(ctx context.Context, opts streamOptions) {

	js, err := jetstream.New(c.nc)
	if err != nil {
		// jetstream.New can only fail if invalid options are provided;
		// since no options are passed here, this error is unexpected.
		panic(fmt.Sprintf("jetstream.New failed unexpectedly: %v", err))
	}

	cfg := jetstream.StreamConfig{
		Name:        "EVENTS",
		Subjects:    []string{"events.v1.>"},
		Replicas:    opts.replicas,
		Retention:   jetstream.InterestPolicy,
		Storage:     opts.storage,
		Compression: opts.compression,
	}

	bo := backoff.New(10)
	var s jetstream.Stream

	var jetStreamUnavailableLogged bool

	// Create the stream if it does not exist.
	// Exit the loop once the stream is created, already exists,
	// or the context has been canceled.
	for bo.Next(ctx) {
		s, err = js.UpdateStream(ctx, cfg)
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			s, err = js.CreateStream(ctx, cfg)
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
						slog.Warn("cannot verify JetStream availability")
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

	c.mu.Lock()
	// Return immediately if the connection has already been closed.
	if c.closed {
		c.mu.Unlock()
		return
	}
	// Cancel the context to release resources.
	c.js.cancel()
	// Update the JetStream context and stream handle.
	c.js.jetStream = js
	c.js.stream = s
	// Close js.wait to unblock any goroutines waiting for the stream.
	// Do not set it to nil: closing is the signal.
	close(c.js.wait)
	c.mu.Unlock()

	// Update the "up" state now that the stream is available.
	c.refreshUpState()

}

// refreshUpState updates the "up" state based on connection status or stream
// availability. It does nothing if the connection is closed.
func (c *connection) refreshUpState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if c.nc != nil {
		up := c.js.stream != nil && c.nc.IsConnected()
		if !c.up.Bool.CompareAndSwap(!up, up) {
			return
		}
		if up || c.closed {
			if c.up.wait == nil {
				return
			}
			close(c.up.wait)
			c.up.wait = nil
			c.up.timer.Stop()
			c.up.timer = nil
			return
		}
	}
	c.up.wait = make(chan struct{})
	c.up.timer = time.AfterFunc(200*time.Millisecond, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.up.timer == nil {
			return
		}
		close(c.up.wait)
		c.up.wait = nil
		c.up.timer.Stop()
		c.up.timer = nil
	})
}

// stream implements the streams.Stream interface.
type stream struct {
	c *connection
}

// Batch returns a batch publisher for the stream.
func (s *stream) Batch() streams.BatchPublisher {
	return &batch{conn: s.c, futures: make([]jetstream.PubAckFuture, 0, 1)}
}

// Consume returns a buffered channel of the given size that streams events for
// the specified pipeline. Events belonging to the same shard are sent on the
// channel in order, ensuring per-user ordering is preserved.
func (s *stream) Consume(pipeline, size int) streams.Consumer {
	ctx, cancel := context.WithCancel(context.Background())
	consumer := &consumer{
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
		for shard := range numShards {
			consumerName := "EVENTS_" + strconv.Itoa(pipeline) + "_" + strconv.Itoa(shard)
			filterSubject := "events.v1." + strconv.Itoa(pipeline) + "." + strconv.Itoa(shard)
			var cc jetstream.ConsumeContext
			bo := backoff.New(10)
			for bo.Next(ctx) {
				c, err := s.c.js.jetStream.CreateOrUpdateConsumer(ctx, "EVENTS", jetstream.ConsumerConfig{
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
				cc, err = c.Consume(func(msg jetstream.Msg) {
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
	events chan streams.Event
	cancel context.CancelFunc
}

// Close closes the consumer and closes the events channel.
func (c *consumer) Close() {
	c.cancel()
}

// Events returns the channel of events.
func (c *consumer) Events() <-chan streams.Event {
	return c.events
}

// batch implements the streams.Batch interface.
type batch struct {
	conn    *connection
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

// Publish adds an event to the current batch for the given pipelines with the
// provided attributes.
func (batch *batch) Publish(pipelines []int, event map[string]any) error {
	shard := shardOf(event["anonymousId"].(string))
	data, err := types.Marshal(event, schemas.Event)
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		future, err := batch.conn.js.jetStream.PublishMsgAsync(&nats.Msg{
			Subject: "events.v1." + strconv.Itoa(pipeline) + "." + strconv.Itoa(shard),
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
