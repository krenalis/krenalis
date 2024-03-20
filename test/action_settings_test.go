//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestActionSettings(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	dummy := c.AddDummy("Dummy (source)", chichitester.Source, "")
	action := c.AddAction(dummy, "Users", chichitester.ActionToSet{
		Name:           "Action #1",
		Enabled:        true,
		InSchema:       types.Object([]types.Property{{Name: "email", Type: types.Text()}}),
		OutSchema:      types.Object([]types.Property{{Name: "email", Type: types.Text()}}),
		Transformation: chichitester.Transformation{Mapping: map[string]string{"email": "email"}},
		Settings:       nil,
	})
	gotAction := c.Action(dummy, action)

	if gotAction.Settings != nil {
		t.Fatalf("expected nil settings, got %v", gotAction.Settings)
	}

}
