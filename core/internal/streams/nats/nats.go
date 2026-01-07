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

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/streams"
	. "github.com/meergo/meergo/core/streams/nats"
	"github.com/meergo/meergo/tools/types"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

const numShards = 1

type Connection struct {
	js jetstream.JetStream
}

func (conn *Connection) Close() {
	nc := conn.js.Conn()
	_ = nc.Drain()
	nc.Close()
	return
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
func Connect(conf ConnectionOptions) (conn *Connection, err error) {

	defer func() {
		if errors.Is(err, nats.ErrAuthorization) {
			err = errors.New("invalid credentials or access not authorized")
		}
	}()

	opts := nats.Options{
		Servers:  conf.Servers,
		User:     conf.User,
		Password: conf.Password,
		Token:    conf.Token,

		AllowReconnect:     true,
		MaxReconnect:       -1, // reconnects indefinitely
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

	if conf.NKey == nil {
		nc, err := opts.Connect()
		if err != nil {
			return nil, err
		}
		js, err := jetstream.New(nc)
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("cannot init NATS JetStream: %s", err)
		}
		return &Connection{js: js}, nil
	}

	// Copy the private key bytes so future modifications to conf.NKey by caller won't affect the signer.
	pk := slices.Clone(conf.NKey)

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
			return nil, fmt.Errorf("nonce cannot be empty")
		}
		return ed25519.Sign(pk, nonce), nil
	}
	opts.ClosedCB = func(_ *nats.Conn) { destroy() }

	nc, err := opts.Connect()
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("cannot init NATS JetStream: %s", err)
	}

	return &Connection{js: js}, nil
}

type Stream struct {
	c *Connection
	s jetstream.Stream
}

func (conn *Connection) Stream(ctx context.Context, opts StreamOptions) (streams.Stream, error) {
	s, err := conn.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "EVENTS",
		Subjects:    []string{"events.v1.>"},
		Replicas:    opts.Replicas,
		Retention:   jetstream.InterestPolicy,
		Storage:     opts.Storage,
		Compression: opts.Compression,
	})
	if err != nil {
		return nil, err
	}
	return &Stream{conn, s}, nil
}

type Batch struct {
	conn *Connection
	acks []jetstream.PubAckFuture
}

func (stream *Stream) NewBatch() streams.Batch {
	return &Batch{conn: stream.c, acks: make([]jetstream.PubAckFuture, 0, 1)}
}

func (batch *Batch) Publish(pipelines []int, event map[string]any) error {
	shard := shardOf(event["connectionId"].(int), event["anonymousId"].(string))
	data, err := types.Marshal(event, schemas.Event)
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		ack, err := batch.conn.js.PublishMsgAsync(&nats.Msg{
			Subject: "events.v1." + strconv.Itoa(pipeline) + "." + strconv.Itoa(shard),
			Data:    data,
		})
		if err != nil {
			return err
		}
		batch.acks = append(batch.acks, ack)
	}
	return nil
}

func (batch *Batch) Done(ctx context.Context) error {
	select {
	case <-batch.conn.js.PublishAsyncComplete():
	case <-ctx.Done():
		return ctx.Err()
	}
	// TODO(marco): ack.Err() creates a new channel for every call. Use jetstream.WithPublishAsyncErrHandler instead.
	for _, ack := range batch.acks {
		select {
		case err := <-ack.Err():
			return err
		default:
		}
	}
	return nil
}

// Consume returns a buffered channel of the given size that streams events for
// the specified pipeline. Events belonging to the same shard are sent on the
// channel in order, ensuring per-user ordering is preserved.
func (stream *Stream) Consume(ctx context.Context, pipeline, size int) (<-chan streams.Event, error) {
	done := ctx.Done()
	ch := make(chan streams.Event, size)
	ccs := make([]jetstream.ConsumeContext, 0, numShards)
	for shard := range numShards {
		consumerName := "EVENTS_" + strconv.Itoa(pipeline) + "_" + strconv.Itoa(shard)
		c, err := stream.s.Consumer(ctx, consumerName)
		if err == jetstream.ErrConsumerNotFound {
			c, err = stream.c.js.CreateOrUpdateConsumer(ctx, "EVENTS", jetstream.ConsumerConfig{
				Name:          consumerName,
				Durable:       consumerName,
				FilterSubject: "events.v1." + strconv.Itoa(pipeline) + "." + strconv.Itoa(shard),
				AckPolicy:     jetstream.AckExplicitPolicy,
			})
		}
		if err != nil {
			return nil, err
		}
		cc, err := c.Consume(func(msg jetstream.Msg) {
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
			case ch <- event:
			case <-done:
				return
			}
		})
		if err != nil {
			return nil, err
		}
		ccs = append(ccs, cc)
	}
	go func() {
		<-ctx.Done()
		for _, cc := range ccs {
			cc.Stop()
		}
		close(ch)
	}()
	return ch, nil
}

func (stream *Stream) Close() error {
	return nil
}

func shardOf(connectionID int, anonymousID string) int {
	h := fnv.New32a()
	var buf [20]byte
	n := strconv.AppendInt(buf[:0], int64(connectionID), 10)
	n = append(n, '|')
	n = append(n, anonymousID...)
	_, _ = h.Write(n)
	return int(h.Sum32() % uint32(numShards))
}
