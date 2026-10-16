// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"context"
	"maps"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/state/ratelimiter"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
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

// TestApplyPanicsDuringDispatch verifies that a dispatch panic propagates
// unchanged after the mutation has been applied, without advancing the version.
func TestApplyPanicsDuringDispatch(t *testing.T) {
	ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: &Organization{ID: stateTestOrganizationID}, consentPurposes: map[string]*ConsentPurpose{}}
	state := &State{changing: &sync.RWMutex{}, workspaces: map[string]*Workspace{stateTestWorkspaceID: ws}}
	state.version.next = sync.Cond{L: &state.version.RWMutex}
	state.version.current = 7
	panicValue := &struct{ reason string }{"listener failure"}
	state.listeners = []any{func(AddConsentPurpose) { panic(panicValue) }}
	defer func() {
		if got := recover(); got != panicValue {
			t.Fatalf("expected original dispatch panic %v, got %v", panicValue, got)
		}

		purpose := ws.consentPurposes[stateTestConsentPurposeID]
		if purpose == nil {
			t.Fatal("expected consent purpose mutation to be applied, got nil")
		}

		if purpose.Name != "Marketing" {
			t.Fatalf("expected purpose name %q, got %q", "Marketing", purpose.Name)
		}

		if got := state.Version(); got != 7 {
			t.Fatalf("expected version to remain 7 after failed dispatch, got %d", got)
		}
	}()
	_ = state.applyNotification(notification{8, "AddConsentPurpose", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":"Marketing"}`}, nil)
}

// TestApplyProcessesNotificationInOrder verifies that a notification is applied
// and dispatched before its version is published and version waiters succeed.
func TestApplyProcessesNotificationInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		org := &Organization{ID: stateTestOrganizationID}
		ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: org, consentPurposes: map[string]*ConsentPurpose{}}
		state := &State{changing: &sync.RWMutex{}, workspaces: map[string]*Workspace{stateTestWorkspaceID: ws}}
		state.version.next = sync.Cond{L: &state.version.RWMutex}
		state.version.current = 7
		state.close.ctx, state.close.cancel = context.WithCancel(t.Context())
		defer state.close.cancel()
		calls := 0
		release := make(chan struct{})
		releaseDispatch := sync.OnceFunc(func() { close(release) })
		defer releaseDispatch()
		state.listeners = []any{func(e AddConsentPurpose) {
			calls++
			if got := state.Version(); got != 7 {
				t.Errorf("expected version 7 during dispatch, got %d", got)
			}
			if e.Name != "Marketing" {
				t.Errorf("expected Marketing during dispatch, got %q", e.Name)
			}
			if state.changing.TryLock() {
				state.changing.Unlock()
				t.Error("expected changing held during dispatch, got unlocked state")
			}
			purpose := ws.consentPurposes[e.ID]
			if purpose == nil || purpose.Name != e.Name {
				t.Errorf("expected applied consent purpose during dispatch, got %#v", purpose)
			}
			<-release
		}}
		completed := make(chan error, 1)
		go func() { completed <- state.waitVersion(t.Context(), 8) }()
		synctest.Wait()

		state.close.Go(func() {
			err := state.applyNotification(notification{8, "AddConsentPurpose", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":"Marketing"}`}, nil)
			if err != nil {
				t.Errorf("expected applied notification, got %v", err)
			}
		})
		synctest.Wait()
		if calls != 1 || state.Version() != 7 {
			t.Fatalf("expected pending dispatch and version 7, got calls=%d version=%d", calls, state.Version())
		}
		select {
		case err := <-completed:
			t.Fatalf("expected pending version wait during dispatch, got %v", err)
		default:
		}

		releaseDispatch()
		state.close.Wait()
		err := <-completed
		if err != nil {
			t.Fatalf("expected version wait after dispatch, got %v", err)
		}
		if !state.changing.TryLock() {
			t.Error("expected changing released after application, got locked state")
		} else {
			state.changing.Unlock()
		}
		if calls != 1 || state.Version() != 8 {
			t.Fatalf("expected one dispatch and version 8, got %d and %d", calls, state.Version())
		}
		purpose := ws.consentPurposes[stateTestConsentPurposeID]
		if purpose == nil || purpose.Name != "Marketing" {
			t.Fatalf("expected Marketing consent purpose, got %#v", purpose)
		}
	})
}

// TestElectLeaderIgnoresStaleElections verifies that stale and duplicate
// elections are ignored, while newer elections update the state and notify
// listeners only when the leader changes.
func TestElectLeaderIgnoresStaleElections(t *testing.T) {

	const currentNumber = math.MaxInt32 + 2
	for _, test := range []struct {
		name       string
		number     int
		leader     string
		apply      bool
		dispatches int
	}{
		{"old", currentNumber - 1, "previous", false, 0},
		{"duplicate", currentNumber, "current", false, 0},
		{"duplicate-different-leader", currentNumber, "other", false, 0},
		{"next", currentNumber + 1, "next", true, 1},
		{"missed-elections", currentNumber + 3, "next", true, 1},
		{"same-leader", currentNumber + 1, "current", true, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				previous := election{number: currentNumber, leader: "current", sequence: 10, validUntil: time.Unix(1, 0)}
				state := &State{mu: &sync.Mutex{}, changing: &sync.RWMutex{}, election: previous}
				state.version.current = 123
				calls := 0
				var received ElectLeader
				state.listeners = []any{func(e ElectLeader) {
					calls++
					received = e
				}}
				event := ElectLeader{Number: test.number, Leader: test.leader}
				payload, err := json.Marshal(event)
				if err != nil {
					t.Fatalf("expected encoded election, got %v", err)
				}

				now := time.Now()
				err = state.applyNotification(notification{Version: 0, Name: "ElectLeader", Payload: string(payload)}, nil)
				if err != nil {
					t.Fatalf("expected election handling, got %v", err)
				}
				if test.apply {
					if state.election.number != test.number || state.election.leader != test.leader {
						t.Fatalf("expected election %d with leader %q, got %+v", test.number, test.leader, state.election)
					}
					if expected := now.Add(grantedLeaderInterval); state.election.validUntil != expected {
						t.Fatalf("expected leader validity until %v, got %v", expected, state.election.validUntil)
					}
					if state.election.sequence != 0 {
						t.Fatalf("expected reset heartbeat sequence, got %d", state.election.sequence)
					}
				} else if state.election != previous {
					t.Fatalf("expected unchanged election %+v, got %+v", previous, state.election)
				}
				if calls != test.dispatches {
					t.Fatalf("expected %d listener calls, got %d", test.dispatches, calls)
				}
				if test.dispatches == 1 && received != event {
					t.Fatalf("expected listener event %+v, got %+v", event, received)
				}
				if got := state.Version(); got != 123 {
					t.Fatalf("expected state version 123, got %d", got)
				}

			})
		})
	}

}

// TestReplaceConsentPurpose checks that replacing a consent purpose does not
// modify the original.
func TestReplaceConsentPurpose(t *testing.T) {

	const purposeID = "111111111111"
	purpose := &ConsentPurpose{
		ID:                     purposeID,
		EventConsentLocations:  []EventConsentLocation{{PurposeCode: "marketing"}},
		ProfileConsentLocation: &ProfileConsentLocation{Property: "consents", JSONKey: "a.b"},
	}
	workspace := &Workspace{
		mu:              &sync.Mutex{},
		consentPurposes: map[string]*ConsentPurpose{purposeID: purpose},
	}

	updated := workspace.replaceConsentPurpose(purposeID, func(purpose *ConsentPurpose) {
		purpose.EventConsentLocations = []EventConsentLocation{}
		purpose.ProfileConsentLocation = nil
	})

	if updated.EventConsentLocations == nil {
		t.Fatal("expected non-nil event consent locations, got nil")
	}
	if len(updated.EventConsentLocations) != 0 {
		t.Fatalf("expected 0 event consent locations, got %d", len(updated.EventConsentLocations))
	}
	if updated.ProfileConsentLocation != nil {
		t.Fatalf("expected no profile consent location, got %#v", updated.ProfileConsentLocation)
	}
	if len(purpose.EventConsentLocations) != 1 {
		t.Fatalf("expected 1 original event consent location, got %d", len(purpose.EventConsentLocations))
	}
	if purpose.EventConsentLocations[0].PurposeCode != "marketing" {
		t.Fatalf("expected original purpose code %q, got %q", "marketing", purpose.EventConsentLocations[0].PurposeCode)
	}
	if purpose.ProfileConsentLocation == nil {
		t.Fatal("expected original profile consent location, got nil")
	}
	if purpose.ProfileConsentLocation.JSONKey != "a.b" {
		t.Fatalf("expected original JSON key %q, got %q", "a.b", purpose.ProfileConsentLocation.JSONKey)
	}

}

// TestReplaceOrganizationPreservesRateLimitBucket verifies that replacing an
// organization retains its local rate-limit bucket.
func TestReplaceOrganizationPreservesRateLimitBucket(t *testing.T) {
	const organizationID = "111111111111"
	bucket := new(ratelimiter.Limiter).NewBucket("test", organizationID, 1, 1)
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
	bucket := new(ratelimiter.Limiter).NewBucket("test", workspaceID, 1, 1)
	eventBucket := new(ratelimiter.Limiter).NewBucket("test-events", workspaceID, 1, 1)
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

// TestDecodeNotificationOmitsPayloadDetails verifies that schema decoding
// errors do not expose payload details in the panic message.
func TestDecodeNotificationOmitsPayloadDetails(t *testing.T) {
	defer func() {
		got := recover()
		const expected = "invalid notification payload UpdatePipeline (version 8)"
		if got != expected {
			t.Fatalf("expected panic %q, got %v", expected, got)
		}
	}()
	var event UpdatePipeline
	decodeNotification(notification{8, "UpdatePipeline", `{"InSchema":{"kind":"PrivateCredentialValue"}}`}, &event)
}

// TestApplyPanicsOnInvalidNotificationPayload verifies that invalid payloads
// panic without application, dispatch or version advancement.
func TestApplyPanicsOnInvalidNotificationPayload(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
	}{
		{"malformed", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":`},
		{"incompatible shape", `[]`},
		{"incompatible field", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":42}`},
		{"trailing JSON", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a"} {}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: &Organization{ID: stateTestOrganizationID}, consentPurposes: map[string]*ConsentPurpose{}}
				state := &State{changing: &sync.RWMutex{}, workspaces: map[string]*Workspace{stateTestWorkspaceID: ws}}
				state.version.next = sync.Cond{L: &state.version.RWMutex}
				state.version.current = 7
				listenerCalls := 0
				state.listeners = []any{func(AddConsentPurpose) { listenerCalls++ }}
				defer func() {
					got := recover()
					message, ok := got.(string)
					const expected = "invalid notification payload AddConsentPurpose (version 8)"
					if !ok || message != expected {
						t.Fatalf("expected panic %q, got %v", expected, got)
					}

					if len(ws.consentPurposes) != 0 {
						t.Fatalf("expected no state mutation, got %d consent purposes", len(ws.consentPurposes))
					}
					if listenerCalls != 0 {
						t.Fatalf("expected no listener dispatch, got %d calls", listenerCalls)
					}
					if got := state.Version(); got != 7 {
						t.Fatalf("expected version 7, got %d", got)
					}
				}()
				_ = state.applyNotification(notification{Version: 8, Name: "AddConsentPurpose", Payload: test.payload}, nil)
			})
		})
	}
}

// TestPipelineNotificationFilterDecoding verifies that create and update
// notifications decode valid filters and reject invalid ones without leaving
// state changes behind or dispatching the event.
func TestPipelineNotificationFilterDecoding(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, filter := range []struct {
			name      string
			payload   string
			expected  *Where
			wantPanic bool
		}{
			{"null", `null`, nil, false},
			{"empty-and", `{"operator":"And","rules":[]}`, &Where{Operator: OpAnd, Rules: []WhereRule{}}, false},
			{
				"condition", `{"operator":"And","rules":[{"property":["name"],"operator":"Is","values":["Ada"]}]}`,
				&Where{Operator: OpAnd, Rules: []WhereRule{
					&WhereCondition{Property: []string{"name"}, Operator: OpIs, Values: []any{"Ada"}},
				}},
				false,
			},
			{"unknown-property", `{"operator":"And","rules":[{"property":["PrivateCredentialValue"],"operator":"Is","values":["Ada"]}]}`, nil, true},
			{"legacy-shape", `{"logical":"And","conditions":[]}`, nil, true},
		} {
			t.Run(operation+"/"+filter.name, func(t *testing.T) {

				format := &Connector{Code: "format"}
				updatedFormat := &Connector{Code: "updated-format"}
				org := &Organization{mu: &sync.Mutex{}, ID: stateTestOrganizationID, usage: newOrganizationUsage(OrganizationLimits{})}
				connection := &Connection{mu: &sync.Mutex{}, ID: stateTestConnectionID, organization: org, pipelines: map[string]*Pipeline{}}
				state := &State{
					mu: &sync.Mutex{}, connections: map[string]*Connection{connection.ID: connection},
					pipelines: map[string]*Pipeline{}, connectors: map[string]*Connector{format.Code: format, updatedFormat.Code: updatedFormat},
				}
				createCalls, updateCalls := 0, 0
				state.listeners = []any{
					func(CreatePipeline) { createCalls++ },
					func(UpdatePipeline) { updateCalls++ },
				}
				schema := types.Object([]types.Property{{Name: "name", Type: types.String()}})
				var event any = CreatePipeline{
					ID: stateTestPipelineID, Connection: connection.ID, Name: "changed", InSchema: schema, Filter: []byte(filter.payload), Format: updatedFormat.Code,
				}
				name := "CreatePipeline"
				apply := state.createPipeline
				if operation == "update" {
					pipeline := &Pipeline{
						mu: &sync.Mutex{}, ID: stateTestPipelineID, Name: "original", connection: connection, organization: org,
						format: format, propertiesToUnset: []string{"original"}, Query: "original", FormatSettings: json.Value(`{"original":true}`),
						InSchema: types.Object([]types.Property{
							{Name: "name", Type: types.String()},
							{Name: "original", Type: types.String()},
						}),
						Filter: &Where{Operator: OpOr, Rules: []WhereRule{
							&WhereCondition{Property: []string{"name"}, Operator: OpIs, Values: []any{"original"}},
						}},
					}
					state.pipelines[pipeline.ID] = pipeline
					connection.pipelines[pipeline.ID] = pipeline
					org.usage.addPipeline(format)
					event = UpdatePipeline{ID: pipeline.ID, Name: "changed", InSchema: schema, Filter: []byte(filter.payload), Format: updatedFormat.Code}
					name = "UpdatePipeline"
					apply = state.updatePipeline
				}
				payload, err := json.Marshal(event)
				if err != nil {
					t.Fatalf("expected encoded event, got %v", err)
				}

				if filter.wantPanic {

					pipelines := maps.Clone(state.pipelines)
					connectionPipelines := maps.Clone(connection.pipelines)
					counts := org.usage.counts
					connectorUsage := maps.Clone(org.usage.connectorUsage)
					var originalPipeline []byte
					var originalFormat *Connector
					if pipeline := state.pipelines[stateTestPipelineID]; pipeline != nil {
						originalFormat = pipeline.format
						originalPipeline = pipelineUpdateSnapshot(t, pipeline)
					}
					defer func() {

						got := recover()
						expected := "invalid notification payload " + name + " (version 8)"
						if got != expected {
							t.Fatalf("expected panic %q, got %v", expected, got)
						}
						if !maps.Equal(pipelines, state.pipelines) || !maps.Equal(connectionPipelines, connection.pipelines) || counts != org.usage.counts || !maps.Equal(connectorUsage, org.usage.connectorUsage) || createCalls != 0 || updateCalls != 0 {
							t.Fatal("expected unchanged indexes, usage and listeners, got mutation or dispatch")
						}
						if pipeline := state.pipelines[stateTestPipelineID]; pipeline != nil {
							encodedPipeline := pipelineUpdateSnapshot(t, pipeline)
							if pipeline.format != originalFormat || !bytes.Equal(encodedPipeline, originalPipeline) {
								t.Fatal("expected unchanged pipeline fields, got mutation")
							}
						}

					}()
					apply(notification{8, name, string(payload)})
					t.Fatal("expected invalid filter panic, got normal return")

				}

				organization := apply(notification{8, name, string(payload)})
				pipeline := state.pipelines[stateTestPipelineID]
				if pipeline == nil {
					t.Fatal("expected pipeline to exist, got nil")
				}
				if organization != org.ID || pipeline.Name != "changed" || org.usage.counts.Pipelines != 1 {
					t.Fatalf("expected applied pipeline and count 1, got %q, %#v and %d", organization, pipeline, org.usage.counts.Pipelines)
				}
				if !types.Equal(pipeline.InSchema, schema) {
					t.Fatalf("expected input schema %v, got %v", schema, pipeline.InSchema)
				}
				if operation == "create" {
					if createCalls != 1 || updateCalls != 0 {
						t.Fatalf("expected one CreatePipeline dispatch, got %d create and %d update", createCalls, updateCalls)
					}
				} else {
					if createCalls != 0 || updateCalls != 1 {
						t.Fatalf("expected one UpdatePipeline dispatch, got %d create and %d update", createCalls, updateCalls)
					}
				}
				if !reflect.DeepEqual(filter.expected, pipeline.Filter) {
					t.Fatalf("expected filter %#v, got %#v", filter.expected, pipeline.Filter)
				}
				if pipeline.FormatSettings != nil {
					t.Fatalf("expected absent format settings, got %#v", pipeline.FormatSettings)
				}
				if connection.pipelines[pipeline.ID] != pipeline || pipeline.format != updatedFormat {
					t.Fatal("expected pipeline indexed by connection with updated format, got inconsistent references")
				}
				if !maps.Equal(org.usage.connectorUsage, map[*Connector]int{updatedFormat: 1}) {
					t.Fatalf("expected one use of updated format, got %#v", org.usage.connectorUsage)
				}

			})
		}
	}
}

// TestSeeLeader verifies heartbeat ordering, timestamp freshness, clock skew
// tolerance, and expiry handling, including monotonic clock preservation.
func TestSeeLeader(t *testing.T) {

	const (
		validity        = 3500 * time.Millisecond
		initialValidity = time.Second
		maxClockSkew    = 500 * time.Millisecond
	)
	zeroTimestamp := int64(0)
	minTimestamp := int64(math.MinInt64)
	maxTimestamp := int64(math.MaxInt64)
	for _, test := range []struct {
		name              string
		election          int
		sequence          uint64
		timestampOffset   time.Duration
		timestampOverride *int64
		expectedSequence  uint64
		remaining         time.Duration
	}{
		{"fresh", 7, 11, 0, nil, 11, validity},
		{"lost-message", 7, 13, 0, nil, 13, validity},
		{"maximum-sequence", 7, math.MaxUint64, 0, nil, math.MaxUint64, validity},
		{"previous-election", 6, 11, 0, nil, 10, initialValidity},
		{"future-election", 8, 11, 0, nil, 10, initialValidity},
		{"duplicate", 7, 10, 0, nil, 10, initialValidity},
		{"out-of-order", 7, 9, 0, nil, 10, initialValidity},
		{"zero-sequence", 7, 0, 0, nil, 10, initialValidity},
		{"legacy", 7, 0, 0, &zeroTimestamp, 10, initialValidity},
		{"missing-timestamp", 7, 11, 0, &zeroTimestamp, 10, initialValidity},
		{"oldest-timestamp", 7, 11, 0, &minTimestamp, 10, initialValidity},
		{"latest-timestamp", 7, 11, 0, &maxTimestamp, 10, initialValidity},
		{"delayed", 7, 11, -time.Second, nil, 11, validity - time.Second},
		{"preserve-later-expiry", 7, 11, -3 * time.Second, nil, 11, initialValidity},
		{"almost-expired", 7, 11, -validity + time.Millisecond, nil, 11, initialValidity},
		{"expired", 7, 11, -validity, nil, 10, initialValidity},
		{"too-old", 7, 11, -4 * time.Second, nil, 10, initialValidity},
		{"clock-skew-limit", 7, 11, maxClockSkew, nil, 11, validity + maxClockSkew},
		{"too-far-in-future", 7, 11, maxClockSkew + time.Millisecond, nil, 10, initialValidity},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				now := time.Now()
				timestamp := now.Add(test.timestampOffset).UnixMilli()
				if test.timestampOverride != nil {
					timestamp = *test.timestampOverride
				}
				event := SeeLeader{Election: test.election, Sequence: test.sequence, Timestamp: timestamp}
				previous := election{number: 7, leader: "leader", sequence: 10, validUntil: now.Add(initialValidity)}
				state := &State{mu: &sync.Mutex{}, election: previous}
				payload, err := json.Marshal(event)
				if err != nil {
					t.Fatalf("expected encoded heartbeat, got %v", err)
				}

				state.seeLeader(notification{0, "SeeLeader", string(payload)})
				expected := previous
				expected.sequence = test.expectedSequence
				expected.validUntil = now.Add(test.remaining)
				if state.election != expected {
					t.Errorf("expected election %+v, got %+v", expected, state.election)
				}

			})
		})
	}

	// synctest omits monotonic readings, so check them outside the bubble.
	t.Run("monotonic-expiry", func(t *testing.T) {
		state := &State{mu: &sync.Mutex{}, election: election{number: 7, leader: "leader"}}
		payload, err := json.Marshal(SeeLeader{Election: 7, Sequence: 1, Timestamp: time.Now().UnixMilli()})
		if err != nil {
			t.Fatalf("expected encoded heartbeat, got %v", err)
		}
		state.seeLeader(notification{0, "SeeLeader", string(payload)})
		if state.election.sequence != 1 {
			t.Fatalf("expected accepted heartbeat with sequence 1, got %+v", state.election)
		}
		validUntil := state.election.validUntil
		if validUntil == validUntil.Round(0) {
			t.Fatalf("expected expiry with a monotonic clock reading, got %v", validUntil)
		}
	})

}

// TestSeeLeaderAfterFollowerRestart verifies that replays after a follower
// restart use the original heartbeat expiry and preserve the startup grace.
func TestSeeLeaderAfterFollowerRestart(t *testing.T) {

	const (
		startupGrace      = 3 * time.Second
		heartbeatValidity = startupGrace + 500*time.Millisecond
	)
	for _, test := range []struct {
		name             string
		delay            time.Duration
		expectedSequence uint64
		remaining        time.Duration
	}{
		{"fresh-replay", 100 * time.Millisecond, 10, heartbeatValidity - 100*time.Millisecond},
		{"preserve-startup-grace", time.Second, 10, startupGrace},
		{"expired-replay", 4 * time.Second, 0, startupGrace},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				sent := time.Now()
				payload, err := json.Marshal(SeeLeader{Election: 7, Sequence: 10, Timestamp: sent.UnixMilli()})
				if err != nil {
					t.Fatalf("expected encoded heartbeat, got %v", err)
				}
				n := notification{0, "SeeLeader", string(payload)}
				state := &State{mu: &sync.Mutex{}, election: election{number: 7, leader: "leader"}}
				state.seeLeader(n)
				if state.election.sequence != 10 {
					t.Fatalf("expected original heartbeat accepted, got %+v", state.election)
				}

				time.Sleep(test.delay)
				// Simulate a restart that preserves the election but loses the received sequence.
				restartedAt := time.Now()
				initial := election{number: 7, leader: "leader", validUntil: restartedAt.Add(grantedLeaderInterval)}
				state = &State{mu: &sync.Mutex{}, election: initial}
				state.seeLeader(n)
				expected := initial
				expected.sequence = test.expectedSequence
				expected.validUntil = restartedAt.Add(test.remaining)
				if state.election != expected {
					t.Fatalf("expected restarted election %+v, got %+v", expected, state.election)
				}

				// Replay again without restarting the follower.
				time.Sleep(time.Second)
				state.seeLeader(n)
				if state.election != expected {
					t.Fatalf("expected replay to leave election %+v unchanged, got %+v", expected, state.election)
				}

			})
		})
	}

}

// TestSeeLeaderAfterLeaderRestart verifies that a follower rejects heartbeats
// from a new election until it is applied, then accepts the reset sequence
// and continues to reject heartbeats from the previous election.
func TestSeeLeaderAfterLeaderRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		now := time.Now()
		previous := election{number: 7, leader: "previous", sequence: 10, validUntil: now.Add(time.Second)}
		state := &State{mu: &sync.Mutex{}, election: previous}

		// Both heartbeats must be rejected, even though 11 exceeds the last received sequence.
		for _, sequence := range []uint64{1, 11} {
			payload, err := json.Marshal(SeeLeader{Election: 8, Sequence: sequence, Timestamp: now.UnixMilli()})
			if err != nil {
				t.Fatalf("expected encoded heartbeat, got %v", err)
			}
			state.seeLeader(notification{0, "SeeLeader", string(payload)})
			if state.election != previous {
				t.Fatalf("expected future election heartbeat with sequence %d ignored, got %+v", sequence, state.election)
			}
		}

		// Apply the new election with the node ID assigned after the leader's restart.
		state.electLeader(notification{0, "ElectLeader", `{"Number":8,"Leader":"restarted"}`})
		expected := election{number: 8, leader: "restarted", sequence: 0, validUntil: now.Add(grantedLeaderInterval)}
		if state.election != expected {
			t.Fatalf("expected new election with reset sequence %+v, got %+v", expected, state.election)
		}

		payload, err := json.Marshal(SeeLeader{Election: 8, Sequence: 1, Timestamp: now.UnixMilli()})
		if err != nil {
			t.Fatalf("expected encoded heartbeat, got %v", err)
		}
		state.seeLeader(notification{0, "SeeLeader", string(payload)})
		expected.sequence = 1
		expected.validUntil = now.Add(grantedLeaderInterval + leaderClockSkew)
		if state.election != expected {
			t.Fatalf("expected new leader heartbeat accepted as %+v, got %+v", expected, state.election)
		}

		payload, err = json.Marshal(SeeLeader{Election: 7, Sequence: 11, Timestamp: now.UnixMilli()})
		if err != nil {
			t.Fatalf("expected encoded previous leader heartbeat, got %v", err)
		}
		state.seeLeader(notification{0, "SeeLeader", string(payload)})
		if state.election != expected {
			t.Fatalf("expected previous leader heartbeat ignored, got %+v", state.election)
		}

	})
}

// pipelineUpdateSnapshot returns a JSON snapshot of the pipeline fields that
// updatePipeline may modify. The format reference is compared separately.
func pipelineUpdateSnapshot(t *testing.T, pipeline *Pipeline) []byte {

	t.Helper()

	snapshot, err := json.Marshal(struct {
		Name               string
		Enabled            bool
		InSchema           types.Type
		OutSchema          types.Type
		Filter             *Where
		RequiredConsents   RequiredConsents
		Transformation     Transformation
		Query              string
		Path               string
		Sheet              string
		Compression        Compression
		OrderBy            string
		FormatSettings     json.Value
		ExportMode         ExportMode
		Matching           Matching
		UpdateOnDuplicates bool
		TableName          string
		TableKey           string
		UserIDColumn       string
		UpdatedAtColumn    string
		UpdatedAtFormat    string
		Incremental        bool
		PropertiesToUnset  []string
	}{
		Name:               pipeline.Name,
		Enabled:            pipeline.Enabled,
		InSchema:           pipeline.InSchema,
		OutSchema:          pipeline.OutSchema,
		Filter:             pipeline.Filter,
		RequiredConsents:   pipeline.RequiredConsents,
		Transformation:     pipeline.Transformation,
		Query:              pipeline.Query,
		Path:               pipeline.Path,
		Sheet:              pipeline.Sheet,
		Compression:        pipeline.Compression,
		OrderBy:            pipeline.OrderBy,
		FormatSettings:     pipeline.FormatSettings,
		ExportMode:         pipeline.ExportMode,
		Matching:           pipeline.Matching,
		UpdateOnDuplicates: pipeline.UpdateOnDuplicates,
		TableName:          pipeline.TableName,
		TableKey:           pipeline.TableKey,
		UserIDColumn:       pipeline.UserIDColumn,
		UpdatedAtColumn:    pipeline.UpdatedAtColumn,
		UpdatedAtFormat:    pipeline.UpdatedAtFormat,
		Incremental:        pipeline.Incremental,
		PropertiesToUnset:  pipeline.propertiesToUnset,
	})
	if err != nil {
		t.Fatalf("expected encoded pipeline snapshot, got %v", err)
	}

	return snapshot
}
