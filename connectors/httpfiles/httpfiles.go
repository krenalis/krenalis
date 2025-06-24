//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package httpfiles implements the HTTP Files connector.
// (https://datatracker.ietf.org/doc/html/rfc7540)
package httpfiles

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Name:       "HTTP Files",
		Categories: meergo.CategoryFileStorage,
		AsSource: &meergo.AsFileStorageSource{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsFileStorageDestination{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
		Icon: icon,
	}, New)
}

// New returns a new HTTP Files connection.
func New(conf *meergo.FileStorageConfig) (*HTTPFiles, error) {
	c := HTTPFiles{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of HTTP Files connector")
		}
	}
	return &c, nil
}

type HTTPFiles struct {
	conf     *meergo.FileStorageConfig
	settings *innerSettings
}

type innerSettings struct {
	Host    string
	Port    int
	Headers []meergo.KV
}

// AbsolutePath returns the absolute representation of the given path name.
func (h *HTTPFiles) AbsolutePath(ctx context.Context, name string) (string, error) {
	if name[0] != '/' {
		name = "/" + name
	}
	path, query := name, ""
	parsingQuery := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '#' || (!parsingQuery && (c < ' ' || c == 0x7f)) {
			return "", meergo.InvalidPathErrorf("path cannot contains “#“, and control characters")
		}
		if c == '%' && (i+2 < len(name) || !ishex(name[i+1]) || !ishex(name[i+2])) {
			return "", meergo.InvalidPathErrorf("path contains an invalid escape sequence")
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
func (h *HTTPFiles) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	u, err := h.AbsolutePath(ctx, name)
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
func (h *HTTPFiles) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if h.settings == nil {
			s.Port = 443
		} else {
			s = *h.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, h.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Input{Name: "Port", Label: "Port", Placeholder: "443", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&meergo.KeyValue{Name: "Headers", Label: "Headers", KeyLabel: "Key", ValueLabel: "Value",
				KeyComponent:   &meergo.Input{Label: "Key", Placeholder: "Key", Type: "text", MinLength: 1, MaxLength: 100},
				ValueComponent: &meergo.Input{Label: "Value", Placeholder: "Value", Type: "text", MinLength: 1, MaxLength: 10000},
			},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (h *HTTPFiles) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	u, err := h.AbsolutePath(ctx, name)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	for _, header := range h.settings.Headers {
		req.Header[header.Key] = []string{header.Value}
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

// saveSettings saves the settings.
func (h *HTTPFiles) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return meergo.NewInvalidSettingsError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65535 {
		return meergo.NewInvalidSettingsError("port must be in range [1,65535]")
	}
	// Validate Headers.
	for _, header := range s.Headers {
		if n := utf8.RuneCountInString(header.Key); n == 0 || n > 100 {
			return meergo.NewInvalidSettingsError("header key length must be in range [1,100]")
		}
		if n := utf8.RuneCountInString(header.Value); n == 0 || n > 10000 {
			return meergo.NewInvalidSettingsError("header value length must be in range [1,10000]")
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
