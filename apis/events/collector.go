//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
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

// eventDateLayout is the layout used for dates in events.
const eventDateLayout = "2006-01-02T15:04:05.999Z"

// Errors handled by the HTTP server of the collector.
var (
	errUnauthorized = errors.New("unauthorized")
	errBadRequest   = errors.New("bad request")
	errNotFound     = errors.New("not found")
)

// A Collector collects events and sends them to streams.
type Collector struct {
	mu      sync.Mutex // for the streams field.
	state   *state.State
	streams map[int]*eventCollectorStream
}

// eventCollectorStream represents a stream used by the event collector.
type eventCollectorStream struct {
	id        int
	workspace int
	stream    connector.StreamConnection
	sending   sync.WaitGroup
}

// NewCollector returns a new event collector. It reads events sent from
// mobile, server and website sources and sends them to the corresponding
// stream. If a source does not have a stream, events will be sent to
// defaultStream.
func NewCollector(ctx context.Context, st *state.State,
	defaultStream connector.StreamConnection) (*Collector, error) {

	var collector = Collector{
		state:   st,
		streams: map[int]*eventCollectorStream{},
	}

	// Open and add the streams.
	streams := map[int]*state.Connection{}
	for _, c := range st.Connections() {
		if s, ok := collector.suitableStreamOf(c); ok {
			streams[s.ID] = s
		}
	}
	for _, s := range streams {
		collector.replaceStream(nil, s)
	}

	// Add the default stream.
	collector.streams[0] = &eventCollectorStream{stream: defaultStream}

	st.AddListener(collector.onAddConnection)
	st.AddListener(collector.onDeleteConnection)
	st.AddListener(collector.onDeleteWorkspace)
	st.AddListener(collector.onSetConnectionSettings)
	st.AddListener(collector.onSetConnectionStatus)
	st.AddListener(collector.onSetConnectionStream)
	st.AddListener(collector.onSetWarehouseSettings)

	return &collector, nil
}

// ServeHTTP serves event messages from HTTP.
// A message is a JSON stream of JSON objects where the first object is the
// message header.
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := c.serveHTTP(r)
	if err != nil {
		switch err {
		case errBadRequest:
			http.Error(w, "Bad Request", http.StatusBadRequest)
		case errNotFound:
			http.Error(w, "Invalid path or identifier", http.StatusNotFound)
		case errUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		default:
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")
	return
}

// MessageHeader represents the header of an event message.
type MessageHeader struct {
	ReceivedAt time.Time   `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
}

// serveHTTP is called by the ServeHTTP method to serve an event request.
func (collector *Collector) serveHTTP(r *http.Request) error {

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

	// Validate the source.
	sourceID, _ := strconv.Atoi(src)
	source, ok := collector.state.Connection(sourceID)
	if !ok || !source.Enabled || source.Role != state.SourceRole {
		return errNotFound
	}
	if typ := source.Connector().Type; typ != state.MobileType && typ != state.WebsiteType {
		return errNotFound
	}

	// Validate the key.
	var server *state.Connection
	if key != "" {
		server, ok = collector.state.ConnectionByKey(key)
		if !ok || !server.Enabled || server.Role != state.SourceRole || server.Connector().Type != state.ServerType {
			return errUnauthorized
		}
		if server.Workspace().ID != source.Workspace().ID {
			return errUnauthorized
		}
	}

	var streamID int
	if server != nil {
		if s, ok := server.Stream(); ok {
			streamID = s.ID
		}
	}
	if streamID == 0 {
		if s, ok := source.Stream(); ok {
			streamID = s.ID
		}
	}
	var stream *eventCollectorStream
	collector.mu.Lock()
	stream, ok = collector.streams[streamID]
	if !ok {
		// Use the default stream.
		stream, ok = collector.streams[0]
	}
	collector.mu.Unlock()
	stream.sending.Add(1)

	// Prepare the event data.
	var event bytes.Buffer
	enc := json.NewEncoder(&event)
	enc.SetEscapeHTML(false)
	request := MessageHeader{
		ReceivedAt: date,
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
	err = stream.stream.Send(event.Bytes(), opts, func(err error) {
		ch <- struct{}{}
	})
	_ = <-ch

	stream.sending.Done()

	return err
}

// onAddConnection is called when a connection is added.
func (collector *Collector) onAddConnection(n state.AddConnectionNotification) {
	c, _ := collector.state.Connection(n.ID)
	if s, ok := collector.suitableStreamOf(c); ok {
		go collector.replaceStream(nil, s)
	}
}

// onDeleteConnection is called when a connection is deleted.
func (collector *Collector) onDeleteConnection(n state.DeleteConnectionNotification) {
	// Check if an open stream has been deleted.
	old, ok := collector.streams[n.ID]
	if ok {
		go collector.replaceStream(old, nil)
	}
	// Check if a connector with an open stream has been deleted.
	keepOpen := map[int]bool{0: true}
	for _, c := range collector.state.Connections() {
		if s, ok := collector.suitableStreamOf(c); ok {
			keepOpen[s.ID] = true
		}
	}
	for _, s := range collector.streams {
		if !keepOpen[s.id] {
			go collector.replaceStream(s, nil)
			break
		}
	}
}

// onDeleteWorkspace is called when a workspace is deleted.
func (collector *Collector) onDeleteWorkspace(n state.DeleteWorkspaceNotification) {
	// Check if one o more open streams have been deleted.
	for _, s := range collector.streams {
		if s.workspace == n.ID {
			go collector.replaceStream(s, nil)
		}
	}
	// Check if one or more connectors with an open stream have been deleted.
	keepOpen := map[int]bool{0: true}
	for _, c := range collector.state.Connections() {
		if s, ok := collector.suitableStreamOf(c); ok {
			keepOpen[s.ID] = true
		}
	}
	for _, s := range collector.streams {
		if !keepOpen[s.id] {
			go collector.replaceStream(s, nil)
		}
	}
}

// onSetConnectionSettings is called when settings of a connection is changed.
func (collector *Collector) onSetConnectionSettings(n state.SetConnectionSettingsNotification) {
	old, ok := collector.streams[n.Connection]
	if ok {
		new, _ := collector.state.Connection(n.Connection)
		go collector.replaceStream(old, new)
	}
}

// onSetConnectionStatus is called when the status of a connection is changed.
func (collector *Collector) onSetConnectionStatus(n state.SetConnectionStatusNotification) {
	c, _ := collector.state.Connection(n.Connection)
	s, ok := collector.suitableStreamOf(c)
	if ok {
		if _, ok := collector.streams[s.ID]; !ok {
			go collector.replaceStream(nil, s)
		}
	} else if s, ok = c.Stream(); ok {
		if s, ok := collector.streams[s.ID]; ok {
			go collector.replaceStream(s, nil)
		}
	}
}

// onSetConnectionStream is called when the stream of a connection is changed.
func (collector *Collector) onSetConnectionStream(n state.SetConnectionStreamNotification) {
	old, ok := collector.streams[n.Connection]
	if ok {
		new, _ := collector.state.Connection(n.Connection)
		go collector.replaceStream(old, new)
	}
}

// onSetWarehouseSettings is called when the settings of a workspace data
// warehouse are changed.
func (collector *Collector) onSetWarehouseSettings(n state.SetWarehouseSettingsNotification) {
	ws, _ := collector.state.Workspace(n.Workspace)
	if n.Settings == nil {
		// Close the streams of the workspace.
		for _, c := range ws.Connections() {
			if s, ok := collector.streams[c.ID]; ok {
				go collector.replaceStream(s, nil)
			}
		}
		return
	}
	// Open the streams of the workspace if they are not already open.
	for _, c := range ws.Connections() {
		s, ok := collector.suitableStreamOf(c)
		if !ok {
			continue
		}
		if _, ok := collector.streams[s.ID]; !ok {
			go collector.replaceStream(nil, c)
		}
	}
}

// replaceStream replaces the stream old with new, opening the new stream and
// closing the old one. If old is nil, it only opens and adds the new stream.
// If new is nil, it only closes and removes the old one.
func (collector *Collector) replaceStream(old *eventCollectorStream, new *state.Connection) {
	// Open to the new stream.
	if new != nil {
		var stream connector.StreamConnection
		for stream == nil {
			var err error
			stream, err = connector.RegisteredStream(new.Connector().Name).Open(
				context.Background(), &connector.StreamConfig{
					Role:     connector.SourceRole,
					Settings: new.Settings,
				})
			if err != nil {
				// Wait and retry.
				log.Printf("[warning] cannot connect to stream %d", new.ID)
				time.Sleep(10 * time.Millisecond)
				collector.mu.Lock()
				if collector.streams[new.ID] != old {
					collector.mu.Unlock()
					return
				}
				collector.mu.Unlock()
			}
		}
		collector.mu.Lock()
		if collector.streams[new.ID] != old {
			if err := stream.Close(); err != nil {
				log.Printf("[warning] an error occurred closing the stream %d: %s", new.ID, err)
			}
			collector.mu.Unlock()
			return
		}
		collector.streams[new.ID] = &eventCollectorStream{
			id:        new.ID,
			stream:    stream,
			workspace: new.Workspace().ID,
		}
		collector.mu.Unlock()
	}
	// Close the old stream.
	if old != nil {
		old.sending.Wait()
		err := old.stream.Close()
		if err != nil {
			log.Printf("[warning] an error occurred closing the stream %d: %s", old.id, err)
		}
	}
}

// suitableStreamOf returns the stream of c only if the collector must send
// events to it.
func (collector *Collector) suitableStreamOf(c *state.Connection) (*state.Connection, bool) {
	if c == nil || !c.Enabled || c.Role != state.SourceRole {
		return nil, false
	}
	s, ok := c.Stream()
	if !ok || !s.Enabled || len(s.Settings) == 0 || s.Workspace().Warehouse == nil {
		return nil, false
	}
	return s, true
}
