// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
)

func Test_WorkspaceIdentifiers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Test the default value for ResolveIdentitiesOnBatchImport, when a
	// workspace is created.
	if ws := c.Workspace(); !ws.ResolveIdentitiesOnBatchImport {
		t.Fatalf("expected ResolveIdentitiesOnBatchImport to be true (which is the default), got %t", ws.ResolveIdentitiesOnBatchImport)
	}

	if ws := c.Workspace(); ws.Identifiers == nil || len(ws.Identifiers) != 0 {
		t.Fatalf("expected an empty slice, got %v", ws.Identifiers)
	}
	c.UpdateIdentityResolution(true, []string{})
	if ws := c.Workspace(); !ws.ResolveIdentitiesOnBatchImport {
		t.Fatalf("expected ResolveIdentitiesOnBatchImport to be true, got %t", ws.ResolveIdentitiesOnBatchImport)
	}
	if ws := c.Workspace(); ws.Identifiers == nil || len(ws.Identifiers) != 0 {
		t.Fatalf("expected an empty slice, got %v", ws.Identifiers)
	}
	c.UpdateIdentityResolution(true, []string{"dummy_id"})
	if ws := c.Workspace(); len(ws.Identifiers) != 1 || ws.Identifiers[0] != "dummy_id" {
		t.Fatalf("expected \"dummy_id\", got %v", ws.Identifiers)
	}
	c.UpdateIdentityResolution(true, []string{"email", "android.id"})
	if ws := c.Workspace(); len(ws.Identifiers) != 2 || ws.Identifiers[0] != "email" || ws.Identifiers[1] != "android.id" {
		t.Fatalf("expected \"email\" and \"android.id\", got %v", ws.Identifiers)
	}

	// Test an invalid path.
	err := c.UpdateIdentityResolutionErr([]string{"invalid path"})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
	expected := `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identifier \"invalid path\" is not a valid property path"}}`
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err)
	}

	// Test a not existent path in the user schema.
	err = c.UpdateIdentityResolutionErr([]string{"non_existent"})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
	expected = `unexpected HTTP status code 422: {"error":{"code":"PropertyNotExist","message":"property \"non_existent\" does not exist in the user schema"}}`
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err)
	}

	// Test a not allowed type for identifiers.
	err = c.UpdateIdentityResolutionErr([]string{"phone_numbers"})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
	expected = `unexpected HTTP status code 422: {"error":{"code":"TypeNotAllowed","message":"property \"phone_numbers\" has a type array(text), which is not allowed for identifiers"}}`
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err)
	}

	// Test the disabling of ResolveIdentitiesOnBatchImport.
	c.UpdateIdentityResolution(false, []string{})
	if ws := c.Workspace(); ws.ResolveIdentitiesOnBatchImport {
		t.Fatalf("expected ResolveIdentitiesOnBatchImport to be false, got %t", ws.ResolveIdentitiesOnBatchImport)
	}

	// Test the enabling of ResolveIdentitiesOnBatchImport.
	c.UpdateIdentityResolution(true, []string{})
	if ws := c.Workspace(); !ws.ResolveIdentitiesOnBatchImport {
		t.Fatalf("expected ResolveIdentitiesOnBatchImport to be true, got %t", ws.ResolveIdentitiesOnBatchImport)
	}

}
