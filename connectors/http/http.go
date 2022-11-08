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
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis"
	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon []byte

// Make sure it implements the StreamConnection interface.
var _ connector.StreamConnection = &connection{}

func init() {
	apis.RegisterStreamConnector("HTTP", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	URL     string
	Headers map[string]string
}

// New returns a new HTTP connection.
func New(ctx context.Context, conf *connector.StreamConfig) (connector.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connection")
		}
	}
	c.firehose = conf.Firehose
	return &c, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "HTTP",
		Type: connector.TypeStream,
		Icon: icon,
	}
}

// Reader returns a ReadCloser from which to read the data and its last update
// time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader() (io.ReadCloser, time.Time, error) {
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.settings.URL, nil)
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
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {

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
			return nil, err
		}
		// Validate URL.
		if n := utf8.RuneCountInString(s.URL); n < 10 || n > 1000 {
			return nil, ui.Errorf("URL length must be in range [10,1000]")
		}
		if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
			return nil, ui.Errorf("schema of URL must be http or https")
		}
		_, err = url.Parse(s.URL)
		if err != nil {
			return nil, ui.Errorf("URL is not a valid URL")
		}
		// Validate Headers.
		for k, v := range s.Headers {
			if n := utf8.RuneCountInString(k); n == 0 || n > 100 {
				return nil, ui.Errorf("header key length must be in range [1,100]")
			}
			if n := utf8.RuneCountInString(v); n == 0 || n > 10000 {
				return nil, ui.Errorf("header value length must be in range [1,10000]")
			}
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "url", Value: s.URL, Label: "URL", Placeholder: "https://example.com", Type: "url", MinLength: 10, MaxLength: 1000},
			&ui.KeyValue{Name: "headers", Value: headers, Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &ui.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &ui.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil
}

// Write writes the data read from p. contentType is the data's content type.
func (c *connection) Write(p io.Reader, contentType string) error {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.settings.URL, p)
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
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("server responded with status: %s", res.Status)
	}
	return nil
}
