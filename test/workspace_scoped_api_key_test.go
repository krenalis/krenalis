// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

const organizationRequestCannotSpecifyWorkspace = "organization request cannot specify a workspace"

// TestWorkspaceScopedAPIKeyCannotManageOrganizationWorkspaces checks that a
// workspace-scoped API key cannot access organization-wide workspace endpoints.
func TestWorkspaceScopedAPIKeyCannotManageOrganizationWorkspaces(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	token := k.CreateWorkspaceRestrictedAPIKey("workspace-scoped")
	headers := http.Header{
		"Authorization":      []string{"Bearer " + token},
		"Krenalis-Workspace": nil, // Test endpoint authorization rather than conflicting-header validation.
	}

	t.Run("list workspaces", func(t *testing.T) {
		var response any
		err := k.TryCall("GET", "/v1/workspaces", headers, nil, &response)
		assertWorkspaceScopedKeyRejected(t, err)
	})

	t.Run("create workspace", func(t *testing.T) {
		body := map[string]any{
			"name": "attacker-workspace",
		}
		err := k.TryCall("POST", "/v1/workspaces", headers, body, nil)
		assertWorkspaceScopedKeyRejected(t, err)
	})

	t.Run("test workspace creation", func(t *testing.T) {
		body := map[string]any{
			"name": "attacker-workspace",
		}
		err := k.TryCall("POST", "/v1/workspaces/test", headers, body, nil)
		assertWorkspaceScopedKeyRejected(t, err)
	})
}

func assertWorkspaceScopedKeyRejected(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected request to be rejected, got nil")
	}
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T", err)
	}
	if statusErr.Response.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP status %d, got %d: %s", http.StatusUnauthorized, statusErr.Response.Code, statusErr.Response.Text)
	}
	if !strings.Contains(statusErr.Response.Text, organizationRequestCannotSpecifyWorkspace) {
		t.Fatalf("expected response to contain %q, got %q", organizationRequestCannotSpecifyWorkspace, statusErr.Response.Text)
	}
}
