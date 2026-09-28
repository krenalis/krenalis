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

// TestConsentPurposeLocationsAPI checks canonical responses and replacement
// semantics for consent locations.
func TestConsentPurposeLocationsAPI(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	tests := []struct {
		name      string
		locations string
		want      string
	}{
		{
			name: "omitted locations",
			want: `"eventConsentLocations":[],"profileConsentLocation":null`,
		},
		{
			name:      "null locations",
			locations: `,"eventConsentLocations":null,"profileConsentLocation":null`,
			want:      `"eventConsentLocations":[],"profileConsentLocation":null`,
		},
		{
			name:      "empty event locations",
			locations: `,"eventConsentLocations":[]`,
			want:      `"eventConsentLocations":[],"profileConsentLocation":null`,
		},
		{
			name:      "omitted JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing"}`,
			want:      `"eventConsentLocations":[],"profileConsentLocation":{"property":"marketing"}`,
		},
		{
			name:      "null JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":null}`,
			want:      `"eventConsentLocations":[],"profileConsentLocation":{"property":"marketing"}`,
		},
		{
			name:      "empty JSON key",
			locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":""}`,
			want:      `"eventConsentLocations":[],"profileConsentLocation":{"property":"marketing"}`,
		},
		{
			name: "literal keys and event order",
			locations: `,"eventConsentLocations":[{"purposeCode":"a.b"},{"purposeCode":"A.B"}],` +
				`"profileConsentLocation":{"property":"consents","jsonKey":" purpose.\"code\"\\u0061 "}`,
			want: `"eventConsentLocations":[{"purposeCode":"a.b"},{"purposeCode":"A.B"}],` +
				`"profileConsentLocation":{"property":"consents","jsonKey":" purpose.\"code\"\\u0061 "}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			k.Call("POST", "v1/consent-purposes", nil, json.Value(`{"name":"Marketing"`+test.locations+`}`), nil)
			var response struct {
				Purposes []map[string]any `json:"purposes"`
			}
			k.Call("GET", "v1/consent-purposes", nil, nil, &response)
			if len(response.Purposes) != 1 {
				t.Fatalf("expected one purpose, got %v", response.Purposes)
			}
			got := response.Purposes[0]
			id, ok := got["id"].(string)
			if !ok || id == "" {
				t.Fatalf("expected a purpose ID, got %v", got["id"])
			}
			delete(got, "id")
			var want map[string]any
			err := json.Unmarshal([]byte(`{"name":"Marketing",`+test.want+`}`), &want)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("purpose = %v, want %v", got, want)
			}

			k.Call("PUT", "v1/consent-purposes/"+id, nil, json.Value(`{"name":"Marketing"`+test.locations+`}`), nil)
			k.Call("GET", "v1/consent-purposes", nil, nil, &response)
			want["id"] = id
			if len(response.Purposes) != 1 || !reflect.DeepEqual(response.Purposes[0], want) {
				t.Fatalf("unchanged purposes = %v, want %v", response.Purposes, want)
			}

			replacement := map[string]any{
				"name": "Marketing",
				"eventConsentLocations": []any{
					map[string]any{"purposeCode": "updated"},
					map[string]any{"purposeCode": "#CFK567"},
				},
				"profileConsentLocation": map[string]any{"property": "consents", "jsonKey": "updated.key"},
			}
			k.Call("PUT", "v1/consent-purposes/"+id, nil, replacement, nil)
			k.Call("GET", "v1/consent-purposes", nil, nil, &response)
			replacement["id"] = id
			if len(response.Purposes) != 1 || !reflect.DeepEqual(response.Purposes[0], replacement) {
				t.Fatalf("updated purposes = %v, want %v", response.Purposes, replacement)
			}

			k.Call("PUT", "v1/consent-purposes/"+id, nil, map[string]any{"name": "Marketing"}, nil)
			k.Call("GET", "v1/consent-purposes", nil, nil, &response)
			want["eventConsentLocations"] = []any{}
			want["profileConsentLocation"] = nil
			if len(response.Purposes) != 1 || !reflect.DeepEqual(response.Purposes[0], want) {
				t.Fatalf("replaced purposes = %v, want %v", response.Purposes, want)
			}
			k.Call("DELETE", "v1/consent-purposes/"+id, nil, nil, nil)

		})
	}

}

// TestConsentPurposeLocationsRequiredByPipelines checks that a pipeline can
// require only the consent purposes with a consent location for its target, and
// that such a location cannot be removed while a pipeline requires it.
func TestConsentPurposeLocationsRequiredByPipelines(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	eventLocations := []any{map[string]any{"purposeCode": "marketing"}}
	profileLocation := map[string]any{"property": "marketing"}

	k.Call("POST", "v1/consent-purposes", nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, nil)
	var response struct {
		Purposes []struct {
			ID string `json:"id"`
		} `json:"purposes"`
	}
	k.Call("GET", "v1/consent-purposes", nil, nil, &response)
	if len(response.Purposes) != 1 {
		t.Fatalf("expected one purpose, got %v", response.Purposes)
	}
	purpose := response.Purposes[0].ID
	requiredConsents := &krenalistester.RequiredConsents{Operator: "and", Purposes: []string{purpose}}

	// An event pipeline can require a purpose with an event consent location.
	javaScript := k.CreateJavaScriptSource("JavaScript", nil)
	k.CreatePipeline(javaScript, "Event", krenalistester.PipelineToSet{
		Name:             "Import events",
		Enabled:          true,
		RequiredConsents: requiredConsents,
	})

	// A profile pipeline cannot require a purpose without a profile consent
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
	err = k.TryCall("PUT", "v1/consent-purposes/"+purpose, nil, map[string]any{
		"name":                   "Marketing",
		"profileConsentLocation": profileLocation,
	}, nil)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationInUse))

	// Once the purpose has a profile consent location, a profile pipeline can
	// require it, and that location cannot be removed.
	k.Call("PUT", "v1/consent-purposes/"+purpose, nil, map[string]any{
		"name":                   "Marketing",
		"eventConsentLocations":  eventLocations,
		"profileConsentLocation": profileLocation,
	}, nil)
	userPipelineID := k.CreatePipeline(dummy, "User", userPipeline)
	err = k.TryCall("PUT", "v1/consent-purposes/"+purpose, nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, nil)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationInUse))

	// Once the profile pipeline no longer requires the purpose, the profile
	// consent location can be removed, and the pipeline cannot require the
	// purpose again.
	userPipeline.RequiredConsents = nil
	k.UpdatePipeline(userPipelineID, userPipeline)
	k.Call("PUT", "v1/consent-purposes/"+purpose, nil, map[string]any{
		"name":                  "Marketing",
		"eventConsentLocations": eventLocations,
	}, nil)
	userPipeline.RequiredConsents = requiredConsents
	err = k.TryUpdatePipeline(userPipelineID, userPipeline)
	expectAPIError(t, err, http.StatusUnprocessableEntity, string(core.ConsentPurposeLocationNotSet))

}
