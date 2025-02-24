//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	mathrand "math/rand/v2"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/json"
)

const maxIDLen = len("@9223372036854775807")

type notification struct {
	PID     uint32
	Name    string
	Payload string
	Ack     chan<- struct{}
}

type Tx struct {
	*db.Tx
	notifyKey []byte
	acks      *acks
	ack       <-chan struct{}
}

// Notify sends a notification on the transaction.
func (tx *Tx) Notify(ctx context.Context, n any) error {
	t := reflect.TypeOf(n)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	const start = "NOTIFY meergo, '"
	name := t.Name()
	b := []byte(start)
	b, err := appendEncodeNotification(b, tx.notifyKey, name, n)
	if err != nil {
		return err
	}
	for len(b) > 8000-maxIDLen-2 {
		const n = 8000 - 2
		s := append([]byte(nil), b[:n]...)
		s = append(s, '*', '\'')
		_, err = tx.Exec(ctx, string(s))
		if err != nil {
			return err
		}
		copy(b[len(start):], b[n:])
		b = b[:len(b)-(n-len(start))]
	}
	if name != "SeeLeader" && name != "LoadState" {
		var id int
		id, tx.ack = tx.acks.create()
		b = append(b, '@')
		b = strconv.AppendInt(b, int64(id), 10)
	}
	b = append(b, '\'')
	_, err = tx.Exec(ctx, string(b))
	return err
}

// Transaction executes f in a transaction.
func (state *State) Transaction(ctx context.Context, f func(tx *Tx) error) error {
	tx := &Tx{
		notifyKey: state.encryptionKey[:32],
		acks:      state.notifications.acks,
	}
	var err error
	tx.Tx, err = state.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := recover(); err != nil {
			_ = tx.Tx.Rollback(ctx)
			panic(err)
		}
	}()
	err = f(tx)
	if err != nil {
		_ = tx.Tx.Rollback(ctx)
		return err
	}
	err = tx.Tx.Commit(ctx)
	if err != nil {
		return err
	}
	if tx.ack != nil {
		<-tx.ack
	}
	return nil
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
	if s[0] != '@' {
		return 0, "", "", errors.New("invalid identifier")
	}
	id, _ = strconv.Atoi(s[1:])
	if id < 1 {
		return 0, "", "", errors.New("invalid identifier")
	}
	return
}

// ListenToNotifications listens to notifications in its goroutine and sends
// them on the returned channel. Call stop to halt the listening and close the
// channel.
func (state *State) listenToNotifications() (notifications <-chan notification, stop func()) {
	ch := make(chan notification)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	stopped := make(chan struct{})
	stop = func() {
		cancel()
		<-stopped
		close(ch)
	}
	var bo *backoff.Backoff
	go func() {
		var err error
		var b bytes.Buffer
		var sleep time.Duration
		for {
			select {
			case <-ctx.Done():
				close(stopped)
				return
			default:
			}
			if err != nil {
				if bo == nil {
					bo = backoff.New(10)
					bo.SetCap(5 * time.Second)
				}
				slog.Error(fmt.Sprintf("error occurred listening notifications, try again after %s", bo.WaitTime()), "err", err)
				err = nil
				bo.Next(ctx)
				continue
			}
			if sleep > 0 {
				time.Sleep(sleep)
				sleep = 0
			}
			b.Reset()
			conn, err := state.db.Conn(ctx)
			if err != nil {
				continue
			}
			_, err = conn.Exec(ctx, "LISTEN meergo")
			if err != nil {
				continue
			}
			if started != nil {
				started <- struct{}{}
			}
			err = func() error {
				for {
					n, err := conn.Underlying().WaitForNotification(ctx)
					if err != nil {
						return err
					}
					if bo != nil {
						bo = nil
					}
					if n.Channel != "meergo" {
						continue
					}
					if strings.HasSuffix(n.Payload, "*") {
						b.WriteString(n.Payload[:len(n.Payload)-1])
						continue
					}
					p, identifier, _ := strings.Cut(n.Payload, "@")
					b.WriteString(p)
					var payload string
					if b.Len() > 0 {
						br := base64.NewDecoder(base64.RawStdEncoding, &b)
						ciphertext, err := io.ReadAll(br)
						if err != nil {
							return err
						}
						data, err := decryptAESGCM(ciphertext, state.encryptionKey[:32])
						if err != nil {
							return err
						}
						zr, err := gzip.NewReader(bytes.NewReader(data))
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
					if identifier != "" {
						payload += "@" + identifier
					}
					id, name, payload, err := parsePayload(payload)
					if err != nil {
						return err
					}
					var ack chan<- struct{}
					if id > 0 {
						ack = state.notifications.acks.pop(id)
					}
					ch <- notification{n.PID, name, payload, ack}
				}
			}()
			if err != nil {
				_, _ = conn.Exec(ctx, "UNLISTEN meergo")
				continue
			}
			conn.Close()
		}
	}()
	<-started
	started = nil
	return ch, stop
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
		id = mathrand.IntN(math.MaxInt-1) + 1
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

// appendEncodeNotification compresses, encrypts, and Base64-encodes a
// notification.
//
// It first GZIP-compresses the notification name and JSON-encoded data, then
// encrypts the compressed data using AES-GCM. Finally, it Base64-encodes the
// encrypted data, appends it to the provided byte slice, and returns the
// extended slice.
func appendEncodeNotification(b, key []byte, name string, n any) ([]byte, error) {
	var z bytes.Buffer
	zw := gzip.NewWriter(&z)
	defer zw.Close()
	_, err := io.WriteString(zw, name)
	if err != nil {
		return nil, err
	}
	err = json.Encode(zw, n)
	if err != nil {
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	encryptedData, err := encryptAESGCM(z.Bytes(), key)
	if err != nil {
		return nil, err
	}
	return base64.RawStdEncoding.AppendEncode(b, encryptedData), nil
}

// encryptAESGCM encrypts the given data using AES-GCM with the provided key.
func encryptAESGCM(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Generates a random nonce and prepends it to the ciphertext for decryption.
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptAESGCM decrypts the given ciphertext using AES-GCM with the provided
// key.
func decryptAESGCM(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// Extracts the nonce from the beginning of the ciphertext before performing decryption.
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext is too short to contain the nonce")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
