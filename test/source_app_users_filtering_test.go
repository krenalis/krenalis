// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestSourceAppUsersFiltering(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Import users from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", krenalistester.Source)
	importUsersID := c.CreatePipeline(dummySrc, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		Filter: &krenalistester.Filter{
			Logical: krenalistester.OpAnd,
			Conditions: []krenalistester.FilterCondition{
				{
					Property: "email",
					Operator: krenalistester.OpIsNot,
					Values:   []string{"kdericut4@example.com"},
				},
			},
		},
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
	})
	run := c.RunPipeline(importUsersID)
	c.WaitForRunsCompletionAllowFailed(dummySrc, run)

	_, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)

	// Dummy exposes 10 profiles, but one of them was filtered out, so there must
	// be 9.
	const expectedCount = 9
	if expectedCount != total {
		t.Fatalf("expected %d profiles, got %d", expectedCount, total)
	}
	t.Logf("the APIs successfully returned %d profiles", total)
}
