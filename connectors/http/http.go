//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package http implements the HTTP connector.
// (https://datatracker.ietf.org/doc/html/rfc7540)
package http

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
var icon = "<svg></svg>"

// Make sure it implements the UI and the StorageConnection interfaces.
var _ interface {
	connector.UI
	connector.StorageConnection
} = (*connection)(nil)

func init() {
	connector.RegisterStorage(connector.Storage{
		Name: "HTTP",
		Icon: icon,
	}, open)
}

// open opens an HTTP connection and returns it.
func open(ctx context.Context, conf *connector.StorageConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	conf     *connector.StorageConfig
	settings *settings
}

type settings struct {
	Host    string
	Port    int
	Headers map[string]string
}

// CompletePath returns the complete representation of the given path name or an
// InvalidPathError if name is not valid for use in calls to Open and Write.
func (c *connection) CompletePath(name string) (string, error) {
	if name[0] != '/' {
		return "", connector.InvalidPathErrorf("path must start with a slash")
	}
	path, query := name, ""
	parsingQuery := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '#' || (!parsingQuery && (c < ' ' || c == 0x7f)) {
			return "", connector.InvalidPathErrorf("path cannot contains “#“, and control characters")
		}
		if c == '%' && (i+2 < len(name) || !ishex(name[i+1]) || !ishex(name[i+2])) {
			return "", connector.InvalidPathErrorf("path contains an invalid escape sequence")
		}
		if c == '?' && !parsingQuery {
			path, query = name[:i], name[i+1:]
			parsingQuery = true
		}
	}
	host := c.settings.Host
	if c.settings.Port != 443 {
		host = net.JoinHostPort(host, strconv.Itoa(c.settings.Port))
	}
	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     path,
		RawQuery: query,
	}
	return u.String(), nil
}

// Reader opens the file at the given path name and returns a ReadCloser from
// which to read the file and its last update time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader(name string) (io.ReadCloser, time.Time, error) {
	u, err := c.CompletePath(name)
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

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings == nil {
			s.Port = 443
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.ValidateSettings(values)
		if err != nil {
			return nil, nil, err
		}
		err = c.conf.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Label: "Port", Placeholder: "443", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.KeyValue{Name: "headers", Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &ui.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &ui.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, ui.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, ui.Errorf("port must be in range [1,65536]")
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
	return json.Marshal(&s)
}

// Write writes the data read from r into the file with the given path name.
// contentType is the file's content type.
func (c *connection) Write(r io.Reader, name, contentType string) error {
	u, err := c.CompletePath(name)
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

func ishex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}
