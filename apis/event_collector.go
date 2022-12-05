//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"chichi/connector"
)

// maxRequestSize is the maximum size in bytes of an event request body.
const maxRequestSize = 500 * 1024

// Errors returned to and handled by the ServeHTTP method.
var errUnauthorized = errors.New("unauthorized")

// eventDateLayout is the layout used for dates in events.
var eventDateLayout = "2006-01-02T15:04:05.999Z07:00"

// eventCollectorStream represents a source stream connection to send events
// to.
type eventCollectorStream struct {
	ID        int
	Connector string
	Settings  []byte
	Producers []*eventCollectorProducer
}

// eventCollectorProducer represents an event producer. It can be a website,
// mobile or server connection.
type eventCollectorProducer struct {
	ID   int
	Type connector.Type
	Keys []string
}

// An eventCollector collects events and sends them to event streams.
type eventCollector struct {
	sync.RWMutex

	// sources are the allowed sources. If nil, all sources are allowed.
	sources map[int]struct{}

	// route maps a connection key to the stream to send its events to.
	routes map[string]connector.EventStreamConnection

	// defaultStream is the stream to send events to if request key is empty.
	// If nil, the requests with an empty key are denied.
	defaultStream connector.EventStreamConnection
}

// newEventCollector returns a new event collector that sends events to
// streams.
func newEventCollector(ctx context.Context, streams []*eventCollectorStream) (*eventCollector, error) {

	var collector = eventCollector{
		sources: map[int]struct{}{},
		routes:  map[string]connector.EventStreamConnection{},
	}

	for _, s := range streams {
		stream, err := connector.RegisteredEventStream(s.Connector).Connect(ctx, &connector.EventStreamConfig{
			Role:     connector.SourceRole,
			Settings: s.Settings,
		})
		if err != nil {
			return nil, err
		}
		for _, p := range s.Producers {
			if t := p.Type; t == connector.WebsiteType || t == connector.MobileType {
				collector.sources[p.ID] = struct{}{}
			}
			for _, k := range p.Keys {
				collector.routes[k] = stream
			}
		}
	}

	return &collector, nil
}

// ServeHTTP serves event messages from HTTP.
// A message is a JSON stream of JSON objects where the first object is the
// message header.
func (c *eventCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := c.serveHTTP(r)
	if err != nil {
		switch err {
		case errBadRequest:
			http.Error(w, "Bad Request", http.StatusBadRequest)
		case errUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		default:
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// MessageHeader represents the header of an event message.
type MessageHeader struct {
	ReceivedAt string      `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
}

// serveHTTP is called by the ServeHTTP method to serve an event request.
func (c *eventCollector) serveHTTP(r *http.Request) error {

	date := time.Now().UTC()

	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" || len(params) > 1 {
		return errBadRequest
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		return errBadRequest
	}

	// Validate the content length.
	if cl := r.Header.Get("Content-Length"); cl != "" {
		length, _ := strconv.Atoi(cl)
		if length <= 0 || length > maxRequestSize {
			return errBadRequest
		}
	}

	// Authenticate the request.
	src, key, ok := r.BasicAuth()
	if !ok || src == "" {
		return errUnauthorized
	}
	// Validate the source.
	source, _ := strconv.Atoi(src)
	if source <= 0 || source > math.MaxInt32 {
		return errUnauthorized
	}
	c.RLock()
	sources, routes := c.sources, c.routes
	c.RUnlock()
	if sources != nil {
		if _, ok := sources[source]; !ok {
			return errUnauthorized
		}
	}
	// Validate the key.
	var stream connector.EventStreamConnection
	if key == "" {
		stream = c.defaultStream
	} else {
		stream = routes[key]
	}
	if stream == nil {
		return errUnauthorized
	}

	// Prepare the event data.
	var event bytes.Buffer
	enc := json.NewEncoder(&event)
	enc.SetEscapeHTML(false)
	request := MessageHeader{
		ReceivedAt: date.Format(eventDateLayout),
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Proto:      r.Proto,
		URL:        r.URL.String(),
		Headers:    r.Header,
	}
	err = enc.Encode(request)
	if err != nil {
		return err
	}
	body := &io.LimitedReader{R: r.Body, N: maxRequestSize + 1}
	_, err = event.ReadFrom(body)
	if err != nil {
		return err
	}
	if body.N == 0 {
		return errBadRequest
	}

	// Send the event to the stream.
	var ch chan struct{}
	opts := connector.SendOptions{OrderKey: src}
	err = stream.Send(event.Bytes(), opts, func(err error) {
		ch <- struct{}{}
	})
	_ = <-ch

	return err
}
