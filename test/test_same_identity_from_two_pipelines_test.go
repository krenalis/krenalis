// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestSameIdentityFromTwoPipelines(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Prevents Identity Resolution from running automatically and ensures there
	// are no identifiers.
	c.UpdateIdentityResolution(false, nil)

	dummy := c.CreateDummy("Dummy", meergotester.Source)

	// Import the "first_name" property from the first pipeline.
	pipeline1 := c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
		Name:    "Import users (1)",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "firstName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "firstName",
			},
		},
	})

	// Import the "last_name" property from the second pipeline: this will create
	// separated identities that refer to the same "identity" - from the API's
	// point of view.
	pipeline2 := c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
		Name:    "Import users (2)",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "lastName", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"last_name": "lastName",
			},
		},
	})

	// Executes the two pipelines and waits for them to complete.
	exec1 := c.ExecutePipeline(pipeline1)
	exec2 := c.ExecutePipeline(pipeline2)
	c.WaitForExecutionsCompletion(dummy, exec1, exec2)

	// Run the Identity Resolution and wait for its completion.
	c.RunIdentityResolution()

	// Check that there are 10 profiles.
	profiles, _, total := c.Profiles([]string{"first_name", "last_name"}, "first_name", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 profiles, got %d", total)
	}
	profile := profiles[0]
	if profile.Attributes["first_name"] != "Ariela" {
		t.Fatalf("expected first name %q, got %q", "Ariela", profile.Attributes["first_name"])
	}

	// Check that there are 20 identities in total.
	identities, total := c.ConnectionIdentities(dummy, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 identities, got %d", total)
	}

	// Make sure both pipelines appear 10 times, respectively each among all
	// identities imported by this connection.
	pipeline1Count, pipeline2Count := 0, 0
	for _, identity := range identities {
		switch identity.Pipeline {
		case pipeline1:
			pipeline1Count++
		case pipeline2:
			pipeline2Count++
		default:
			t.Fatalf("unexpected identity pipeline %d", identity.Pipeline)
		}
	}
	if pipeline1Count != 10 {
		t.Fatalf("expected 10 identities with pipeline %d, got %d", pipeline1, pipeline1Count)
	}
	if pipeline2Count != 10 {
		t.Fatalf("expected 10 identities with pipeline %d, got %d", pipeline2, pipeline2Count)
	}

}
