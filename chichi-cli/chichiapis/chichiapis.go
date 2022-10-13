//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package chichiapis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var chichiAPIs struct {
	url       string
	accountID int
}

var initialized bool

// Init initializes this package to connect to the Chichi APIs. url is the URL
// of the APIs, while accountID is the ID of the account which interacts with
// the APIs. This method should be called only once.
func Init(url string, accountID int) {
	if initialized {
		panic("already initialized")
	}
	chichiAPIs.url = url
	chichiAPIs.accountID = accountID
	initialized = true
}

// callAdmin calls the given method on the Chichi Admin APIs, passing body in
// the request (which is serialized in JSON). Returns the method response
// de-serialized from JSON.
func callAdmin(method string, body any) (any, error) {

	// Some initial validation.
	if strings.HasPrefix(method, "/") {
		panic("method should not begin with /")
	}
	if !initialized {
		panic("package 'chichiapis' not initialized")
	}

	// Create an HTTP client which does not follow redirects.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirect")
		},
	}

	// Call the APIs.
	url := chichiAPIs.url + method
	jsonBody := &bytes.Buffer{}
	err := json.NewEncoder(jsonBody).Encode(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, jsonBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "session="+strconv.Itoa(chichiAPIs.accountID))
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot POST on %q: %s", url, err)
	}
	defer resp.Body.Close()

	// Check the status code.
	if resp.StatusCode != http.StatusOK {
		respText, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("got unexpected status %d from %q, response body: %s", resp.StatusCode, url, respText)
	}

	// Return the result.
	var v any
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return nil, fmt.Errorf("cannot decode JSON response from %q: %s", url, err)
	}
	return v, nil
}

// callAPI calls the given path on the Chichi APIs, passing body in the
// request (which is serialized in JSON). It deserializes the response in the
// response argument if not nil.
func callAPI(method string, path string, body io.Reader, response any) error {

	// Some initial validation.
	if method != "GET" && method != "POST" {
		panic("method must be GET or POST")
	}
	if method == "GET" && body != nil {
		panic("body must be nil for the GET method")
	}
	if strings.HasPrefix(path, "/") {
		panic("path should not begin with /")
	}
	if !initialized {
		panic("package 'chichiapis' not initialized")
	}

	// Create an HTTP client which does not follow redirects.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirect")
		},
	}

	// Call the APIs.
	url := chichiAPIs.url + path
	jsonBody := &bytes.Buffer{}
	if body != nil {
		err := json.NewEncoder(jsonBody).Encode(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, url, jsonBody)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot POST on %q: %s", url, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// Check the status code.
	if resp.StatusCode != http.StatusOK {
		respText, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("got unexpected status %d from %q, response body: %s", resp.StatusCode, url, respText)
	}

	// Return the result.
	if response != nil {
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return fmt.Errorf("cannot decode JSON response from %q: %s", url, err)
		}
	}

	return nil
}
