// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestEventRateLimiterAcceptsMaximumRequestBatch verifies that the minimum
// event capacity admits the largest valid batch that fits in a request body.
func TestEventRateLimiterAcceptsMaximumRequestBatch(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	connectionID := k.CreateJavaScriptSource("Rate-limited source", nil)
	writeKeys := k.EventWriteKeys(connectionID)
	if len(writeKeys) != 1 {
		t.Fatalf("expected one event write key, got %d", len(writeKeys))
	}

	// Keep this value in sync with collector.maxRequestSize.
	const maxRequestSize = 500 * 1024
	event := map[string]any{"type": "page", "userId": "u"}
	encodedEvent, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	// A JSON array needs one byte per element for its comma or closing bracket,
	// plus its opening bracket.
	eventCount := (maxRequestSize - 1) / (len(encodedEvent) + 1)
	events := make([]map[string]any, eventCount)
	for i := range events {
		events[i] = event
	}
	body, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > maxRequestSize {
		t.Fatalf("expected request body not to exceed %d bytes, got %d", maxRequestSize, len(body))
	}
	if len(body)+len(encodedEvent)+1 <= maxRequestSize {
		t.Fatalf("expected one more event to exceed the %d-byte request limit", maxRequestSize)
	}

	if err := k.TryCallWithoutRetry(http.MethodPost, "/v1/events",
		http.Header{"Authorization": []string{"Bearer " + writeKeys[0]}},
		events, nil); err != nil {
		t.Fatal(err)
	}
}
