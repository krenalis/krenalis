// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestOrganizationResourceLimits checks the externally visible organization
// limits.
func TestOrganizationResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	activeOrg := activeOrganization(t, k)

	t.Run("create organization exposes limits and counts", func(t *testing.T) {
		limits := krenalistester.OrganizationLimits{
			Members:     3,
			AccessKeys:  4,
			Workspaces:  5,
			Connectors:  6,
			Connections: 7,
			Pipelines:   8,
			API: krenalistester.APILimits{
				Workspace: krenalistester.APILimit{
					QuotaPerHour:  202,
					BurstCapacity: 22,
				},
				Ingestion: krenalistester.APILimit{
					QuotaPerHour:  303,
					BurstCapacity: 33,
				},
				Nonspecific: krenalistester.APILimit{
					QuotaPerHour:  101,
					BurstCapacity: 11,
				},
			},
		}
		id := k.CreateOrganization("Limited organization", true, limits)
		org := k.Organization(id)
		if org.Limits != limits {
			t.Fatalf("expected limits %#v, got %#v", limits, org.Limits)
		}
		if org.Counts != (krenalistester.OrganizationCounts{}) {
			t.Fatalf("expected counts %#v, got %#v", krenalistester.OrganizationCounts{}, org.Counts)
		}
	})

	t.Run("access key limit is enforced", func(t *testing.T) {
		org := k.Organization(activeOrg.ID)
		limits := org.Limits
		limits.AccessKeys = org.Counts.AccessKeys
		k.UpdateOrganization(org.ID, org.Name, limits)

		err := k.TryCall("POST", "/v1/keys", nil, map[string]any{
			"name": "extra key",
			"type": krenalistester.AccessKeyTypeAPI,
		}, nil)
		expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.AccessKeysLimitReached))
	})

	t.Run("connector limit counts distinct connectors", func(t *testing.T) {
		org := k.Organization(activeOrg.ID)
		limits := org.Limits
		limits.AccessKeys = 20
		limits.Connectors = 1
		limits.Connections = 20
		limits.Pipelines = 20
		k.UpdateOrganization(org.ID, org.Name, limits)

		dummy := k.CreateDummy("Dummy 1", krenalistester.Source)
		k.CreateDummy("Dummy 2", krenalistester.Source)

		_, err := k.TryCreateConnection(krenalistester.ConnectionToCreate{
			Name:      "JavaScript",
			Role:      krenalistester.Source,
			Connector: "javascript",
			Strategy:  newDefaultStrategy(),
		})
		expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConnectorsLimitReached))

		org = k.Organization(activeOrg.ID)
		limits = org.Limits
		limits.Pipelines = org.Counts.Pipelines
		k.UpdateOrganization(org.ID, org.Name, limits)

		_, err = k.TryCreatePipeline(dummy, "User", organizationLimitsUserPipeline())
		expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.PipelinesLimitReached))
	})

	t.Run("API limits are updated", func(t *testing.T) {
		org := k.Organization(activeOrg.ID)
		limits := org.Limits
		limits.API.Workspace.QuotaPerHour = 402
		limits.API.Workspace.BurstCapacity = 42
		limits.API.Ingestion.QuotaPerHour = 503
		limits.API.Ingestion.BurstCapacity = 53
		limits.API.Nonspecific.QuotaPerHour = 301
		limits.API.Nonspecific.BurstCapacity = 31
		k.UpdateOrganization(org.ID, org.Name, limits)

		org = k.Organization(activeOrg.ID)
		if org.Limits.API != limits.API {
			t.Fatalf("expected API limits %#v, got %#v", limits.API, org.Limits.API)
		}
	})
}

// activeOrganization returns the organization created by the test harness.
func activeOrganization(t *testing.T, k *krenalistester.Krenalis) krenalistester.Organization {
	t.Helper()

	orgs := k.Organizations(0, 10)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(orgs))
	}
	return orgs[0]
}

// expectAPIError checks the HTTP status and Krenalis error code.
func expectAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error code %q, got nil", code)
	}
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T", err)
	}
	if statusErr.Response.Code != status {
		t.Fatalf("expected HTTP status %d, got %d", status, statusErr.Response.Code)
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(statusErr.Response.Text), &response); err != nil {
		t.Fatalf("expected JSON error response, got %q", statusErr.Response.Text)
	}
	if response.Error.Code != code {
		t.Fatalf("expected error code %q, got %q", code, response.Error.Code)
	}
}

// newDefaultStrategy returns the default conversion strategy as a pointer.
func newDefaultStrategy() *krenalistester.Strategy {
	return new(krenalistester.Strategy("Conversion"))
}

// organizationLimitsUserPipeline returns a minimal user pipeline for limit
// tests.
func organizationLimitsUserPipeline() krenalistester.PipelineToSet {
	return krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
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
	}
}
