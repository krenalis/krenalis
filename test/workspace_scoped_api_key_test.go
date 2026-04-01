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

// TestWorkspaceScopedAPIKeyCannotManageOrganizationWorkspaces checks that a
// workspace-scoped API key cannot access organization-wide workspace endpoints.
func TestWorkspaceScopedAPIKeyCannotManageOrganizationWorkspaces(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	token := c.CreateWorkspaceRestrictedAPIKey("workspace-scoped")
	headers := http.Header{
		"Authorization": []string{"Bearer " + token},
	}

	t.Run("list workspaces", func(t *testing.T) {
		var response any
		err := c.Call("GET", "/v1/workspaces", headers, nil, &response)
		assertWorkspaceScopedKeyRejected(t, err, "workspaces cannot be listed with a workspace restricted API key")
	})

	t.Run("create workspace", func(t *testing.T) {
		body := map[string]any{
			"name": "attacker-workspace",
		}
		err := c.Call("POST", "/v1/workspaces", headers, body, nil)
		assertWorkspaceScopedKeyRejected(t, err, "workspaces cannot be created with a workspace restricted API key")
	})

	t.Run("test workspace creation", func(t *testing.T) {
		body := map[string]any{
			"name": "attacker-workspace",
		}
		err := c.Call("POST", "/v1/workspaces/test", headers, body, nil)
		assertWorkspaceScopedKeyRejected(t, err, "workspace creation cannot be tested with a workspace restricted API key")
	})
}

func assertWorkspaceScopedKeyRejected(t *testing.T, err error, wantMessage string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected request to be rejected, got nil")
	}
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T", err)
	}
	if statusErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP status %d, got %d: %s", http.StatusUnauthorized, statusErr.Code, statusErr.ResponseText)
	}
	if !strings.Contains(statusErr.ResponseText, wantMessage) {
		t.Fatalf("expected response to contain %q, got %q", wantMessage, statusErr.ResponseText)
	}
}
