// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestTransformDataMatching checks the Admin preview's matching argument,
// including validation failures and its compatibility with functions.
func TestTransformDataMatching(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	matching := types.Property{
		Name: "email_address", Type: types.String(), CreateRequired: true, UpdateRequired: true,
	}
	status := types.Property{Name: "status", Type: types.String(), CreateRequired: true}
	outSchema := types.Object([]types.Property{matching, status})
	tests := []struct {
		name           string
		matching       string
		purpose        string
		mapping        map[string]string
		outSchema      types.Type
		isFunction     bool
		expectedStatus int
		message        string
	}{
		{
			name: "create", matching: "email_address", purpose: "Create", expectedStatus: http.StatusOK,
		},
		{
			name: "update", matching: "email_address", purpose: "Update", expectedStatus: http.StatusOK,
		},
		{
			name: "missing argument preserves required checks", purpose: "Create",
			expectedStatus: http.StatusUnprocessableEntity,
			message:        "property «email_address» is missing but it is required for creation",
		},
		{
			name: "invalid path", matching: "email_address.", purpose: "Create",
			expectedStatus: http.StatusBadRequest, message: "output matching property path is not valid",
		},
		{
			name: "missing path", matching: "missing", purpose: "Create",
			expectedStatus: http.StatusBadRequest, message: "does not exist",
		},
		{
			name: "mapped matching", matching: "email_address", purpose: "Create",
			mapping:        map[string]string{"email_address": "source"},
			expectedStatus: http.StatusBadRequest, message: "mapping cannot map over the output matching property",
		},
		{
			name: "other required property remains checked", matching: "email_address", purpose: "Create",
			mapping:        map[string]string{"status": "null"},
			expectedStatus: http.StatusUnprocessableEntity,
			message:        "property «status» is missing but it is required for creation",
		},
		{
			name: "nested matching preserves output", matching: "parent.email_address", purpose: "Create",
			outSchema: types.Object([]types.Property{
				{Name: "parent", Type: types.Object([]types.Property{matching}), CreateRequired: true}, status,
			}),
			expectedStatus: http.StatusOK,
		},
		{
			name: "nested required sibling", matching: "parent.email_address", purpose: "Create",
			outSchema: types.Object([]types.Property{
				{Name: "parent", Type: types.Object([]types.Property{matching, status})},
			}),
			mapping:        map[string]string{"parent.status": "null"},
			expectedStatus: http.StatusUnprocessableEntity,
			message:        "property «parent.status» is missing but it is required for creation",
		},
		{
			name: "function ignores matching argument", matching: "not..a.path", purpose: "Create", isFunction: true,
			outSchema: types.Object([]types.Property{status}), expectedStatus: http.StatusOK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			schema := test.outSchema
			if !schema.Valid() {
				schema = outSchema
			}
			mapping := test.mapping
			if mapping == nil {
				mapping = map[string]string{"status": "source"}
			}
			transformation := map[string]any{"mapping": mapping}
			if test.isFunction {
				transformation = map[string]any{"function": map[string]any{
					"language": "Python", "source": "def transform(record):\n    return {'status': 'subscribed'}",
				}}
			}
			body := map[string]any{
				"data":           map[string]any{"source": "subscribed"},
				"inSchema":       types.Object([]types.Property{{Name: "source", Type: types.String()}}),
				"outSchema":      schema,
				"transformation": transformation,
				"purpose":        test.purpose,
			}
			if test.matching != "" {
				body["matchingOut"] = test.matching
			}

			var response struct {
				Data map[string]any `json:"data"`
			}
			err := k.TryCall("POST", "/v1/transformations", nil, body, &response)
			if err != nil {
				statusErr, ok := err.(*krenalistester.StatusCodeError)
				if !ok {
					t.Fatalf("expected HTTP response, got %T: %v", err, err)
				}
				if statusErr.Response.Code != test.expectedStatus {
					t.Fatalf("expected HTTP status %d, got %d: %s",
						test.expectedStatus, statusErr.Response.Code, statusErr.Response.Text)
				}
				if !strings.Contains(statusErr.Response.Text, test.message) {
					t.Fatalf("expected response containing %q, got %q", test.message, statusErr.Response.Text)
				}
				return
			}
			if test.expectedStatus != http.StatusOK {
				t.Fatalf("expected HTTP status %d, got %d", test.expectedStatus, http.StatusOK)
			}
			expected := map[string]any{"status": "subscribed"}
			if !reflect.DeepEqual(response.Data, expected) {
				t.Fatalf("expected output %#v, got %#v", expected, response.Data)
			}

		})
	}

}
