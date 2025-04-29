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

func TestCDNEventURL(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	const expectedCDNURL = "http://127.0.0.1:9091"
	gotCDNURL := c.CDNURL()
	if gotCDNURL != expectedCDNURL {
		t.Fatalf("expected CDN URL: %q, got: %q", expectedCDNURL, gotCDNURL)
	}

	const expectedEventURL = "http://127.0.0.1:9091/api/v1/events"
	gotEventURL := c.EventURL()
	if gotEventURL != expectedEventURL {
		t.Fatalf("expected Event URL: %q, got: %q", expectedEventURL, gotEventURL)
	}

}
