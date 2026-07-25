// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"database/sql/driver"
	"fmt"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state/ratelimiter"
)

type valuerStringer interface {
	driver.Valuer
	String() string
}

// TestValuerStringerConsistency verifies that String and Value agree.
func TestValuerStringerConsistency(t *testing.T) {
	tests := []struct {
		name string
		v    valuerStringer
		want string
	}{
		{"AccessKeyTypeAPI", AccessKeyTypeAPI, "API"},
		{"AccessKeyTypeMCP", AccessKeyTypeMCP, "MCP"},
		{"Normal", Normal, "Normal"},
		{"Inspection", Inspection, "Inspection"},
		{"Maintenance", Maintenance, "Maintenance"},
		{"Application", Application, "Application"},
		{"Database", Database, "Database"},
		{"File", File, "File"},
		{"FileStorage", FileStorage, "FileStorage"},
		{"MessageBroker", MessageBroker, "MessageBroker"},
		{"SDK", SDK, "SDK"},
		{"Webhook", Webhook, "Webhook"},
		{"WebhooksPerNone", WebhooksPerNone, "None"},
		{"WebhooksPerAccount", WebhooksPerAccount, "Account"},
		{"WebhooksPerConnection", WebhooksPerConnection, "Connection"},
		{"WebhooksPerConnector", WebhooksPerConnector, "Connector"},
		{"Healthy", Healthy, "Healthy"},
		{"NoRecentData", NoRecentData, "NoRecentData"},
		{"RecentError", RecentError, "RecentError"},
		{"Source", Source, "Source"},
		{"Destination", Destination, "Destination"},
		{"TargetEvent", TargetEvent, "Event"},
		{"TargetUser", TargetUser, "User"},
		{"TargetGroup", TargetGroup, "Group"},
		{"JavaScript", JavaScript, "JavaScript"},
		{"Python", Python, "Python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.v.Value()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Value() = %v, want %s", got, tt.want)
			}
			if got := tt.v.String(); got != tt.want {
				t.Fatalf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestValuerStringerInvalidValues verifies invalid value handling.
func TestValuerStringerInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		v    valuerStringer
	}{
		{"AccessKeyType", AccessKeyType(-1)},
		{"WarehouseMode", WarehouseMode(-1)},
		{"ConnectorType", ConnectorType(-1)},
		{"WebhooksPer", WebhooksPer(-1)},
		{"Health", Health(-1)},
		{"Role", Role(-1)},
		{"Target", Target(-1)},
		{"Language", Language(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.v.Value()
			if err == nil {
				t.Fatal("Value() did not return an error")
			}

			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("String() did not panic")
				}
				if fmt.Sprint(got) != err.Error() {
					t.Fatalf("panic = %v, want %s", got, err)
				}
			}()
			_ = tt.v.String()
		})
	}
}

// TestReplacePreservesRateLimitBuckets verifies that entity updates retain
// their separate local rate-limit buckets.
func TestReplacePreservesRateLimitBuckets(t *testing.T) {
	organizationBucket := ratelimiter.NewNonspecificBucket("111111111111")
	workspaceBucket := ratelimiter.NewWorkspaceBucket("222222222222")
	ingestionBucket := ratelimiter.NewIngestionBucket("222222222222")
	organization := &Organization{
		mu:         new(sync.Mutex),
		workspaces: map[string]*Workspace{},
		bucket:     organizationBucket,
		ID:         "111111111111",
	}
	workspace := &Workspace{
		mu:              new(sync.Mutex),
		organization:    organization,
		apiBucket:       workspaceBucket,
		ingestionBucket: ingestionBucket,
		ID:              "222222222222",
	}
	organization.workspaces[workspace.ID] = workspace
	state := &State{
		mu:            new(sync.Mutex),
		organizations: map[string]*Organization{organization.ID: organization},
		workspaces:    map[string]*Workspace{workspace.ID: workspace},
	}

	updatedOrganization := state.replaceOrganization(organization.ID, func(organization *Organization) {
		organization.Name = "updated"
	})
	updatedWorkspace := state.replaceWorkspace(workspace.ID, func(workspace *Workspace) {
		workspace.Name = "updated"
	})

	if updatedOrganization.bucket != organizationBucket {
		t.Fatal("organization update replaced its API rate-limit bucket")
	}
	if updatedWorkspace.apiBucket != workspaceBucket {
		t.Fatal("workspace update replaced its API rate-limit bucket")
	}
	if updatedWorkspace.ingestionBucket != ingestionBucket {
		t.Fatal("workspace update replaced its ingestion rate-limit bucket")
	}
}
