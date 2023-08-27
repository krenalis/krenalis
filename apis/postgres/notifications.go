//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgres

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxIDLen = len("9223372036854775807")

type Notification struct {
	PID     uint32
	Name    string
	Payload string
	Ack     chan<- struct{}
}

// Notify sends a notification.
func (db *DB) Notify(ctx context.Context, payload any) error {
	return notify(ctx, db, payload)
}

// Notify sends a notification.
func (tx *Tx) Notify(ctx context.Context, payload any) error {
	return notify(ctx, tx, payload)
}

// notify sends a notification on the connection conn.
func notify(ctx context.Context, conn Connection, payload any) error {
	t := reflect.TypeOf(payload)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var b bytes.Buffer
	b.WriteString(t.Name())
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(payload)
	if err != nil {
		return err
	}
	s := b.String()
	s = s[:len(s)-1] // remove new line added by Encode.
	s = escape(s)
	if len(s) > 8000-maxIDLen-2 {
		if db, ok := conn.(*DB); ok {
			// Send within a transaction.
			return db.Transaction(ctx, func(tx *Tx) error {
				return notify(ctx, tx, payload)
			})
		}
		var z strings.Builder
		bw := base64.NewEncoder(base64.RawStdEncoding, &z)
		zw := gzip.NewWriter(bw)
		if _, err = zw.Write(b.Bytes()); err != nil {
			_ = zw.Close()
			_ = bw.Close()
			return err
		}
		if err = zw.Close(); err != nil {
			_ = bw.Close()
			return err
		}
		if err = bw.Close(); err != nil {
			return err
		}
		s = z.String()
		for len(s) > 8000-maxIDLen-2 {
			const k = 8000 - maxIDLen - 3
			_, err = conn.Exec(ctx, "NOTIFY chichi, '+"+s[:k]+"'")
			if err != nil {
				return err
			}
			s = s[k:]
		}
	}
	if tx, ok := conn.(*Tx); ok {
		id, ack := tx.acks.create()
		tx.ack = ack
		s += strconv.Itoa(id)
	}
	_, err = conn.Exec(ctx, "NOTIFY chichi, '"+s+"'")
	return err
}

// ListenToNotifications listens to notifications in its goroutine and sends
// them on the returned channel. Call stop to halt the listening and close the
// channel.
func (db *DB) ListenToNotifications() (notifications <-chan Notification, stop func()) {
	ch := make(chan Notification)
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	stop = func() {
		cancel()
		<-stopped
		close(ch)
	}
	go func() {
		var err error
		var b bytes.Buffer
		var sleep time.Duration
		for {
			if err != nil {
				select {
				case <-ctx.Done():
					close(stopped)
					return
				default:
					log.Printf("[error] %s", err)
				}
			}
			if sleep > 0 {
				time.Sleep(sleep)
				sleep = 0
			}
			b.Reset()
			var conn *Conn
			conn, err = db.Conn(ctx)
			if err != nil {
				sleep = 10 * time.Millisecond
				continue
			}
			_, err = conn.Exec(ctx, "LISTEN chichi")
			if err != nil {
				continue
			}
			err = func() error {
				for {
					n, err := conn.conn.Conn().WaitForNotification(ctx)
					if err != nil {
						return err
					}
					if n.Channel != "chichi" {
						continue
					}
					if len(n.Payload) > 0 && n.Payload[0] == '+' {
						b.WriteString(n.Payload[1:])
						continue
					}
					payload := n.Payload
					if b.Len() > 0 {
						b.WriteString(n.Payload)
						br := base64.NewDecoder(base64.RawStdEncoding, &b)
						zr, err := gzip.NewReader(br)
						if err != nil {
							return err
						}
						var s strings.Builder
						_, err = io.Copy(&s, zr)
						if err != nil {
							_ = zr.Close()
							return err
						}
						if err = zr.Close(); err != nil {
							return err
						}
						payload = s.String()
						b.Reset()
					}
					id, name, payload, err := parsePayload(payload)
					if err != nil {
						return err
					}
					var ack chan<- struct{}
					if id > 0 {
						ack = db.acks.pop(id)
					}
					ch <- Notification{n.PID, name, payload, ack}
				}
			}()
			if err != nil {
				_, _ = conn.Exec(ctx, "UNLISTEN chichi")
				continue
			}
			err = conn.Close(ctx)
		}
	}()
	return ch, stop
}

// parsePayload parses a notification payload and returns the identifier, name,
// and effective payload of the notification. If there is no identifier, it
// returns 0 as identifier.
func parsePayload(s string) (id int, name, payload string, err error) {
	i := strings.IndexByte(s, '{')
	if i == -1 {
		return 0, "", "", errors.New("missing payload")
	}
	if i == 0 {
		return 0, "", "", errors.New("missing name")
	}
	name, s = s[:i], s[i:]
	i = strings.LastIndexByte(s, '}')
	if i == -1 {
		return 0, "", "", errors.New("invalid payload")
	}
	payload, s = s[:i+1], s[i+1:]
	if s == "" {
		return
	}
	id, _ = strconv.Atoi(s)
	if id < 1 {
		return 0, "", "", errors.New("invalid identifier")
	}
	return
}

// acks contains channels used by transactions for which a notification has
// been sent. These channels are used to receive an acknowledgment when the
// notification has been received and processed.
type acks struct {
	sync.Mutex
	ids map[int]chan struct{}
}

// newAcks returns a new empty acks.
func newAcks() *acks {
	return &acks{ids: map[int]chan struct{}{}}
}

// create creates a new ack channel and returns it with its identifier.
func (acks *acks) create() (int, <-chan struct{}) {
	var id int
	var ack chan struct{}
	for ack == nil {
		id = rand.Intn(math.MaxInt-1) + 1
		acks.Lock()
		_, ok := acks.ids[id]
		if !ok {
			ack = make(chan struct{}, 1)
			acks.ids[id] = ack
		}
		acks.Unlock()
	}
	return id, ack
}

// pop removes the ack channel with identifier id and returns it. Returns nil
// if the ack channel does not exist.
func (acks *acks) pop(id int) chan<- struct{} {
	acks.Lock()
	ack, ok := acks.ids[id]
	if ok {
		delete(acks.ids, id)
	}
	acks.Unlock()
	return ack
}
