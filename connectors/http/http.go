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
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis"
	"chichi/connector"
)

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
	URL         string
	ContentType string
	Headers     map[string]string
}

// New returns a new HTTP connection.
func New(ctx context.Context, settings []byte, fh connector.Firehose) (connector.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connection")
		}
	}
	c.firehose = fh
	return &c, nil
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
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {

	var s settings

	if event == "save" {
		// Save the settings.
		err := json.Unmarshal(form, &s)
		if err != nil {
			return nil, err
		}
		// Validate URL.
		if n := utf8.RuneCountInString(s.URL); n < 10 || n > 1000 {
			return nil, connector.UIErrorf("URL length must be in range [10,1000]")
		}
		if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
			return nil, connector.UIErrorf("schema of URL must be http or https")
		}
		_, err = url.Parse(s.URL)
		if err != nil {
			return nil, connector.UIErrorf("URL is not a valid URL")
		}
		// Validate ContentType.
		if n := utf8.RuneCountInString(s.ContentType); n == 0 || n > 100 {
			return nil, connector.UIErrorf("content type length must be in range [3,100]")
		}
		// Validate Headers.
		for k, v := range s.Headers {
			if n := utf8.RuneCountInString(k); n == 0 || n > 100 {
				return nil, connector.UIErrorf("header key length must be in range [1,100]")
			}
			if n := utf8.RuneCountInString(v); n == 0 || n > 10000 {
				return nil, connector.UIErrorf("header value length must be in range [1,10000]")
			}
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	}

	if c.settings != nil {
		s = *c.settings
	}

	var headers map[string]any
	for k, v := range s.Headers {
		headers[k] = v
	}

	ui := &connector.SettingsUI{
		Components: []connector.Component{
			&connector.Input{Name: "url", Value: s.URL, Label: "URL", Placeholder: "https://example.com", Type: "url", MinLength: 10, MaxLength: 1000},
			&connector.Input{Name: "contentType", Value: s.ContentType, Label: "Content type", Placeholder: "text/plain", Type: "text", MinLength: 3, MaxLength: 100},
			&connector.KeyValue{Name: "headers", Value: headers, Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &connector.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &connector.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Actions: []connector.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil
}

// Write writes the data read from p.
func (c *connection) Write(p io.Reader) error {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.settings.URL, p)
	if err != nil {
		return err
	}
	if c.settings.ContentType != "" {
		req.Header.Set("Content-Type", c.settings.ContentType)
	}
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
