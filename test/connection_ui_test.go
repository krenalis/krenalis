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

func TestConnectionUI(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	values := map[string]any{
		"LargeDataset": true,
	}

	dummy := c.AddDummyWithUIValues("Dummy (source)", chichitester.Source, values)
	ui := c.GetConnectionUI(dummy)
	got := ui["Values"].(map[string]any)

	if !maps.Equal(values, got) {
		t.Fatalf("expected values %#v, got %#v", values, got)
	}

}
