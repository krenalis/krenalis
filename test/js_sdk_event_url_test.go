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
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	const expectedJavaScriptSDKURL = "https://cdn.krenalis.com/krenalis.min.js"
	gotJavaScriptSDKURL := k.JavaScriptSDKURL()
	if gotJavaScriptSDKURL != expectedJavaScriptSDKURL {
		t.Fatalf("expected JavaScript SDK URL: %q, got: %q", expectedJavaScriptSDKURL, gotJavaScriptSDKURL)
	}

	expectedExternalEventURL := "http://" + k.Addr() + "/v1/events"
	gotExternalEventURL := k.ExternalEventURL()
	if gotExternalEventURL != expectedExternalEventURL {
		t.Fatalf("expected external event URL: %q, got: %q", expectedExternalEventURL, gotExternalEventURL)
	}

}
