//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/open2b/chichi/test/chichitester"

	"golang.org/x/exp/maps"
)

func TestConnectionSettings(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	settings := map[string]any{
		"LargeDataset": true,
	}

	dummy := c.AddDummyWithSettings("Dummy (source)", chichitester.Source, settings)
	ui := c.GetConnectionUI(dummy)
	values := ui["Form"].(map[string]any)["Values"].(map[string]any)

	if !maps.Equal(settings, values) {
		t.Fatalf("expected settings %#v, got %#v", settings, values)
	}

}
