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
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

type Notification struct {
	PID     uint32
	Name    string
	Payload string
}

// Notify sends a notification.
func (tx *Tx) Notify(payload any) error {
	t := reflect.TypeOf(payload)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := strings.TrimSuffix(t.Name(), "Notification")
	var b bytes.Buffer
	b.WriteString(name)
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err := enc.Encode(payload)
	if err != nil {
		return err
	}
	s := b.String()
	s = s[:len(s)-1] // remove new line added by Encode.
	s = escape(s)
	if len(s) > 8000-2 {
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
		for len(s) > 8000-2 {
			const k = 8000 - 3
			_, err = tx.Exec("NOTIFY chichi, '+" + s[:k] + "'")
			if err != nil {
				return err
			}
			s = s[k:]
		}
	}
	_, err = tx.Exec("NOTIFY chichi, '" + s + "'")
	return err
}

// ListenToNotifications listens to notifications in its goroutine and sends
// them on the returned channel.
func (db *DB) ListenToNotifications(ctx context.Context) <-chan Notification {
	ch := make(chan Notification)
	go func() {
		var err error
		var b bytes.Buffer
		for {
			b.Reset()
			if err != nil {
				log.Printf("[error] %s", err)
			}
			var conn *Conn
			conn, err = db.Conn(ctx)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			_, err = conn.Exec("LISTEN chichi")
			if err != nil {
				_ = conn.Close()
				continue
			}
			err = conn.conn.Raw(func(c any) error {
				cc := c.(*stdlib.Conn).Conn()
				for {
					n, err := cc.WaitForNotification(ctx)
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
					i := strings.IndexByte(payload, '{')
					if i == -1 {
						i = len(payload)
					}
					ch <- Notification{n.PID, payload[:i], payload[i:]}
				}
			})
			if err != nil {
				_, _ = conn.Exec("UNLISTEN chichi")
				_ = conn.Close()
				continue
			}
			err = conn.Close()
		}
	}()
	return ch
}
