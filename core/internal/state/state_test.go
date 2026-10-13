// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"database/sql/driver"
	"fmt"
	"testing"
)

func TestTransformationEqual(t *testing.T) {
	fn1 := &TransformationFunction{ID: "f1", Version: "v1"}
	fn1copy := &TransformationFunction{ID: "f1", Version: "v1"}
	fn2 := &TransformationFunction{ID: "f2", Version: "v1"}

	tests := []struct {
		name  string
		t1    Transformation
		t2    Transformation
		equal bool
	}{
		{
			name:  "nil functions equal mapping",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			t2:    Transformation{Mapping: map[string]string{"a": "b"}, InPaths: []string{"b"}, OutPaths: []string{"a"}},
			equal: true,
		},
		{
			name:  "mapping differs when functions nil",
			t1:    Transformation{Mapping: map[string]string{"a": "b"}},
			t2:    Transformation{Mapping: map[string]string{"a": "c"}},
			equal: false,
		},
		{
			name:  "functions equals",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn1copy},
			equal: true,
		},
		{
			name:  "functions differ",
			t1:    Transformation{Function: fn1},
			t2:    Transformation{Function: fn2},
			equal: false,
		},
		{
			name:  "in paths differ",
			t1:    Transformation{Function: fn1, InPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "out paths differ",
			t1:    Transformation{Function: fn1, OutPaths: []string{"a"}},
			t2:    Transformation{Function: fn1copy, OutPaths: []string{"b"}},
			equal: false,
		},
		{
			name:  "everything equal",
			t1:    Transformation{Function: fn1, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			t2:    Transformation{Function: fn1copy, InPaths: []string{"in"}, OutPaths: []string{"out"}},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.t1.Equal(tt.t2) != tt.equal {
				t.Fatalf("expected %v", tt.equal)
			}
			if tt.t2.Equal(tt.t1) != tt.equal {
				t.Fatalf("expected symmetry %v", tt.equal)
			}
		})
	}
}

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
