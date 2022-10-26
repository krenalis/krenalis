//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package http

// This package is the HTTP connector.
// (https://datatracker.ietf.org/doc/html/rfc7540)

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sync"

	"chichi/connectors"
)

// Make sure it implements the StreamConnection interface.
var _ connectors.StreamConnection = &connection{}

func init() {
	connectors.RegisterStreamConnector("HTTP", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
}

type settings struct {
	URL         string
	ContentType string
	Headers     http.Header
}

// New returns a new HTTP connection.
func New(ctx context.Context, settings []byte, fh connectors.Firehose) (connectors.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connection")
		}
	}
	return &c, nil
}

// Reader returns a Reader.
func (c *connection) Reader() (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.settings.URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	res.Body = &reader{res.Body}
	if res.StatusCode != 200 {
		_ = res.Body.Close()
		return nil, fmt.Errorf("server responded with status: %s", res.Status)
	}
	return res.Body, nil
}

// ServeUserInterface serves the connector's user interface.
func (c *connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {}

// Writer returns a Writer.
func (c *connection) Writer() (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.settings.URL, pr)
	if err != nil {
		return nil, err
	}
	if c.settings.ContentType != "" {
		req.Header.Set("Content-Type", c.settings.ContentType)
	}
	for name, values := range c.settings.Headers {
		req.Header[name] = values
	}
	w := &writer{wr: pw, request: req}
	w.response = sync.NewCond(&w.mu)
	return w, nil
}

type reader struct {
	body io.ReadCloser
}

func (r *reader) Close() error {
	_, _ = io.Copy(io.Discard, r.body)
	_ = r.body.Close()
	return nil
}

func (r *reader) Read(p []byte) (int, error) {
	return r.body.Read(p)
}

type writer struct {
	wr       io.WriteCloser
	response *sync.Cond

	mu      sync.Mutex // for the following fields.
	request *http.Request
	closed  bool
	err     error
}

// Close closes the request body and waits for the server response.
func (w *writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	if w.closed {
		return fs.ErrClosed
	}
	w.err = w.wr.Close()
	w.closed = true
	if w.err != nil {
		return w.err
	}
	// Do the request if it has not yet been done.
	if req := w.request; req != nil {
		go w.doRequest(req)
		w.request = nil
	}
	w.response.Wait()
	return w.err
}

// Write writes the request body.
func (w *writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	if w.err != nil {
		w.mu.Unlock()
		return 0, w.err
	}
	// Do the request if it has not yet been done.
	if req := w.request; req != nil {
		go w.doRequest(req)
		w.request = nil
	}
	w.mu.Unlock()
	n, err := w.wr.Write(p)
	w.mu.Lock()
	if w.err == nil {
		w.err = err
	} else {
		if err != nil && err.Error() == "io: read/write on closed pipe" {
			err = errors.New("connection closed prematurely")
		}
		err = w.err
	}
	w.mu.Unlock()
	return n, err
}

// doRequest does the request.
func (w *writer) doRequest(req *http.Request) {
	defer w.response.Signal()
	res, err := http.DefaultTransport.RoundTrip(req)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return
	}
	if err != nil {
		w.err = err
		return
	}
	if res.StatusCode != 200 {
		w.err = fmt.Errorf("server responded with status: %s", res.Status)
		return
	}
	if !w.closed {
		w.err = errors.New("server responded before finishing reading the body")
	}
}
