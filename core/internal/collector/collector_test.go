// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meergo/meergo/core/internal/streams/streamstest"
)

type repeatReader struct {
	b byte
}

func (r repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
	}
	return len(p), nil
}

func TestCollectorReturnsRequestEntityTooLarge(t *testing.T) {
	c := &Collector{
		sc: &streamstest.Connection{
			StreamValue: &streamstest.Stream{},
			WaitUpValue: true,
		},
	}
	body := io.LimitReader(repeatReader{b: 'a'}, int64(maxRequestSize+1))
	req := httptest.NewRequest(http.MethodPost, "/events", body)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	c.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
}
