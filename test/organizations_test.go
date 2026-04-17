// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

func TestOrganizationsAPI(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	orgHeaders := http.Header{"Krenalis-Workspace": nil}

	// Create a new organization before running the subtests.
	var createResp struct {
		ID string `json:"id"`
	}
	c.MustCall("POST", "/v1/organizations", orgHeaders, map[string]any{"name": "Test Org"}, &createResp)
	orgID := createResp.ID
	if orgID == "" {
		t.Fatal("expected a non-empty organization ID")
	}

	t.Run("read organization by ID", func(t *testing.T) {
		var org struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		c.MustCall("GET", fmt.Sprintf("/v1/organization/%s", orgID), orgHeaders, nil, &org)
		if org.ID != orgID {
			t.Fatalf("expected ID %q, got %q", orgID, org.ID)
		}
		if org.Name != "Test Org" {
			t.Fatalf("expected name %q, got %q", "Test Org", org.Name)
		}
	})

	t.Run("list organizations includes new organization", func(t *testing.T) {
		var resp struct {
			Organizations []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"organizations"`
		}
		c.MustCall("GET", "/v1/organizations", orgHeaders, nil, &resp)
		found := false
		for _, org := range resp.Organizations {
			if org.ID == orgID {
				found = true
				if org.Name != "Test Org" {
					t.Fatalf("expected name %q in list, got %q", "Test Org", org.Name)
				}
				break
			}
		}
		if !found {
			t.Fatalf("organization %q not found in list", orgID)
		}
	})

	t.Run("update organization name", func(t *testing.T) {
		c.MustCall("PUT", fmt.Sprintf("/v1/organization/%s", orgID), orgHeaders, map[string]any{"name": "Updated Org"}, nil)

		var org struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		c.MustCall("GET", fmt.Sprintf("/v1/organization/%s", orgID), orgHeaders, nil, &org)
		if org.Name != "Updated Org" {
			t.Fatalf("expected name %q after update, got %q", "Updated Org", org.Name)
		}
	})

	t.Run("delete organization", func(t *testing.T) {
		c.MustCall("DELETE", fmt.Sprintf("/v1/organization/%s", orgID), orgHeaders, nil, nil)

		err := c.Call("GET", fmt.Sprintf("/v1/organization/%s", orgID), orgHeaders, nil, nil)
		if err == nil {
			t.Fatal("expected error when reading deleted organization, got nil")
		}
		statusErr, ok := err.(*krenalistester.StatusCodeError)
		if !ok {
			t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
		}
		if statusErr.Code != http.StatusNotFound {
			t.Fatalf("expected HTTP status %d, got %d", http.StatusNotFound, statusErr.Code)
		}
	})
}
