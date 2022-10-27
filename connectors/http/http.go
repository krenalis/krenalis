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
	"net/http"

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

// Reader returns a ReadCloser from which to read the data.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader() (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.settings.URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		return nil, fmt.Errorf("server responded with status: %s", res.Status)
	}
	return res.Body, nil
}

// ServeUserInterface serves the connector's user interface.
func (c *connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {}

// Write writes the data read from p.
func (c *connection) Write(p io.Reader) error {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.settings.URL, p)
	if err != nil {
		return err
	}
	if c.settings.ContentType != "" {
		req.Header.Set("Content-Type", c.settings.ContentType)
	}
	for name, values := range c.settings.Headers {
		req.Header[name] = values
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("server responded with status: %s", res.Status)
	}
	return nil
}
