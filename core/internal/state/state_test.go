// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"database/sql/driver"
	"fmt"
	"slices"
	"testing"
)

type valuerStringer interface {
	driver.Valuer
	String() string
}

// TestNewConsentPurpose checks event location normalization and literal purpose codes.
func TestNewConsentPurpose(t *testing.T) {

	tests := []struct {
		name      string
		locations []EventConsentLocation
	}{
		{name: "nil event locations"},
		{name: "empty event locations", locations: []EventConsentLocation{}},
		{
			name: "literal purpose codes",
			locations: []EventConsentLocation{
				{PurposeCode: "vendor.marketing"},
				{PurposeCode: "#CFK567"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			purpose := NewConsentPurpose(ConsentPurpose{EventConsentLocations: test.locations})
			if purpose.EventConsentLocations == nil || !slices.Equal(purpose.EventConsentLocations, test.locations) {
				t.Fatalf("event locations = %#v, want non-nil with values %v", purpose.EventConsentLocations, test.locations)
			}
			paths := purpose.EventPropertyPaths()
			if paths == nil || len(paths) != len(test.locations) {
				t.Fatalf("event paths = %#v, want non-nil with %d paths", paths, len(test.locations))
			}
			for i, location := range test.locations {
				want := []string{"context", "consents", location.PurposeCode}
				if !slices.Equal(paths[i], want) {
					t.Fatalf("event path %d = %v, want %v", i, paths[i], want)
				}
			}
			if purpose.ProfileConsentLocation != nil || purpose.ProfilePropertyPath() != nil {
				t.Fatal("expected the absent profile location and its path to remain nil")
			}

		})
	}

}

// TestResolveRequiredConsents verifies that required purposes are resolved and
// missing purposes are rejected.
func TestResolveRequiredConsents(t *testing.T) {

	const (
		purposeID   = "111111111111"
		workspaceID = "222222222222"
	)
	purpose := &ConsentPurpose{ID: purposeID}
	workspace := &Workspace{
		ID:              workspaceID,
		consentPurposes: map[string]*ConsentPurpose{purposeID: purpose},
	}

	resolved, err := workspace.resolveRequiredConsents(RequiredConsentsByIDs{
		Purposes: []string{purposeID},
		Operator: PurposesOr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Purposes) != 1 || resolved.Purposes[0] != purpose {
		t.Fatalf("resolved purposes = %v, want [%p]", resolved.Purposes, purpose)
	}
	if resolved.Operator != PurposesOr {
		t.Fatalf("resolved operator = %v, want %v", resolved.Operator, PurposesOr)
	}

	_, err = workspace.resolveRequiredConsents(RequiredConsentsByIDs{Purposes: []string{"333333333333"}})
	if err != nil {
		const want = "required consent purpose 333333333333 does not exist in workspace 222222222222"
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
		return
	}
	t.Fatal("resolveRequiredConsents did not return an error for a missing purpose")

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
