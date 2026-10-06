// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"context"
	"maps"
	"reflect"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

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
// unchanged after the mutation has been applied, without advancing the version
// or acknowledging the notification.
func TestApplyPanicsDuringDispatch(t *testing.T) {
	ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: &Organization{ID: stateTestOrganizationID}, consentPurposes: map[string]*ConsentPurpose{}}
	state := &State{changing: &sync.RWMutex{}, workspaces: map[string]*Workspace{stateTestWorkspaceID: ws}}
	state.version.next = sync.Cond{L: &state.version.RWMutex}
	state.version.current = 7
	ack := make(chan struct{}, 1)
	state.notifications.acks.Store(8, ack)
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

		select {
		case <-ack:
			t.Fatal("expected no acknowledgement after failed dispatch, got acknowledgement")
		default:
		}
	}()
	_ = state.applyNotification(notification{8, "AddConsentPurpose", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":"Marketing"}`}, nil)
}

// TestApplyProcessesNotificationInOrder verifies that a notification is applied
// and dispatched before its version is published, the changing lock is released,
// and the notification is acknowledged.
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
		ack := make(chan struct{})
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
		}}
		state.notifications.acks.Store(8, ack)
		broadcast := make(chan struct{})
		go func() {
			state.version.Lock()
			state.version.next.Wait()
			state.version.Unlock()
			close(broadcast)
		}()
		synctest.Wait()

		state.close.Add(1)
		go func() {
			defer state.close.Done()
			err := state.applyNotification(notification{8, "AddConsentPurpose", `{"Workspace":"6NpT4zB8QaR2","ID":"D7hV4xK9mP2a","Name":"Marketing"}`}, nil)
			if err != nil {
				t.Errorf("expected applied notification, got %v", err)
			}
		}()
		// The unbuffered acknowledgement blocks application until we receive it.
		synctest.Wait()
		if calls != 1 {
			t.Errorf("expected notification to be dispatched before acknowledgement, got %d dispatches", calls)
		}
		if state.Version() != 8 {
			t.Errorf("expected version 8 before acknowledgement, got %d", state.Version())
		}
		select {
		case <-broadcast:
		default:
			t.Error("expected version Broadcast before acknowledgement, got no wakeup")
		}
		if !state.changing.TryLock() {
			t.Error("expected changing released before acknowledgement, got locked state")
		} else {
			state.changing.Unlock()
		}
		select {
		case <-ack:
		default:
			t.Fatal("expected notification acknowledgement, got none")
		}
		state.close.Wait()
		if calls != 1 || state.Version() != 8 {
			t.Fatalf("expected one dispatch and version 8, got %d and %d", calls, state.Version())
		}
		purpose := ws.consentPurposes[stateTestConsentPurposeID]
		if purpose == nil || purpose.Name != "Marketing" {
			t.Fatalf("expected Marketing consent purpose, got %#v", purpose)
		}
	})
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
// panic without application, dispatch, version advancement or acknowledgement.
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
				// Use a buffered ACK channel so the test can detect an unexpected acknowledgement without blocking application.
				ack := make(chan struct{}, 1)
				state.notifications.acks.Store(8, ack)
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
					if len(ack) != 0 {
						t.Fatalf("expected no acknowledgment, got %d", len(ack))
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
