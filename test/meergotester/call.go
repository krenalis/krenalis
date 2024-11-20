//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"bytes"
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

// Call calls the API method serializing the given body and deserializing the
// response into response.
//
// Returns an error if the calls returns an error, which may be a
// StatusCodeError error in case of a HTTP request which returned a status code
// which is not 200, or if the HTTP response cannot be decoded into response.
func (c *Meergo) Call(httpMethod, method string, body, response any) error {
	return c.call(httpMethod, method, body, response)
}

// MustCall calls the API method serializing the given body and deserializing
// the response into response.
//
// Calls (*testing.T).Fatal if the call returns an error, if the HTTP response
// cannot be decoded into response, or if the HTTP response's status code is not
// 200.
func (c *Meergo) MustCall(httpMethod, method string, body, response any) {
	err := c.call(httpMethod, method, body, response)
	if err != nil {
		c.t.Logf("an error occurred: %s. The stack trace is:\n%s", err, string(debug.Stack()))
		c.t.Fatal("the test failed. See the error message and the stack trace above")
	}
}

func (c *Meergo) call(httpMethod, method string, body any, response any) error {

	method = strings.TrimLeft(method, "/")
	url := "http://" + testsSettings.MeergoHost + "/" + method

	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(httpMethod, url, data)
	if err != nil {
		return err
	}

	c.t.Logf("[info] %s %s", httpMethod, url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return &StatusCodeError{Code: resp.StatusCode, ResponseText: string(text)}
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
