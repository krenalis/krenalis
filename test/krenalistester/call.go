// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// StatusCodeError is an error returned by TryCall when the HTTP call returned a
// status code which is not 200.
type StatusCodeError struct {
	Request struct {
		Method  string
		Path    string
		HasBody bool
	}
	Response struct {
		Code         int
		Header       http.Header
		Text         string
		BodyExpected bool
	}
}

func (e *StatusCodeError) Error() string {
	s := &strings.Builder{}
	fmt.Fprintf(s, "%s %s: unexpected status code %d", e.Request.Method, e.Request.Path, e.Response.Code)
	if e.Response.Text != "" {
		fmt.Fprintf(s, ": %s", e.Response.Text)
	}
	fmt.Fprintf(s, " [request has body: %t, response body expected: %t]", e.Request.HasBody, e.Response.BodyExpected)
	return s.String()
}

// TryCall calls the API endpoint serializing the given body and deserializing
// the response into response.
//
// Returns an error if the call returns an error, which may be a
// StatusCodeError error in case of a HTTP request which returned a status code
// which is not 200, or if the HTTP response cannot be decoded into response.
// A 429 response is retried according to its Retry-After header. If headers
// contains the "Krenalis-Workspace" key, TryCall does not add it automatically.
// A nil value suppresses the header.
func (k *Krenalis) TryCall(method, path string, headers http.Header, body, response any) error {
	return k.tryCall(method, path, headers, body, response, true)
}

// TryCallWithoutRetry is like TryCall but returns a 429 response
// without retrying it. It is intended for tests that explicitly verify the
// rate-limit response.
func (k *Krenalis) TryCallWithoutRetry(method, path string, headers http.Header, body, response any) error {
	return k.tryCall(method, path, headers, body, response, false)
}

// Call calls the API endpoint serializing the given body and deserializing the
// response into response. It retries 429 responses according to their
// Retry-After header.
//
// Calls (*testing.T).Fatal if the call returns an error, if the HTTP response
// cannot be decoded into response, or if the HTTP response's status code is not
// 200.
// If headers contains the "Krenalis-Workspace" key, Call does not add it
// automatically. A nil value suppresses the header.
func (k *Krenalis) Call(method, path string, headers http.Header, body, response any) {
	must(k.t, k.tryCall(method, path, headers, body, response, true))
}

func (k *Krenalis) tryCall(method, path string, headers http.Header, body, response any, retry bool) error {

	path = strings.TrimLeft(path, "/")
	url := "http://" + k.Addr() + "/" + path

	var bodyValue []byte
	if body != nil {
		var err error
		bodyValue, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	var resp *http.Response

	for resp == nil {

		var data io.Reader
		if bodyValue != nil {
			data = bytes.NewReader(bodyValue)
		}
		req, err := http.NewRequestWithContext(k.t.Context(), method, url, data)
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		if _, ok := headers["Krenalis-Workspace"]; !ok {
			if id := k.WorkspaceID(); id != "" {
				req.Header.Set("Krenalis-Workspace", id)
			}
		}
		for key, values := range headers {
			req.Header[key] = slices.Clone(values)
		}

		k.t.Logf("[info] %s %s: executing request", method, url)
		resp, err = k.httpClient.Do(req)
		if err != nil {
			return err
		}
		k.t.Logf("[info] %s %s: Krenalis responded with HTTP status %d", method, url, resp.StatusCode)

		if resp.StatusCode == http.StatusTooManyRequests && retry {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			delay, err := parseRetryAfter(resp.Header.Get("Retry-After"))
			if err != nil {
				return err
			}
			k.t.Logf("[info] %s %s: retrying after %s", method, url, delay)
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-k.t.Context().Done():
				timer.Stop()
				return k.t.Context().Err()
			}
			resp = nil
		}

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		text, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var scErr StatusCodeError
		scErr.Request.Method = method
		scErr.Request.Path = path
		scErr.Request.HasBody = body != nil
		scErr.Response.Code = resp.StatusCode
		scErr.Response.Header = resp.Header.Clone()
		scErr.Response.Text = string(bytes.TrimSpace(text))
		scErr.Response.BodyExpected = response != nil
		return &scErr
	}

	if response == nil {
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(response); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	extraneous, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(extraneous)) > 0 {
		return fmt.Errorf("server returned extraneous data in response body: %q", string(extraneous))
	}

	return nil
}

const maxRetryAfter = 20 * time.Minute // Keep in sync with the corresponding constant in the Krenalis rate limiter.

// parseRetryAfter parses the positive integer number of seconds emitted by
// Krenalis. It rejects all other HTTP Retry-After formats.
func parseRetryAfter(value string) (time.Duration, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds < 1 || seconds > int64(maxRetryAfter/time.Second) || strconv.FormatInt(seconds, 10) != value {
		return 0, fmt.Errorf("Retry-After value %q is not valid", value)
	}
	return time.Duration(seconds) * time.Second, nil
}

// must fails the test t if err is not nil, additionally printing the call
// stack.
func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("%s\nTest call stack: %s", err, string(debug.Stack()))
	}
}
