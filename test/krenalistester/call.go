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
	"strings"
	"testing"
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
		Code        int
		Text        string
		HasResponse bool
	}
}

func (e *StatusCodeError) Error() string {
	s := &strings.Builder{}
	fmt.Fprintf(s, "%s %s: unexpected status code %d", e.Request.Method, e.Request.Path, e.Response.Code)
	if e.Response.Text != "" {
		fmt.Fprintf(s, ": %s", e.Response.Text)
	}
	fmt.Fprintf(s, " [has body: %t, has response: %t]", e.Request.HasBody, e.Response.HasResponse)
	return s.String()
}

// TryCall calls the API endpoint serializing the given body and deserializing
// the response into response.
//
// Returns an error if the calls returns an error, which may be a
// StatusCodeError error in case of a HTTP request which returned a status code
// which is not 200, or if the HTTP response cannot be decoded into response. If
// headers contains the "Krenalis-Workspace" key, TryCall does not add it
// automatically. A nil value suppresses the header.
func (k *Krenalis) TryCall(method, path string, headers http.Header, body, response any) error {
	return k.tryCall(method, path, headers, body, response)
}

// Call calls the API endpoint serializing the given body and deserializing the
// response into response.
//
// Calls (*testing.T).Fatal if the call returns an error, if the HTTP response
// cannot be decoded into response, or if the HTTP response's status code is not
// 200.
// If headers contains the "Krenalis-Workspace" key, Call does not add it
// automatically. A nil value suppresses the header.
func (k *Krenalis) Call(method, path string, headers http.Header, body, response any) {
	must(k.t, k.tryCall(method, path, headers, body, response))
}

func (k *Krenalis) tryCall(method, path string, headers http.Header, body any, response any) error {

	path = strings.TrimLeft(path, "/")
	url := "http://" + k.Addr() + "/" + path

	var data io.Reader
	if body != nil {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(body)
		if err != nil {
			return err
		}
		data = &b
	}
	req, err := http.NewRequest(method, url, data)
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
	resp, err := k.httpClient.Do(req)
	if err != nil {
		return err
	}
	k.t.Logf("[info] %s %s: Krenalis responded with HTTP status %d", method, url, resp.StatusCode)
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
		scErr.Response.Text = string(bytes.TrimSpace(text))
		scErr.Response.HasResponse = response != nil
		return &scErr
	}

	if response != nil {
		dec := json.NewDecoder(resp.Body)
		dec.UseNumber()
		err = dec.Decode(&response)
		if err != nil {
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
	}

	return nil
}

// must fails the test t if err is not nil, additionally printing the call
// stack.
func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("%s\nTest call stack: %s", err, string(debug.Stack()))
	}
}
