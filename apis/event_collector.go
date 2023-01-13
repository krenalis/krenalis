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

	"chichi/apis/state"
	"chichi/connector"
)

// maxRequestSize is the maximum size in bytes of an event request body.
const maxRequestSize = 500 * 1024

// Errors returned to and handled by the ServeHTTP method.
var errUnauthorized = errors.New("unauthorized")

// eventDateLayout is the layout used for dates in events.
var eventDateLayout = "2006-01-02T15:04:05.999Z07:00"

// An eventCollector collects events and sends them to event streams.
type eventCollector struct {
	sync.RWMutex

	sources map[int]*state.Connection

	keys map[string]*state.Connection

	// defaultStream is the stream to send events to if request sources don't
	// have a stream.
	defaultStream connector.EventStreamConnection

	streamConnections map[int]connector.EventStreamConnection
}

// newEventCollector returns a new event collector. Reads events sent from
// mobile, server and website sources in connections and sends them to the
// source streams in connections. defaultStream is the stream to send events if
// a source does not have a stream.
func newEventCollector(ctx context.Context, connections []*state.Connection,
	defaultStream connector.EventStreamConnection) (*eventCollector, error) {

	var collector = eventCollector{
		sources:           map[int]*state.Connection{},
		keys:              map[string]*state.Connection{},
		defaultStream:     defaultStream,
		streamConnections: map[int]connector.EventStreamConnection{},
	}

	for _, c := range connections {
		if !c.Enabled || c.Role != state.SourceRole {
			continue
		}
		switch conn := c.Connector(); conn.Type {
		case state.EventStreamType:
			if len(c.Settings) == 0 {
				continue
			}
			stream, err := connector.RegisteredEventStream(conn.Name).Connect(ctx, &connector.EventStreamConfig{
				Role:     connector.SourceRole,
				Settings: c.Settings,
			})
			if err != nil {
				return nil, err
			}
			collector.streamConnections[c.ID] = stream
		case state.MobileType, state.WebsiteType:
			collector.sources[c.ID] = c
		case state.ServerType:
			for _, key := range c.Keys {
				collector.keys[key] = c
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
		if length < 1 || length > maxRequestSize {
			return errBadRequest
		}
	}

	// Authenticate the request.
	src, key, ok := r.BasicAuth()
	if !ok || src == "" {
		return errUnauthorized
	}
	// Validate the key.
	var server *state.Connection
	if key != "" {
		server = c.keys[key]
		if server == nil {
			return errUnauthorized
		}
	}

	// Validate the source.
	sourceID, _ := strconv.Atoi(src)
	if sourceID < 1 || sourceID > math.MaxInt32 {
		return errBadRequest
	}
	source, ok := c.sources[sourceID]
	if !ok {
		return errNotFound
	}

	var streamID int
	var streamConnection connector.EventStreamConnection
	if server != nil && server.Stream() != nil {
		streamID = server.Stream().ID
	} else if s := source.Stream(); s != nil {
		streamID = s.ID
	}
	if streamID == 0 {
		streamConnection = c.defaultStream
	} else {
		streamConnection = c.streamConnections[streamID]
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
	ch := make(chan struct{})
	opts := connector.SendOptions{OrderKey: src}
	err = streamConnection.Send(event.Bytes(), opts, func(err error) {
		ch <- struct{}{}
	})
	_ = <-ch

	return err
}
