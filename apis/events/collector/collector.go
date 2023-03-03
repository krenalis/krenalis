//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package collector

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
	"time"

	"chichi/apis/state"
	"chichi/connector"

	"golang.org/x/text/unicode/norm"
)

// maxRequestSize is the maximum size in bytes of an event request body.
const maxRequestSize = 500 * 1024

// Errors handled by the HTTP server of the collector.
var (
	errUnauthorized = errors.New("unauthorized")
	errBadRequest   = errors.New("bad request")
	errNotFound     = errors.New("not found")
)

// A Collector collects events and sends them to streams.
type Collector struct {
	state  *state.State
	stream connector.StreamConnection
}

// New returns a new event collector. It reads events sent from mobile, server
// and website sources and sends them to stream.
func New(ctx context.Context, st *state.State, stream connector.StreamConnection) (*Collector, error) {
	var collector = Collector{
		state:  st,
		stream: stream,
	}
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

	// Prepare the event data.
	var payload bytes.Buffer
	enc := json.NewEncoder(&payload)
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

	// Read the request body and append it to the payload.
	body, err := readRequestBody(r.Body)
	if err != nil {
		return err
	}
	payload.Write(body)

	// Send the event to the stream.
	ch := make(chan struct{})
	opts := connector.SendOptions{OrderKey: src}
	err = collector.stream.Send(payload.Bytes(), opts, func(err error) {
		ch <- struct{}{}
	})
	_ = <-ch

	return err
}

// readRequestBody reads a request body and returns it in a canonical
// representation. The request body cannot be longer than maxRequestSize bytes
// and must be a JSON object, otherwise it returns the errBadRequest error.
func readRequestBody(r io.Reader) ([]byte, error) {
	// Read r and check that it is not longer than maxRequestSize bytes.
	lr := &io.LimitedReader{R: r, N: maxRequestSize + 1}
	var b bytes.Buffer
	_, err := b.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if lr.N == 0 {
		return nil, errBadRequest
	}
	// Decode as a JSON object.
	dec := json.NewDecoder(&b)
	dec.UseNumber()
	var payload map[string]any
	err = dec.Decode(&payload)
	if err != nil {
		return nil, errBadRequest
	}
	// Check that the body contains at most only whitespace characters.
	for {
		c, err := b.ReadByte()
		if err == io.EOF {
			break
		}
		switch c {
		case ' ', '\n', '\r', '\t':
		default:
			return nil, errBadRequest
		}
	}
	b.Reset()
	// Encode it in a canonical representation.
	w := norm.NFC.Writer(&b)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "")
	enc.SetEscapeHTML(false)
	err = enc.Encode(payload)
	if err != nil {
		return nil, errBadRequest
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	p := b.Bytes()
	return p[:len(p)-1], nil
}
