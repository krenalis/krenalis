// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

func TestJavaScriptSDKEventURL(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	const expectedJavaScriptSDKURL = "https://cdn.krenalis.com/krenalis.min.js"
	gotJavaScriptSDKURL := c.JavaScriptSDKURL()
	if gotJavaScriptSDKURL != expectedJavaScriptSDKURL {
		t.Fatalf("expected JavaScript SDK URL: %q, got: %q", expectedJavaScriptSDKURL, gotJavaScriptSDKURL)
	}

	expectedExternalEventURL := "http://" + c.Addr() + "/v1/events"
	gotExternalEventURL := c.ExternalEventURL()
	if gotExternalEventURL != expectedExternalEventURL {
		t.Fatalf("expected external event URL: %q, got: %q", expectedExternalEventURL, gotExternalEventURL)
	}

}
