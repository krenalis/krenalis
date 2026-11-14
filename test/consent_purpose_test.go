// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestConsentPurposeLocationsCanonicalization checks that consent locations
// are returned in canonical form.
func TestConsentPurposeLocationsCanonicalization(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	tests := []struct {
		name      string
		locations string
		want      core.ConsentPurposeToSet
	}{
		{
			name: "omitted locations",
			want: core.ConsentPurposeToSet{Name: "Marketing", EventConsentLocations: []core.EventConsentLocation{}},
		},
		{
			name:      "null locations",
			locations: `,"eventConsentLocations":null,"profileConsentLocation":null`,
			want:      core.ConsentPurposeToSet{Name: "Marketing", EventConsentLocations: []core.EventConsentLocation{}},
		},
		{
			name:      "empty event locations",
			locations: `,"eventConsentLocations":[]`,
			want:      core.ConsentPurposeToSet{Name: "Marketing", EventConsentLocations: []core.EventConsentLocation{}},
		},
		{
			name:      "omitted JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing"}`,
			want: core.ConsentPurposeToSet{
				Name:                  "Marketing",
				EventConsentLocations: []core.EventConsentLocation{},
				ProfileConsentLocation: &core.ProfileConsentLocation{
					Property: "marketing",
				},
			},
		},
		{
			name:      "null JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":null}`,
			want: core.ConsentPurposeToSet{
				Name:                  "Marketing",
				EventConsentLocations: []core.EventConsentLocation{},
				ProfileConsentLocation: &core.ProfileConsentLocation{
					Property: "marketing",
				},
			},
		},
		{
			name:      "empty JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":""}`,
			want: core.ConsentPurposeToSet{
				Name:                  "Marketing",
				EventConsentLocations: []core.EventConsentLocation{},
				ProfileConsentLocation: &core.ProfileConsentLocation{
					Property: "marketing",
				},
			},
		},
		{
			name: "literal keys and event order",
			locations: `,"eventConsentLocations":[{"purposeCode":"a.b"},{"purposeCode":"A.B"}],` +
				`"profileConsentLocation":{"property":"consents","jsonKey":" purpose.\"code\"\\u0061 "}`,
			want: core.ConsentPurposeToSet{
				Name: "Marketing",
				EventConsentLocations: []core.EventConsentLocation{
					{PurposeCode: "a.b"},
					{PurposeCode: "A.B"},
				},
				ProfileConsentLocation: &core.ProfileConsentLocation{
					Property: "consents",
					JSONKey:  ` purpose."code"\u0061 `,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			body := json.Value(`{"name":"Marketing"` + test.locations + `}`)
			var response struct {
				ID string `json:"id"`
			}
			k.Call("POST", "v1/consent-purposes", nil, body, &response)
			id := response.ID
			if id == "" {
				t.Fatalf("expected a purpose ID, got %q", id)
			}
			got := k.ConsentPurpose(id)

			want := core.ConsentPurpose{
				ID:                     id,
				Name:                   test.want.Name,
				EventConsentLocations:  test.want.EventConsentLocations,
				ProfileConsentLocation: test.want.ProfileConsentLocation,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected purpose %v, got %v", want, got)
			}

			k.Call("PUT", "v1/consent-purposes/"+id, nil, body, nil)
			got = k.ConsentPurpose(id)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected purpose after unchanged update %v, got %v", want, got)
			}
			k.Call("DELETE", "v1/consent-purposes/"+id, nil, nil, nil)

		})
	}

}

// TestConsentPurposeLocationsReplacement checks that PUT fully replaces
// consent locations rather than preserving existing values.
func TestConsentPurposeLocationsReplacement(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	initial := core.ConsentPurposeToSet{
		Name: "Marketing",
		EventConsentLocations: []core.EventConsentLocation{
			{PurposeCode: "initial"},
		},
		ProfileConsentLocation: &core.ProfileConsentLocation{
			Property: "initial",
			JSONKey:  "initial.key",
		},
	}
	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "v1/consent-purposes", nil, initial, &response)
	id := response.ID
	if id == "" {
		t.Fatalf("expected a purpose ID, got %q", id)
	}
	initialGot := k.ConsentPurpose(id)
	initialWant := core.ConsentPurpose{
		ID:                     id,
		Name:                   initial.Name,
		EventConsentLocations:  initial.EventConsentLocations,
		ProfileConsentLocation: initial.ProfileConsentLocation,
	}
	if !reflect.DeepEqual(initialGot, initialWant) {
		t.Fatalf("expected initial purpose %v, got %v", initialWant, initialGot)
	}

	update := core.ConsentPurposeToSet{
		Name: "Marketing",
		EventConsentLocations: []core.EventConsentLocation{
			{PurposeCode: "updated"},
			{PurposeCode: "#CFK567"},
		},
		ProfileConsentLocation: &core.ProfileConsentLocation{
			Property: "consents",
			JSONKey:  "updated.key",
		},
	}
	k.Call("PUT", "v1/consent-purposes/"+id, nil, update, nil)
	updated := core.ConsentPurpose{
		ID:                     id,
		Name:                   update.Name,
		EventConsentLocations:  update.EventConsentLocations,
		ProfileConsentLocation: update.ProfileConsentLocation,
	}
	got := k.ConsentPurpose(id)
	if !reflect.DeepEqual(got, updated) {
		t.Fatalf("expected purpose after replacement %v, got %v", updated, got)
	}

	k.Call("PUT", "v1/consent-purposes/"+id, nil, map[string]any{"name": "Marketing"}, nil)
	want := core.ConsentPurpose{
		ID:                    id,
		Name:                  "Marketing",
		EventConsentLocations: []core.EventConsentLocation{},
	}
	got = k.ConsentPurpose(id)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected purpose after clearing locations %v, got %v", want, got)
	}

	k.Call("DELETE", "v1/consent-purposes/"+id, nil, nil, nil)

}

// TestPipelineRequiredConsentPurposeLocations checks that a pipeline can
// require a purpose only when that purpose has a location for the pipeline
// target, and that the location cannot be removed while a pipeline requires
// the purpose.
func TestPipelineRequiredConsentPurposeLocations(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	eventLocations := []any{map[string]any{"purposeCode": "marketing"}}
	profileLocation := map[string]any{"property": "marketing"}

	var response struct {
		ID string `json:"id"`
	}
	k.Call("POST", "v1/consent-purposes", nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, &response)
	purposeID := response.ID
	requiredConsents := &krenalistester.RequiredConsents{Operator: "and", Purposes: []string{purposeID}}

	// An event pipeline can require a purpose with an event consent location.
	javaScript := k.CreateJavaScriptSource("JavaScript", nil)
	k.CreatePipeline(javaScript, "Event", krenalistester.PipelineToSet{
		Name:             "Import events",
		Enabled:          true,
		RequiredConsents: requiredConsents,
	})

	// A user pipeline cannot require a purpose without a profile consent
	// location.
	dummy := k.CreateDummy("Dummy", krenalistester.Source)
	userPipeline := krenalistester.PipelineToSet{
		Name:    "Import users",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{"email": "email"},
		},
		RequiredConsents: requiredConsents,
	}
	_, err := k.TryCreatePipeline(dummy, "User", userPipeline)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationNotSet))

	// The event consent location cannot be removed while the event pipeline
	// requires the purpose.
	err = k.TryCall("PUT", "v1/consent-purposes/"+purposeID, nil, map[string]any{
		"name":                   "Marketing",
		"profileConsentLocation": profileLocation,
	}, nil)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationInUse))

	// Once the purpose has a profile consent location, a user pipeline can
	// require it, and that location cannot be removed.
	k.Call("PUT", "v1/consent-purposes/"+purposeID, nil, map[string]any{
		"name":                   "Marketing",
		"eventConsentLocations":  eventLocations,
		"profileConsentLocation": profileLocation,
	}, nil)
	userPipelineID := k.CreatePipeline(dummy, "User", userPipeline)
	err = k.TryCall("PUT", "v1/consent-purposes/"+purposeID, nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, nil)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationInUse))

	// Once the user pipeline no longer requires the purpose, the profile
	// consent location can be removed. The pipeline then cannot require the
	// purpose again.
	userPipeline.RequiredConsents = nil
	k.UpdatePipeline(userPipelineID, userPipeline)
	k.Call("PUT", "v1/consent-purposes/"+purposeID, nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, nil)
	userPipeline.RequiredConsents = requiredConsents
	err = k.TryUpdatePipeline(userPipelineID, userPipeline)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationNotSet))

}
