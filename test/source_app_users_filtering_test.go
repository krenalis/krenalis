//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestSourceAppUsersFiltering(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Import users from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
	importUsersID := c.CreateAction(dummySrc, "Users", meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		Filter: &meergotester.Filter{
			Logical: meergotester.OpAnd,
			Conditions: []meergotester.FilterCondition{
				{
					Property: "email",
					Operator: meergotester.OpIsNot,
					Values:   []string{"kdericut4@example.com"},
				},
			},
		},
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
	})
	exec := c.ExecuteAction(importUsersID)
	c.WaitForExecutionsCompletionAllowFailed(dummySrc, exec)

	_, _, total := c.Users([]string{"email"}, "", false, 0, 100)

	// Dummy exposes 10 users, but one of them was filtered out, so there must
	// be 9.
	const expectedCount = 9
	if expectedCount != total {
		t.Fatalf("expected %d users, got %d", expectedCount, total)
	}
	t.Logf("the APIs successfully returned %d users", total)
}
