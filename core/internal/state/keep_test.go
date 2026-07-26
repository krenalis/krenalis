// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"slices"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state/ratelimiter"
)

func TestAddAndRemoveLinkedConnection(t *testing.T) {
	const (
		connA = "2Qn5zBpR9YH7"
		connB = "5zBpR9Y2QnM3"
		connC = "8QaT3mN7KxP5"
		connD = "B7mN9qK2xAC3"
		connE = "G3mN7Kx8QaD4"
	)

	tests := []struct {
		id      string
		with    []string
		without []string
	}{
		{connA, []string{connA}, []string{}},
		{connA, []string{connA, connB}, []string{connB}},
		{connB, []string{connA, connB}, []string{connA}},
		{connC, []string{connB, connC, connD, connE}, []string{connB, connD, connE}},
		{connE, []string{connA, connC, connD, connE}, []string{connA, connC, connD}},
	}

	// Test the addLinkedConnection function.
	for _, test := range tests {
		without := slices.Clone(test.without)
		got := addLinkedConnection(test.without, test.id)
		if got == nil {
			t.Fatalf("expected %#v, got nil", test.with)
		}
		if !slices.Equal(test.with, got) {
			t.Fatalf("expected %#v, got %#v", test.with, got)
		}
		if !slices.Equal(without, test.without) {
			t.Fatalf("the 'without' slice has been changed")
		}
	}

	// Test the removeLinkedConnection function.
	for _, test := range tests {
		with := slices.Clone(test.with)
		got := removeLinkedConnection(test.with, test.id)
		if got == nil {
			t.Fatal("unexpected nil")
		}
		if !slices.Equal(test.without, got) {
			t.Fatalf("expected %#v, got %#v", test.without, got)
		}
		if !slices.Equal(with, test.with) {
			t.Fatalf("the 'with' slice has been changed")
		}
	}

}

// TestReplaceOrganizationPreservesRateLimitBucket verifies that replacing an
// organization retains its local rate-limit bucket.
func TestReplaceOrganizationPreservesRateLimitBucket(t *testing.T) {
	const organizationID = "111111111111"
	bucket := ratelimiter.NewBucket("test", organizationID, 1, 1)
	organization := &Organization{
		mu:         new(sync.Mutex),
		workspaces: map[string]*Workspace{},
		bucket:     bucket,
		ID:         organizationID,
	}
	state := &State{
		mu:            new(sync.Mutex),
		organizations: map[string]*Organization{organizationID: organization},
	}

	updated := state.replaceOrganization(organizationID, func(organization *Organization) {
		organization.Name = "updated"
	})

	if updated.bucket != bucket {
		t.Fatal("organization update replaced its rate-limit bucket")
	}
}

// TestReplaceWorkspacePreservesRateLimitBuckets verifies that replacing a
// workspace retains both of its local rate-limit buckets.
func TestReplaceWorkspacePreservesRateLimitBuckets(t *testing.T) {
	const (
		organizationID = "111111111111"
		workspaceID    = "222222222222"
	)
	organization := &Organization{
		mu:         new(sync.Mutex),
		workspaces: map[string]*Workspace{},
		ID:         organizationID,
	}
	bucket := ratelimiter.NewBucket("test", workspaceID, 1, 1)
	eventBucket := ratelimiter.NewBucket("test-events", workspaceID, 1, 1)
	workspace := &Workspace{
		mu:           new(sync.Mutex),
		organization: organization,
		bucket:       bucket,
		eventBucket:  eventBucket,
		ID:           workspaceID,
	}
	organization.workspaces[workspaceID] = workspace
	state := &State{
		mu:            new(sync.Mutex),
		organizations: map[string]*Organization{organizationID: organization},
		workspaces:    map[string]*Workspace{workspaceID: workspace},
	}

	updated := state.replaceWorkspace(workspaceID, func(workspace *Workspace) {
		workspace.Name = "updated"
	})

	if updated.bucket != bucket {
		t.Fatal("workspace update replaced its rate-limit bucket")
	}
	if updated.eventBucket != eventBucket {
		t.Fatal("workspace update replaced its event rate-limit bucket")
	}
}
