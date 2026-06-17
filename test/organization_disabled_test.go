// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/krenalis/analytics-go"
)

// TestOrganizationDisabled tests the disable/enable flow of an organization.
func TestOrganizationDisabled(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Retrieve the organization ID.
	orgs := c.Organizations(0, 100)
	if len(orgs) != 1 {
		t.Fatalf("expected exactly one organization, got %d", len(orgs))
	}
	orgID := orgs[0].ID

	// Test that the organization is enabled at startup.
	if !orgs[0].Enabled {
		t.Fatal("organization should be enabled, but it is disabled")
	}

	// Test that the call to the method that sets the state of an organization
	// fails if the organizations key is not provided.
	t.Run("set status without organizations API key is rejected", func(t *testing.T) {
		err := c.SetOrganizationStatusErr(orgID, false, http.Header{"Krenalis-Workspace": nil})
		statusErr, ok := err.(*krenalistester.StatusCodeError)
		if !ok {
			t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
		}
		if statusErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected HTTP status %d, got %d: %s", http.StatusUnauthorized, statusErr.Code, statusErr.ResponseText)
		}
		// The organization must still be enabled: the request was rejected
		// before it could change anything.
		if !c.Organization(orgID).Enabled {
			t.Fatal("the unauthenticated request must not have changed the organization status")
		}
	})

	// Configure identity resolution, set up (and then run) a simple pipeline
	// that imports users from a Dummy source, while the organization is
	// enabled.
	c.UpdateIdentityResolution(true, []string{"email"})
	dummySrc := c.CreateDummy("Dummy (source)", krenalistester.Source)
	importPipeline := c.CreatePipeline(dummySrc, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "firstName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
			},
		},
	})
	run := c.RunPipeline(importPipeline)
	c.WaitRunsCompletion(dummySrc, run)

	// Set up a JavaScript source with an Event pipeline so that events can be
	// ingested through the /v1/events endpoint. This must be done while the
	// organization is still enabled, as creating connections and pipelines is
	// itself rejected once the organization is disabled.
	jsSrc := c.CreateJavaScriptSource("JavaScript (source)", nil)
	keys := c.EventWriteKeys(jsSrc)
	if len(keys) != 1 {
		t.Fatalf("expected exactly one event write key, got %d", len(keys))
	}
	writeKey := keys[0]
	c.CreatePipeline(jsSrc, "Event", krenalistester.PipelineToSet{
		Name:    "Store events",
		Enabled: true,
	})

	// Disable the organization.
	c.SetOrganizationStatus(orgID, false)

	// The organization endpoint must still be reachable and must report the new
	// status.
	org := c.Organization(orgID)
	if org.Enabled {
		t.Fatal("expected the organization to be reported as disabled after disabling it")
	}

	// Test that API calls are rejected with the Unprocessable
	// "OrganizationDisabled" error (except events ingestion, tested below).

	t.Run("create connection is rejected", func(t *testing.T) {
		_, err := c.CreateConnectionErr(krenalistester.ConnectionToCreate{
			Name:      "Dummy that should not be created",
			Role:      krenalistester.Source,
			Connector: "dummy",
			Settings:  json.Value("{}"),
		})
		assertOrganizationDisabled(t, err)
	})

	t.Run("create pipeline is rejected", func(t *testing.T) {
		_, err := c.CreatePipelineErr(dummySrc, "User", krenalistester.PipelineToSet{
			Name:    "Pipeline that should not be created",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			Transformation: &krenalistester.Transformation{
				Mapping: map[string]string{"email": "email"},
			},
		})
		assertOrganizationDisabled(t, err)
	})

	t.Run("run pipeline is rejected", func(t *testing.T) {
		_, err := c.RunPipelineErr(importPipeline)
		assertOrganizationDisabled(t, err)
	})

	t.Run("delete pipeline is rejected", func(t *testing.T) {
		err := c.DeletePipelineErr(importPipeline)
		assertOrganizationDisabled(t, err)
	})

	t.Run("delete connection is rejected", func(t *testing.T) {
		err := c.DeleteConnectionErr(dummySrc)
		assertOrganizationDisabled(t, err)
	})

	t.Run("start identity resolution is rejected", func(t *testing.T) {
		err := c.StartIdentityResolutionErr()
		assertOrganizationDisabled(t, err)
	})

	t.Run("repair warehouse is rejected", func(t *testing.T) {
		err := c.RepairWarehouseErr()
		assertOrganizationDisabled(t, err)
	})

	t.Run("update identity resolution is rejected", func(t *testing.T) {
		err := c.UpdateIdentityResolutionErr([]string{"email"})
		assertOrganizationDisabled(t, err)
	})

	t.Run("read events is rejected", func(t *testing.T) {
		err := c.EventsErr([]string{"type"})
		assertOrganizationDisabled(t, err)
	})

	t.Run("read event schema is rejected", func(t *testing.T) {
		err := c.Call("GET", "/v1/events/schema", nil, nil, nil)
		assertOrganizationDisabled(t, err)
	})

	t.Run("create event listener is rejected", func(t *testing.T) {
		err := c.Call("POST", "/v1/events/listeners", nil, map[string]any{}, nil)
		assertOrganizationDisabled(t, err)
	})

	t.Run("read listened events is rejected", func(t *testing.T) {
		err := c.Call("GET", "/v1/events/listeners/nonexistent", nil, nil, nil)
		assertOrganizationDisabled(t, err)
	})

	t.Run("delete event listener is rejected", func(t *testing.T) {
		err := c.Call("DELETE", "/v1/events/listeners/nonexistent", nil, nil, nil)
		assertOrganizationDisabled(t, err)
	})

	t.Run("alter profile schema is rejected", func(t *testing.T) {
		err := c.AlterProfileSchemaErr(types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(254), ReadOptional: true},
		}), nil, nil)
		assertOrganizationDisabled(t, err)
	})

	// Event ingestion must NOT be rejected: while the organization is disabled,
	// incoming events are silently dropped (treated as if every pipeline were
	// disabled) and the request still succeeds. This applies both to the batch
	// endpoint (POST /v1/events) and to the single typed-event endpoint
	// (POST /v1/events/{type}).
	t.Run("event ingestion is accepted but dropped", func(t *testing.T) {
		// POST /v1/events: batch ingestion through the SDK. SendEvent fails the
		// test if the server answers with an error, so a successful call proves
		// the ingest endpoint responded with success.
		c.SendEvent(writeKey, analytics.Track{
			UserId: "dropped-while-disabled",
			Event:  "Event sent while disabled",
		})
		// POST /v1/events/{type}: single typed event authenticated with the
		// event write key. Like POST /v1/events, it must not be rejected while
		// the organization is disabled.
		err := c.Call("POST", "/v1/events/track",
			http.Header{"Authorization": []string{"Bearer " + writeKey}},
			map[string]any{
				"userId": "dropped-while-disabled-typed",
				"event":  "Typed event sent while disabled",
			}, nil)
		if err != nil {
			t.Fatalf("POST /v1/events/track must not be rejected while the organization is disabled, got: %v", err)
		}
		time.Sleep(2 * time.Second)
		// No event must have been stored: nothing is queued for a disabled
		// organization, regardless of the ingestion endpoint used.
		if count := c.CountEventsInWarehouse(t.Context()); count != 0 {
			t.Fatalf("expected no events stored while the organization is disabled, got %d", count)
		}
	})

	// Updating the organization's name uses the organizations API key, so it
	// must remain functional even while the organization is disabled.
	t.Run("update organization name still works", func(t *testing.T) {
		c.UpdateOrganization(orgID, "ACME inc (renamed while disabled)")
		got := c.Organization(orgID)
		if got.Name != "ACME inc (renamed while disabled)" {
			t.Fatalf("expected the organization name to be updated, got %q", got.Name)
		}
		if got.Enabled {
			t.Fatal("renaming should not have re-enabled the organization")
		}
	})

	// Setting the organization status to the same value it currently has.
	t.Run("setting the same status is a no-op", func(t *testing.T) {
		c.SetOrganizationStatus(orgID, false)
		if c.Organization(orgID).Enabled {
			t.Fatal("organization should still be disabled")
		}
	})

	// Re-enable the organization.
	c.SetOrganizationStatus(orgID, true)
	if !c.Organization(orgID).Enabled {
		t.Fatal("expected the organization to be reported as enabled after re-enabling it")
	}

	t.Run("operations succeed again after re-enabling", func(t *testing.T) {
		run := c.RunPipeline(importPipeline)
		c.WaitRunsCompletion(dummySrc, run)
	})

	t.Run("event ingestion works again after re-enabling", func(t *testing.T) {
		// POST /v1/events: batch ingestion.
		c.SendEvent(writeKey, analytics.Track{
			UserId: "stored-after-reenabling",
			Event:  "Event sent after re-enabling",
		})
		// POST /v1/events/{type}: single typed event. It too must be ingested
		// (and stored) again now that the organization is enabled.
		err := c.Call("POST", "/v1/events/track",
			http.Header{"Authorization": []string{"Bearer " + writeKey}},
			map[string]any{
				"userId": "stored-after-reenabling-typed",
				"event":  "Typed event sent after re-enabling",
			}, nil)
		if err != nil {
			t.Fatalf("POST /v1/events/track must succeed after re-enabling the organization, got: %v", err)
		}
		time.Sleep(2 * time.Second)
		// Exactly two events must be stored: the batch one and the typed one,
		// both sent after re-enabling.
		c.WaitEventsStoredIntoWarehouse(t.Context(), 2)
	})
}

// assertOrganizationDisabled fails the test if err is not the expected error
// for an operation rejected because the organization is disabled.
func assertOrganizationDisabled(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
	}
	if statusErr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected HTTP status %d, got %d: %s", http.StatusUnprocessableEntity, statusErr.Code, statusErr.ResponseText)
	}
	if !strings.Contains(statusErr.ResponseText, `"code":"OrganizationDisabled"`) {
		t.Fatalf("expected error code %q in response, got: %s", "OrganizationDisabled", statusErr.ResponseText)
	}
}
