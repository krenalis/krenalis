// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/json"
)

const maxIDLen = len("@9223372036854775807")

type notification struct {
	Version int
	Name    string
	Payload string
}

// notifier sends and receives state notifications.
type notifier struct {
	db          *db.DB
	ch          chan<- notification
	key         *cipher.Key
	nextVersion int
}

// newNotifier constructs a notifier without starting notification reception.
func newNotifier(db *db.DB, ch chan<- notification) *notifier {
	return &notifier{db: db, ch: ch}
}

// Notify sends the notification n within the transaction tx and returns its
// version. For ElectLeader and SeeLeader notifications, the version is always
// 0.
//
// It can only be called after bootstrap has completed.
func (notifier *notifier) Notify(ctx context.Context, tx *db.Tx, n any) (int, error) {
	t := reflect.TypeOf(n)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	var version int
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
		err = tx.QueryRow(ctx, "INSERT INTO notifications (version, name, payload)\n"+
			"SELECT COALESCE(MAX(version), 0) + 1, $1, $2\n"+
			"FROM notifications\n"+
			"RETURNING version", name, payload).Scan(&version)
		if err != nil {
			return 0, err
		}
	}
	const start = "NOTIFY krenalis, '"
	b := []byte(start)
	b, err := appendEncodeNotification(ctx, b, notifier.key, name, n)
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
	if version > 0 {
		b = append(b, '@')
		b = strconv.AppendInt(b, int64(version), 10)
	}
	b = append(b, '\'')
	_, err = tx.Exec(ctx, string(b))
	if err != nil {
		return 0, err
	}
	return version, nil
}

// connect acquires a dedicated connection and completes LISTEN in autocommit.
// On success, the caller owns the connection and must close and release it
// with closeNotificationConnection.
func (notifier *notifier) connect(ctx context.Context) (*db.Conn, error) {

	conn, err := notifier.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, "LISTEN krenalis")
	if err != nil {
		closeNotificationConnection(conn)
		return nil, err
	}

	return conn, nil
}

// init takes ownership of conn and runs the notification listener,
// reconnecting and replaying missed notifications as needed.
func (notifier *notifier) init(ctx context.Context, conn *db.Conn) {

	defer func() {
		if conn != nil {
			closeNotificationConnection(conn)
		}
	}()

	bo := backoff.New(10)
	bo.SetCap(5 * time.Second)
	var acquireFailed bool
	for bo.Next(ctx) {

		if conn == nil {

			var err error
			conn, err = notifier.connect(ctx)
			if err != nil {
				if ctx.Err() == nil && !acquireFailed {
					slog.Warn("failed to establish notification listener; retrying", "error", err)
					acquireFailed = true
				}
				continue
			}
			if acquireFailed {
				slog.Info("notification listener successfully re-established")
				acquireFailed = false
			}

			err = notifier.replay(ctx, conn)
			if err != nil {
				closeNotificationConnection(conn)
				conn = nil
				if ctx.Err() == nil {
					slog.Error("core/state: cannot query notifications; retrying", "retry_after", bo.WaitTime(), "error", err)
				}
				continue
			}

		}

		err := notifier.listen(ctx, conn)
		if err != nil && ctx.Err() == nil {
			slog.Error("core/state: cannot listen to notifications; retrying", "retry_after", bo.WaitTime(), "error", err)
		}
		closeNotificationConnection(conn)
		conn = nil

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
		if n.Channel != "krenalis" {
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
			encrypted, err := io.ReadAll(br)
			if err != nil {
				continue
			}
			data, err := notifier.key.Decrypt(ctx, encrypted)
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
		version, name, payload, err := parsePayload(payload)
		if err != nil {
			continue
		}
		if version > 0 {
			if version < notifier.nextVersion {
				continue
			}
			if version > notifier.nextVersion {
				return nil
			}
			notifier.nextVersion++
		}
		select {
		case notifier.ch <- notification{version, name, payload}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// replay reads missed notifications and enforces contiguous versions.
func (notifier *notifier) replay(ctx context.Context, conn *db.Conn) error {
	const query = "SELECT version, name, payload FROM notifications WHERE version >= $1 ORDER BY version"
	return conn.QueryScan(ctx, query, notifier.nextVersion, func(rows *db.Rows) error {
		for rows.Next() {

			var n notification
			if err := rows.Scan(&n.Version, &n.Name, &n.Payload); err != nil {
				return err
			}
			if n.Version != notifier.nextVersion {
				panic(fmt.Sprintf("core/state: expected notification version %d, got %d", notifier.nextVersion, n.Version))
			}
			notifier.nextVersion++
			select {
			case notifier.ch <- n:
			case <-ctx.Done():
				return ctx.Err()
			}

		}
		return nil
	})
}

// appendEncodeNotification compresses, encrypts, and Base64-encodes a
// notification.
//
// It first GZIP-compresses the notification name and JSON-encoded data, then
// encrypts the compressed data using AES-GCM. Finally, it Base64-encodes the
// encrypted data, appends it to the provided byte slice, and returns the
// extended slice.
type payloadEncryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
}

func appendEncodeNotification(ctx context.Context, b []byte, encryptor payloadEncryptor, name string, n any) ([]byte, error) {
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
	encryptedData, err := encryptor.Encrypt(ctx, z.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot encrypt notification payload: %s", err)
	}
	return base64.RawStdEncoding.AppendEncode(b, encryptedData), nil
}

// closeNotificationConnection closes the listening session before releasing it.
func closeNotificationConnection(conn *db.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = conn.Underlying().Close(ctx)
	_ = conn.Close()
}

// parsePayload parses a notification payload and returns the version, name,
// and effective payload of the notification. If there is no identifier, it
// returns 0 as version.
func parsePayload(s string) (version int, name, payload string, err error) {
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
		return 0, "", "", errors.New("invalid version")
	}
	v, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil || v < 1 {
		return 0, "", "", errors.New("invalid version")
	}
	return int(v), name, payload, nil
}
