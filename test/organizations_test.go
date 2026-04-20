// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
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

	// Create a new organization before running the subtests.
	orgID := c.CreateOrganization("Test Org")

	t.Run("read organization by ID", func(t *testing.T) {
		org := c.Organization(orgID)
		if org.ID != orgID {
			t.Fatalf("expected ID %q, got %q", orgID, org.ID)
		}
		if org.Name != "Test Org" {
			t.Fatalf("expected name %q, got %q", "Test Org", org.Name)
		}
	})

	t.Run("list organizations includes new organization", func(t *testing.T) {
		orgs := c.Organizations(0, 100)
		found := false
		for _, org := range orgs {
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
		c.UpdateOrganization(orgID, "Updated Org")
		org := c.Organization(orgID)
		if org.Name != "Updated Org" {
			t.Fatalf("expected name %q after update, got %q", "Updated Org", org.Name)
		}
	})

	t.Run("delete organization", func(t *testing.T) {
		c.DeleteOrganization(orgID)
		err := c.OrganizationErr(orgID)
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
