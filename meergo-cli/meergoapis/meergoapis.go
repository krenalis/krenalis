//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergoapis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

var meergoAPIs struct {
	url         string
	workspaceID int
	httpClient  *http.Client
}

var initialized bool

// Init initializes this package to connect to the Meergo APIs. url is the URL
// of the APIs, while workspaceID is the ID of the workspace which interacts
// with the APIs. This method should be called only once.
func Init(url string, workspaceID int) {
	if initialized {
		panic("already initialized")
	}
	meergoAPIs.url = url
	meergoAPIs.workspaceID = workspaceID
	initialized = true
}

// callAPI calls the given path on the Meergo APIs, passing body in the
// request (which is serialized in JSON). It deserializes the response in the
// response argument if not nil.
func callAPI(method string, path string, body io.Reader, response any) error {

	// Some initial validation.
	if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" {
		panic("method must be GET or POST")
	}
	if method == "GET" && body != nil {
		panic("body must be nil for the GET method")
	}
	if strings.HasPrefix(path, "/") {
		panic("path should not begin with /")
	}
	if !initialized {
		panic("package 'meergoapis' not initialized")
	}

	// Get or initialize the HTTP client.
	httpClient := meergoAPIs.httpClient
	if httpClient == nil {
		// Create an HTTP client which does not follow redirects and accepts cookies.
		jar, err := cookiejar.New(nil)
		if err != nil {
			panic(fmt.Sprintf("cannot create a cookie jar: %s", err))
		}
		httpClient = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return errors.New("redirect")
			},
			Jar: jar,
		}
		meergoAPIs.httpClient = httpClient
		// Log in.
		err = callAPI("POST", "api/members/login", strings.NewReader(`{"Email":"acme@open2b.com","Password":"foopass2"}`), nil)
		if err != nil {
			err = fmt.Errorf("cannot login: %s", err)
			meergoAPIs.httpClient = nil
			return err
		}
	}

	// Call the APIs.
	url := meergoAPIs.url + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Workspace", "1")
	resp, err := httpClient.Do(req)
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
		if resp.Header.Get("Content-Type") == "application/json" {
			// Read a JSON response.
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return fmt.Errorf("cannot decode JSON response from %q: %s", url, err)
			}
		} else {
			// Read a plain text response.
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("cannot read response body from %q: %s", url, err)
			}
			switch response := response.(type) {
			case *string:
				*response = string(body)
			case *[]byte:
				*response = body
			default:
				return fmt.Errorf("cannot decode the response body into a %T value", response)
			}
		}
	}

	return nil
}
