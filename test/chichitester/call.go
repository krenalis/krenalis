//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
)

// StatusCodeError is an error returned by Call when the HTTP call returned a
// status code which is not 200.
type StatusCodeError struct {
	Code         int
	ResponseText string
}

func (e *StatusCodeError) Error() string {
	if e.ResponseText != "" {
		return fmt.Sprintf("unexpected HTTP status code %d: %s", e.Code, e.ResponseText)
	}
	return fmt.Sprintf("unexpected HTTP status code %d", e.Code)
}

// Call calls the API method serializing the given body.
//
// Returns an error if the calls returns an error, which may be a
// StatusCodeError error in case of a HTTP request which returned a status code
// which is not 200.
func (c *Chichi) Call(httpMethod, method string, body any) (any, error) {
	return c.call(httpMethod, method, body)
}

// MustCall calls the API method serializing the given body.
//
// Calls (*testing.T).Fatal if the call returns an error, or if the HTTP
// response's status code is not 200.
func (c *Chichi) MustCall(httpMethod, method string, body any) any {
	out, err := c.call(httpMethod, method, body)
	if err != nil {
		c.t.Logf("an error occurred: %s. The stack trace is:\n%s", err, string(debug.Stack()))
		c.t.Fatal("the test failed. See the error message and the stack trace above")
	}
	return out
}

func (c *Chichi) call(httpMethod, method string, body any) (any, error) {

	method = strings.TrimLeft(method, "/")
	url := "https://" + testsSettings.ChichiHost + "/" + method

	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(httpMethod, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Workspace", "1")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	c.t.Logf("[info] %s %s", httpMethod, url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, &StatusCodeError{Code: resp.StatusCode, ResponseText: string(text)}
	}

	var out any
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	extraneous, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(extraneous)) > 0 {
		return nil, fmt.Errorf("server returned extraneous data in response body: %q", string(extraneous))
	}

	return out, nil
}
