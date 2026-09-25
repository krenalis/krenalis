// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
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
