// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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

	const expectedJavaScriptSDKURL = "https://cdn.meergo.com/meergo.min.js"
	gotJavaScriptSDKURL := c.JavaScriptSDKURL()
	if gotJavaScriptSDKURL != expectedJavaScriptSDKURL {
		t.Fatalf("expected JavaScript SDK URL: %q, got: %q", expectedJavaScriptSDKURL, gotJavaScriptSDKURL)
	}

	expectedExternalEventURL := "http://" + c.Addr() + "/api/v1/events"
	gotExternalEventURL := c.ExternalEventURL()
	if gotExternalEventURL != expectedExternalEventURL {
		t.Fatalf("expected external event URL: %q, got: %q", expectedExternalEventURL, gotExternalEventURL)
	}

}
