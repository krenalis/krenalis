// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"
	"time"

	"github.com/meergo/analytics-go"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestAnonymousNotAnonymous(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Create a JavaScript connection and get its key.
	var javaScriptKey string
	javaScriptID := c.CreateJavaScriptSource("JavaScript (source)", nil)
	keys := c.EventWriteKeys(javaScriptID)
	if len(keys) != 1 {
		t.Fatalf("expected one key, got %d keys", len(keys))
	}
	javaScriptKey = keys[0]

	// Create a first action, with a filter.
	action1 := c.CreateAction(javaScriptID, "User", meergotester.ActionToSet{
		Name:     "Action 1",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Filter: &meergotester.Filter{
			Logical: "or",
			Conditions: []meergotester.FilterCondition{
				{Property: "messageId", Operator: "is", Values: []string{"message1"}}, // message of the anonymous identity
				{Property: "messageId", Operator: "is", Values: []string{"message3"}}, // message of the not-anonymous identity
			},
		},
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "traits.email",
			},
		},
	})

	// Create a second action, which imports identities from events with a
	// different filter than the first action.
	action2 := c.CreateAction(javaScriptID, "User", meergotester.ActionToSet{
		Name:     "Action 2",
		Enabled:  true,
		InSchema: types.Type{},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Filter: &meergotester.Filter{
			Logical: "or",
			Conditions: []meergotester.FilterCondition{
				{Property: "messageId", Operator: "is", Values: []string{"message2"}}, // message of the anonymous identity
			},
		},
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "traits.email",
			},
		},
	})

	// Import two anonymous identities; each will need to be imported from its
	// own action.
	c.SendEvent(javaScriptKey, analytics.Identify{
		AnonymousId: "f3421606-a5a4-4027-bc81-50aedae5ccf3",
		MessageId:   "message1",
		Traits:      analytics.NewTraits().SetEmail("a@example.com"),
	})

	c.SendEvent(javaScriptKey, analytics.Identify{
		AnonymousId: "f3421606-a5a4-4027-bc81-50aedae5ccf3",
		MessageId:   "message2",
		Traits:      analytics.NewTraits().SetEmail("a@example.com"),
	})

	c.RunIdentityResolution()

	// Wait for the 2 identities to be imported successfully.
	attempts := 0
	var identities []meergotester.UserIdentity
	for {
		var total int
		identities, total = c.ConnectionIdentities(javaScriptID, 0, 100)
		if total == 2 {
			break
		}
		attempts += 1
		if attempts > 10 {
			t.Fatal("too many failed attempts waiting for the identities to be written on the data warehouse")
		}
		time.Sleep(500 * time.Millisecond)
	}

	var action1Found, action2Found bool
	for _, identity := range identities {
		if identity.ID != "" {
			t.Fatalf("expected no identity ID, got %v", identity.ID)
		}
		switch identity.Action {
		case action1:
			action1Found = true
		case action2:
			action2Found = true
		default:
			t.Fatalf("unexpected identity with action %d", identity.Action)
		}
	}
	if !action1Found {
		t.Fatal("identity for action 1 not found")
	}
	if !action2Found {
		t.Fatal("identity for action 2 not found")
	}

	// Log in the user of the first action.
	c.SendEvent(javaScriptKey, analytics.Identify{
		UserId:      "user-id-1234",
		AnonymousId: "f3421606-a5a4-4027-bc81-50aedae5ccf3",
		MessageId:   "message3",
		Traits:      analytics.NewTraits().SetAge(20),
	})

	c.RunIdentityResolution()

	attempts = 0
waitLoop:
	for {
		identities, _ := c.ConnectionIdentities(javaScriptID, 0, 100)
		for _, identity := range identities {
			if identity.ID != "" {
				break waitLoop
			}
		}
		attempts += 1
		if attempts > 10 {
			t.Fatal("too many failed attempts waiting for the not-anonymous identity to be written on the data warehouse")
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Make sure there is only one identity now, as both the anonymous
	// identities, each imported by its own action, have been deleted.
	identities, total := c.ConnectionIdentities(javaScriptID, 0, 100)
	if total != 1 {
		t.Fatalf("expected just one identity, got %d", total)
	}

	// Check that the only existing identity is correct.
	identity := identities[0]
	if identity.Action != action1 {
		t.Fatalf("identity should have action %d, got %d instead", action1, identity.Action)
	}
	if len(identity.AnonymousIds) != 1 {
		t.Fatalf("action should have just one anonymous ID, got %d instead", len(identity.AnonymousIds))
	}
	anonID := identity.AnonymousIds[0]
	if anonID != "f3421606-a5a4-4027-bc81-50aedae5ccf3" {
		t.Fatalf("unexpected anonymous ID %q", anonID)
	}

	// Run the Identity Resolution explicitly (even though technically it should
	// have already been done implicitly during import).
	c.RunIdentityResolution()

	// Check that there is actually only one user in the workspace.
	_, _, total = c.Users([]string{"email"}, "", false, 0, 100)
	if total != 1 {
		t.Fatalf("expected only one user in the workspace, got %d instead", total)
	}

}
