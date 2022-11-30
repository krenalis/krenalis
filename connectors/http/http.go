//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package http

// This package is the HTTP connector.
// (https://datatracker.ietf.org/doc/html/rfc7540)

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon []byte

// Make sure it implements the StorageConnection interface.
var _ connector.StorageConnection = &connection{}

func init() {
	connector.RegisterStorage("HTTP", newConnection)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Host    string
	Port    int
	Headers map[string]string
}

// newConnection returns a new HTTP connection.
func newConnection(ctx context.Context, conf *connector.StorageConfig) (connector.StorageConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connection")
		}
	}
	return &c, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "HTTP",
		Type: connector.StorageType,
		Icon: icon,
	}
}

// Reader returns a ReadCloser from which to read the file with the given path
// and its last update time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader(path string) (io.ReadCloser, time.Time, error) {
	u, err := c.requestURL(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	req, err := http.NewRequestWithContext(c.ctx, "GET", u, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	if res.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		return nil, time.Time{}, fmt.Errorf("server responded with status: %s", res.Status)
	}
	ts, _ := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
	return res.Body, ts, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	var s settings
	var headers map[string]any

	switch event {
	case "load":
		// Load the Form.
		if c.settings != nil {
			s = *c.settings
			for k, v := range s.Headers {
				headers[k] = v
			}
		}
	case "save":
		// Save the settings.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, err
		}
		// Validate Host.
		if n := len(s.Host); n == 0 || n > 253 {
			return nil, nil, ui.Errorf("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Port < 1 || s.Port > 65536 {
			return nil, nil, ui.Errorf("port must be in range [1,65536]")
		}
		// Validate Headers.
		for k, v := range s.Headers {
			if n := utf8.RuneCountInString(k); n == 0 || n > 100 {
				return nil, nil, ui.Errorf("header key length must be in range [1,100]")
			}
			if n := utf8.RuneCountInString(v); n == 0 || n > 10000 {
				return nil, nil, ui.Errorf("header value length must be in range [1,10000]")
			}
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.firehose.SetSettings(b)
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Value: s.Host, Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Value: s.Port, Label: "Port", Placeholder: "443", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.KeyValue{Name: "headers", Value: headers, Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &ui.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &ui.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Write writes the data read from p into the file with the given path.
// contentType is the file's content type.
func (c *connection) Write(r io.Reader, path, contentType string) error {
	u, err := c.requestURL(path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(c.ctx, "POST", u, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	for name, value := range c.settings.Headers {
		req.Header[name] = []string{value}
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if c := res.StatusCode; c != 200 && c != 201 {
		return fmt.Errorf("server responded with status: %s", res.Status)
	}
	return nil
}

// requestURL returns a request URL given the path.
func (c *connection) requestURL(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	p, err := url.Parse(path)
	if err != nil || p.Scheme != "" || p.Host != "" {
		return "", fmt.Errorf("path is not an URL path: %s", err)
	}
	u := url.URL{
		Scheme:   "https",
		Host:     net.JoinHostPort(c.settings.Host, strconv.Itoa(c.settings.Port)),
		Path:     p.Path,
		RawQuery: p.RawQuery,
	}
	return u.String(), nil
}
