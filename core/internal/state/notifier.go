// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

const maxIDLen = len("@9223372036854775807")

type notification struct {
	Version int
	Name    string
	Payload string
}

// notificationSession owns the dedicated connection, buffered fragments, and
// active rows. On terminal errors and panics, rows remain owned by the session
// so the caller can inspect the outcome before cleanup and abort the socket
// before closing the rows.
type notificationSession struct {
	conn      *db.Conn
	fragments bytes.Buffer
	rows      *db.Rows
}

// close releases the session's resources. It is safe to call more than once,
// but not concurrently with any other use of the session.
func (session *notificationSession) close() {

	if session.conn == nil {
		return
	}

	if session.rows != nil {
		// pgx Rows.Close drains unread results. Closing the socket first makes
		// abandonment bounded even if the server is blocked producing a row.
		_ = session.conn.Underlying().PgConn().Conn().Close()
		_ = session.rows.Close()
		session.rows = nil
	}

	closeNotificationConnection(session.conn)
	session.conn = nil
	session.fragments.Reset()

}

// reconstruct consumes one transport fragment; (nil, nil) means the message is
// still incomplete. Every final fragment resets the buffer, including on failure.
func (session *notificationSession) reconstruct(ctx context.Context, key *cipher.Key, fragment string) (*notification, error) {

	if strings.HasSuffix(fragment, "*") {
		session.fragments.WriteString(fragment[:len(fragment)-1])
		return nil, nil
	}

	defer session.fragments.Reset()
	p, identifier, _ := strings.Cut(fragment, "@")
	session.fragments.WriteString(p)
	br := base64.NewDecoder(base64.RawStdEncoding, &session.fragments)
	encrypted, err := io.ReadAll(br)
	if err != nil {
		return nil, err
	}

	data, err := key.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}

	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var s strings.Builder
	_, err = io.Copy(&s, zr)
	if err != nil {
		_ = zr.Close()
		return nil, err
	}

	err = zr.Close()
	if err != nil {
		return nil, err
	}

	payload := s.String()
	if identifier != "" {
		payload += "@" + identifier
	}

	version, name, payload, err := parsePayload(payload)
	if err != nil {
		return nil, err
	}

	return &notification{version, name, payload}, nil
}

// notifier produces notifications and establishes listening connections.
type notifier struct {
	db  *db.DB
	key *cipher.Key
}

// newNotifier constructs a notifier without starting notification reception.
func newNotifier(db *db.DB) *notifier {
	return &notifier{db: db}
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
		n := min(len(b), 8000-2)
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
	cleanupDone := conn.Underlying().PgConn().CleanupDone()
	_ = conn.Underlying().Close(ctx)
	_ = conn.Close()
	// Release ends ownership of the pool connection. After that, only the
	// previously captured cleanup barrier may be used. Close may return while
	// pgconn.asyncClose is still in progress.
	<-cleanupDone
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
