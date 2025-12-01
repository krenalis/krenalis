// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/json"
)

const maxIDLen = len("@9223372036854775807")

type notification struct {
	ID      int64
	Name    string
	Payload string
}

// notifier sends and receives state notifications.
type notifier struct {
	db     *db.DB
	ch     chan<- notification
	key    []byte
	next   int64
	loaded chan struct{}
	closed struct {
		cancel context.CancelFunc
		atomic.Bool
	}
}

// newNotifier returns a new notifier that will send received notifications to
// the notification channel ch. To begin receiving notifications, call Commit.
func newNotifier(db *db.DB, ch chan<- notification) *notifier {
	notifier := &notifier{
		db:     db,
		ch:     ch,
		loaded: make(chan struct{}, 1),
	}
	ctx, cancel := context.WithCancel(context.Background())
	notifier.closed.cancel = cancel
	go notifier.init(ctx)
	return notifier
}

// Close closes the notifier.
func (notifier *notifier) Close() {
	if notifier.closed.Swap(true) {
		return
	}
	notifier.closed.cancel()
}

// CommitAndStartListening commits the transaction tx, which has read the state,
// then starts listening for state change notifications.
// key is the encryption key that will be set and used in the notifier; it must
// be 32 bytes long.
func (notifier *notifier) CommitAndStartListening(ctx context.Context, tx *db.Tx, key []byte) error {
	// Read the last notification ID.
	var latest int64
	err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(id),0) FROM notifications").Scan(&latest)
	if err != nil {
		return err
	}
	if latest == math.MaxInt64 {
		return errors.New("maximum limit for the auto-increment 'notifications.id' column has been reached")
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	notifier.key = key
	notifier.next = latest + 1
	notifier.loaded <- struct{}{}
	return nil
}

// Notify sends the notification n within the transaction tx and returns its
// identifier. For ElectLeader and SeeLeader notifications, the identifier is
// always 0.
//
// It can only be called after a successful call to Commit.
func (notifier *notifier) Notify(ctx context.Context, tx *db.Tx, n any) (int64, error) {
	t := reflect.TypeOf(n)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	var id int64
	switch name {
	case "ElectLeader", "SeeLeader":
	default:
		payload, err := json.Marshal(n)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, "LOCK TABLE notifications IN EXCLUSIVE MODE")
		if err != nil {
			return 0, err
		}
		err = tx.QueryRow(ctx, "INSERT INTO notifications (id, name, payload)\n"+
			"SELECT COALESCE(MAX(id), 0) + 1, $1, $2\n"+
			"FROM notifications\n"+
			"RETURNING id", name, payload).Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	const start = "NOTIFY meergo, '"
	b := []byte(start)
	b, err := appendEncodeNotification(b, notifier.key, name, n)
	if err != nil {
		return 0, err
	}
	for len(b) > 8000-maxIDLen-2 {
		const n = 8000 - 2
		s := append([]byte(nil), b[:n]...)
		s = append(s, '*', '\'')
		_, err = tx.Exec(ctx, string(s))
		if err != nil {
			return 0, err
		}
		copy(b[len(start):], b[n:])
		b = b[:len(b)-(n-len(start))]
	}
	if id > 0 {
		b = append(b, '@')
		b = strconv.AppendInt(b, int64(id), 10)
	}
	b = append(b, '\'')
	_, err = tx.Exec(ctx, string(b))
	if err != nil {
		return 0, err
	}
	return id, nil
}

// init initializes the notifier to listen for notifications.
func (notifier *notifier) init(ctx context.Context) {
	bo := backoff.New(10)
	bo.SetCap(5 * time.Second)
	for bo.Next(ctx) {
		// Acquire a connection.
		conn, err := notifier.db.Conn(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Error(fmt.Sprintf("core/state: cannot acquire connection, try again after %s", bo.WaitTime()), "err", err)
			}
			continue
		}
		_, err = conn.Exec(ctx, "LISTEN meergo")
		if err != nil {
			// Close and release the connection.
			_ = conn.Underlying().Close(ctx)
			_ = conn.Close()
			if ctx.Err() == nil {
				slog.Error(fmt.Sprintf("core/state: cannot execute LISTEN, try again after %s", bo.WaitTime()), "err", err)
			}
			continue
		}
		if notifier.loaded != nil {
			// Waits for the Commit method to be called.
			select {
			case <-notifier.loaded:
				notifier.loaded = nil
			case <-ctx.Done():
				// Close and release the connection.
				_ = conn.Underlying().Close(ctx)
				_ = conn.Close()
				return
			}
		} else {
			// Reads any missed notifications.
			const query = "SELECT id, name, payload FROM notifications WHERE id >= $1 ORDER BY id"
			err = conn.QueryScan(ctx, query, notifier.next, func(rows *db.Rows) error {
				for rows.Next() {
					var id int64
					var name string
					var payload string
					err = rows.Scan(&id, &name, &payload)
					if err != nil {
						return err
					}
					if id != notifier.next {
						panic(fmt.Sprintf("core/state: expected notification %d, got notification %d", notifier.next, id))
					}
					notifier.next++
					notifier.ch <- notification{id, name, payload}
				}
				return nil
			})
			if err != nil {
				// Close and release the connection.
				_ = conn.Underlying().Close(ctx)
				_ = conn.Close()
				if ctx.Err() == nil {
					slog.Error(fmt.Sprintf("core/state: cannot query notifications, try again after %s", bo.WaitTime()), "err", err)
				}
				continue
			}
		}
		err = notifier.listen(ctx, conn)
		if err != nil && ctx.Err() == nil {
			slog.Error(fmt.Sprintf("core/state: cannot listen to notifications, try again after %s", bo.WaitTime()), "err", err)
		}
		// Close and release the connection.
		_ = conn.Underlying().Close(ctx)
		_ = conn.Close()
	}
}

// listen listens to the notifications received from the connection conn and
// sends them to notification channel.
func (notifier *notifier) listen(ctx context.Context, conn *db.Conn) error {
	var b bytes.Buffer
	for {
		n, err := conn.Underlying().WaitForNotification(ctx)
		if err != nil {
			return err
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
				continue
			}
			data, err := decryptAESGCM(ciphertext, notifier.key)
			if err != nil {
				continue
			}
			zr, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				continue
			}
			var s strings.Builder
			_, err = io.Copy(&s, zr)
			if err != nil {
				_ = zr.Close()
				continue
			}
			if err = zr.Close(); err != nil {
				continue
			}
			payload = s.String()
			b.Reset()
		}
		if identifier != "" {
			payload += "@" + identifier
		}
		id, name, payload, err := parsePayload(payload)
		if err != nil {
			continue
		}
		if id > 0 {
			if id < notifier.next {
				continue
			}
			if id > notifier.next {
				return nil
			}
			notifier.next++
		}
		notifier.ch <- notification{id, name, payload}
	}
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
		return nil, fmt.Errorf("cannot encrypt notification payload: %s", err)
	}
	return base64.RawStdEncoding.AppendEncode(b, encryptedData), nil
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
		return nil, errors.New("cannot decrypt notification payload, ciphertext is too short to contain the nonce")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
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

// parsePayload parses a notification payload and returns the identifier, name,
// and effective payload of the notification. If there is no identifier, it
// returns 0 as identifier.
func parsePayload(s string) (id int64, name, payload string, err error) {
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
	id, _ = strconv.ParseInt(s[1:], 10, 64)
	if id < 1 {
		return 0, "", "", errors.New("invalid identifier")
	}
	return
}
