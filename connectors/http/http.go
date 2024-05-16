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

	"github.com/open2b/chichi"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the FileStorage and the UIHandler interfaces.
var _ interface {
	chichi.FileStorage
	chichi.UIHandler
} = (*HTTP)(nil)

func init() {
	chichi.RegisterFileStorage(chichi.FileStorageInfo{
		Name: "HTTP",
		Icon: icon,
	}, New)
}

// New returns a new HTTP connection.
func New(conf *chichi.FileStorageConfig) (*HTTP, error) {
	c := HTTP{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP connector")
		}
	}
	return &c, nil
}

type HTTP struct {
	conf     *chichi.FileStorageConfig
	settings *Settings
}

type Settings struct {
	Host    string
	Port    int
	Headers map[string]string
}

// CompletePath returns the complete representation of the given path name.
func (h *HTTP) CompletePath(ctx context.Context, name string) (string, error) {
	if name[0] != '/' {
		name = "/" + name
	}
	path, query := name, ""
	parsingQuery := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '#' || (!parsingQuery && (c < ' ' || c == 0x7f)) {
			return "", chichi.InvalidPathErrorf("path cannot contains “#“, and control characters")
		}
		if c == '%' && (i+2 < len(name) || !ishex(name[i+1]) || !ishex(name[i+2])) {
			return "", chichi.InvalidPathErrorf("path contains an invalid escape sequence")
		}
		if c == '?' && !parsingQuery {
			path, query = name[:i], name[i+1:]
			parsingQuery = true
		}
	}
	host := h.settings.Host
	if h.settings.Port != 443 {
		host = net.JoinHostPort(host, strconv.Itoa(h.settings.Port))
	}
	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     path,
		RawQuery: query,
	}
	return u.String(), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (h *HTTP) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	u, err := h.CompletePath(ctx, name)
	if err != nil {
		return nil, time.Time{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
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
	return res.Body, ts.UTC(), nil
}

// ServeUI serves the connector's user interface.
func (h *HTTP) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if h.settings == nil {
			s.Port = 443
		} else {
			s = *h.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, h.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&chichi.Input{Name: "Port", Label: "Port", Placeholder: "443", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&chichi.KeyValue{Name: "Headers", Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &chichi.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &chichi.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Values: values,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (h *HTTP) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	u, err := h.CompletePath(ctx, name)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	for name, value := range h.settings.Headers {
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

// saveValues saves the user-entered values as settings.
func (h *HTTP) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return chichi.NewInvalidUIValuesError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return chichi.NewInvalidUIValuesError("port must be in range [1,65536]")
	}
	// Validate Headers.
	for k, v := range s.Headers {
		if n := utf8.RuneCountInString(k); n == 0 || n > 100 {
			return chichi.NewInvalidUIValuesError("header key length must be in range [1,100]")
		}
		if n := utf8.RuneCountInString(v); n == 0 || n > 10000 {
			return chichi.NewInvalidUIValuesError("header value length must be in range [1,10000]")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = h.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	h.settings = &s
	return nil
}

func ishex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}
