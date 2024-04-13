//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package collector

import (
	"bytes"
	"encoding/json"
	"io"

	"golang.org/x/text/unicode/norm"
)

// parse parses a request with events. If batch is true, the request is a batch
// request.
func parse(r io.Reader, batch bool) (*batchEvents, error) {

	// Read the body and check that is not be longer than maxRequestSize bytes and,
	// it is a streaming of JSON objects, otherwise return the errBadRequest error.
	lr := &io.LimitedReader{R: r, N: maxRequestSize + 1}
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if lr.N == 0 {
		return nil, errBadRequest
	}
	b := buf.Bytes()

	// Read the events.
	nr := norm.NFC.Reader(bytes.NewReader(b))
	dec := json.NewDecoder(nr)
	dec.UseNumber()

	var request batchEvents
	if batch {
		err = dec.Decode(&request)
	} else {
		request.Batch = make([]*collectedEvent, 1)
		err = dec.Decode(&request.Batch[0])
	}
	if err != nil {
		return nil, errBadRequest
	}
	if len(request.Batch) == 0 {
		return nil, errBadRequest
	}

	return &request, nil
}
