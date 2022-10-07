//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Stream returns a Reader and a Writer to read from and write to a connector.
// url is the URL of the connector and rt is the round tripper to use for the
// request. The caller is responsible for closing the Reader and the Writer.
func Stream(ctx context.Context, rt http.RoundTripper, url string) (io.ReadCloser, io.WriteCloser, error) {
	r, w := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, "POST", url, r)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := rt.RoundTrip(req)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("unexpected status code %d", res.StatusCode)
	}
	return res.Body, w, nil
}
