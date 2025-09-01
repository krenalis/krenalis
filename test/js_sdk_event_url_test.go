//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
)

func TestJavaScriptSDKEventURL(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	const expectedJavaScriptSDKURL = "https://cdn.jsdelivr.net/npm/@meergo/javascript-sdk/dist/meergo.min.js"
	gotJavaScriptSDKURL := c.JavaScriptSDKURL()
	if gotJavaScriptSDKURL != expectedJavaScriptSDKURL {
		t.Fatalf("expected JavaScript SDK URL: %q, got: %q", expectedJavaScriptSDKURL, gotJavaScriptSDKURL)
	}

	const expectedEventURL = "http://127.0.0.1:9091/api/v1/events"
	gotEventURL := c.EventURL()
	if gotEventURL != expectedEventURL {
		t.Fatalf("expected Event URL: %q, got: %q", expectedEventURL, gotEventURL)
	}

}
