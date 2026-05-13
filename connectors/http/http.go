// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package http provides a connector for HTTP.
// (https://www.ietf.org/rfc/rfc7540.txt)
package http

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterFileStorage(connectors.FileStorageSpec{
		Code:       "http-get",
		Label:      "HTTP GET",
		Categories: connectors.CategoryFileStorage,
		AsSource: &connectors.AsFileStorageSource{
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
	}, New)
	connectors.RegisterFileStorage(connectors.FileStorageSpec{
		Code:       "http-post",
		Label:      "HTTP POST",
		Categories: connectors.CategoryFileStorage,
		AsDestination: &connectors.AsFileStorageDestination{
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connection instance for HTTP GET/HTTP POST requests.
func New(env *connectors.FileStorageEnv) (*HTTP, error) {
	return &HTTP{env: env}, nil
}

type HTTP struct {
	env *connectors.FileStorageEnv
}

type innerSettings struct {
	Host    string          `json:"host"`
	Port    int             `json:"port"`
	Headers []connectors.KV `json:"headers"`
}

// AbsolutePath returns the absolute representation of the given path name.
func (h *HTTP) AbsolutePath(ctx context.Context, name string) (string, error) {
	if name[0] != '/' {
		name = "/" + name
	}
	path, query := name, ""
	parsingQuery := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '#' || (!parsingQuery && (c < ' ' || c == 0x7f)) {
			return "", connectors.InvalidPathErrorf("path cannot contains “#“, and control characters")
		}
		if c == '%' && (i+2 < len(name) || !ishex(name[i+1]) || !ishex(name[i+2])) {
			return "", connectors.InvalidPathErrorf("path contains an invalid escape sequence")
		}
		if c == '?' && !parsingQuery {
			path, query = name[:i], name[i+1:]
			parsingQuery = true
		}
	}
	var s innerSettings
	err := h.env.Settings.Load(ctx, &s)
	if err != nil {
		return "", err
	}
	host := s.Host
	if s.Port != 443 {
		host = net.JoinHostPort(host, strconv.Itoa(s.Port))
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
	u, err := h.AbsolutePath(ctx, name)
	if err != nil {
		return nil, time.Time{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	res, err := transport.RoundTrip(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	if res.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		return nil, time.Time{}, fmt.Errorf("server responded with status: %s", res.Status)
	}
	ts, _ := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
	if ts.IsZero() {
		// For now, let's take the current timestamp. This behavior is a bit
		// implicit at the moment, but when we implement
		// https://github.com/krenalis/krenalis/issues/1777, this behavior will be
		// removed from here.
		ts = time.Now()
	}
	return res.Body, ts.UTC(), nil
}

// ServeUI serves the connector's user interface.
func (h *HTTP) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		err := h.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		if s.Port == 0 {
			s.Port = 443
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, h.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&connectors.Input{Name: "port", Label: "Port", Placeholder: "443", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&connectors.KeyValue{Name: "headers", Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &connectors.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &connectors.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Settings: settings,
		Buttons:  []connectors.Button{connectors.SaveButton},
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (h *HTTP) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	u, err := h.AbsolutePath(ctx, name)
	if err != nil {
		return err
	}
	var s innerSettings
	err = h.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	for _, header := range s.Headers {
		req.Header[header.Key] = []string{header.Value}
	}
	res, err := transport.RoundTrip(req)
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

// saveSettings saves the settings.
func (h *HTTP) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Host.
	if err = validation.ValidateHost(s.Host); err != nil {
		return connectors.NewInvalidSettingsError(err.Error())
	}
	// Validate Port.
	if err := validation.ValidatePort(s.Port); err != nil {
		return connectors.NewInvalidSettingsError(err.Error())
	}
	// Validate Headers.
	for _, header := range s.Headers {
		if n := utf8.RuneCountInString(header.Key); n == 0 || n > 100 {
			return connectors.NewInvalidSettingsError("header key length must be in range [1,100]")
		}
		if n := utf8.RuneCountInString(header.Value); n == 0 || n > 10000 {
			return connectors.NewInvalidSettingsError("header value length must be in range [1,10000]")
		}
	}
	return h.env.Settings.Store(ctx, s)
}

func ishex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

var dialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

var transport http.RoundTripper = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           dialer.DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}
