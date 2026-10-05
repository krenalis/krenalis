// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package externalassets resolves bounded assets from ordered HTTP origins.
package externalassets

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/krenalis/krenalis/tools/validation"
)

// Resolver tries origins in order until transport and consumption both succeed.
type Resolver struct {
	origins []string
	client  *http.Client
}

// NewResolver validates and copies origins without changing their order.
func NewResolver(origins []string) (*Resolver, error) {

	if len(origins) == 0 {
		return nil, fmt.Errorf("external asset origins are empty")
	}
	for _, origin := range origins {
		_, err := assetURL(origin, "validation")
		if err != nil {
			return nil, err
		}
	}

	return &Resolver{origins: append([]string(nil), origins...), client: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Resolve rejects assets exceeding maximum decoded bytes from each origin.
// A failed consumer must discard its result; any failure tries the next origin.
// Consumers must not publish results until Resolve has returned successfully.
func (r *Resolver) Resolve(ctx context.Context, path string, maximum int64, consume func(io.Reader) error) error {

	if r == nil || len(r.origins) == 0 || maximum <= 0 || maximum == math.MaxInt64 || consume == nil {
		return fmt.Errorf("invalid external asset resolution arguments")
	}
	var failures []error
	for _, origin := range r.origins {
		address, err := assetURL(origin, path)
		if err != nil {
			return err
		}
		err = r.attempt(ctx, address, maximum, consume)
		if err != nil {
			// Bound diagnostics originating in an untrusted response or consumer.
			message := err.Error()
			if len(message) > 1024 {
				message = message[:1024] + "..."
			}
			failures = append(failures, fmt.Errorf("origin %q: %s", origin, message))
			continue
		}
		return nil
	}

	return fmt.Errorf("load external asset %q: %w", path, errors.Join(failures...))
}

func (r *Resolver) attempt(ctx context.Context, address string, maximum int64, consume func(io.Reader) error) error {

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	// Explicit negotiation disables net/http's implicit gzip decoding.
	request.Header.Set("Accept-Encoding", "gzip, br")
	response, err := r.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", response.StatusCode)
	}

	var decoded io.Reader = response.Body
	encoding := strings.ToLower(strings.TrimSpace(strings.Join(response.Header.Values("Content-Encoding"), ",")))
	switch encoding {
	case "", "identity":
	case "gzip":
		reader, err := gzip.NewReader(response.Body)
		if err != nil {
			return fmt.Errorf("decode gzip: %w", err)
		}
		defer reader.Close()
		decoded = reader
	case "br":
		decoded = brotli.NewReader(response.Body)
	default:
		return fmt.Errorf("unsupported content encoding %.80q", encoding)
	}

	bounded := &io.LimitedReader{R: decoded, N: maximum + 1}
	err = consume(bounded)
	if err != nil {
		return err
	}
	// Complete decoding also detects corrupt trailers and ignored excess bytes.
	_, err = io.Copy(io.Discard, bounded)
	if bounded.N == 0 {
		return fmt.Errorf("decoded asset exceeds %d bytes", maximum)
	}
	if err != nil {
		return fmt.Errorf("decode asset: %w", err)
	}

	return nil
}

func assetURL(base, path string) (string, error) {

	base, err := validation.ParseURL(base, validation.NoQuery)
	if err != nil {
		return "", fmt.Errorf("invalid asset base URL: %w", err)
	}
	if base == "" {
		return "", fmt.Errorf("asset base URL is empty")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if !fs.ValidPath(path) || path == "." || strings.ContainsAny(path, "%?#\\:") {
		return "", fmt.Errorf("asset path must be an unescaped relative path without dot segments")
	}

	return u.JoinPath(path).String(), nil
}
