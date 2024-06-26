//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/types"
)

func Test_validateActionToSet(t *testing.T) {

	tests := []struct {
		name string // test name, in the form: "Role/Connection Type/Target - any additional information"

		// The ActionToSet to validate.
		action ActionToSet

		// The validation state.
		target state.Target

		connectionRole          state.Role
		connectionConnectorType state.ConnectorType

		connectorType      state.ConnectorType
		connectorHasUI     bool
		connectorHasSheets bool

		provider transformers.Provider

		err string // empty string if no validation error is expected
	}{

		// Actions that are correct.

		{
			name: "Source/App/Users - with mapping",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
		},

		{
			name: "Source/App/Users - with transformation function",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Function: &TransformationFunction{
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						Language:      "Python",
						InProperties:  []string{"email_in"},
						OutProperties: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			provider:                testProvider{},
		},

		{
			name: "Source/Database/Users - with mapping",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                  "SELECT id, timestamp, email_in FROM my_table",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.DatabaseType,
		},
		{
			name: "Source/FileStorage/Users - with",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Connector:              "CSV",
				Path:                   "my_file.csv",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
		},

		{
			name: "Source/Website/Users - with mapping",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.WebsiteType,
		},

		{
			name: "Source/Website/Events",
			action: ActionToSet{
				Name: "Import events",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.WebsiteType,
		},

		{
			name: "Destination/App/Users - with mapping",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: &[]ExportMode{CreateOrUpdate}[0],
				MatchingProperties: &MatchingProperties{
					Internal: "email_in",
					External: types.Property{Name: "email", Type: types.Text()},
				},
				ExportOnDuplicatedUsers: &[]bool{false}[0],
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.AppType,
		},

		{
			name: "Destination/App/Users - with transformation",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Function: &TransformationFunction{
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						Language:      "Python",
						InProperties:  []string{"email_in"},
						OutProperties: []string{"email_out"},
					},
				},
				ExportMode: &[]ExportMode{CreateOrUpdate}[0],
				MatchingProperties: &MatchingProperties{
					Internal: "email_in",
					External: types.Property{Name: "email", Type: types.Text()},
				},
				ExportOnDuplicatedUsers: &[]bool{false}[0],
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.AppType,
			provider:                testProvider{},
		},

		{
			name: "Destination/App/Events - with a mapping",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.AppType,
		},

		// Actions that are invalid.

		{
			name: "Source/App/Users - mapping a not existent property",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"not_existent_property": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			err:                     `invalid mapping: property path "not_existent_property" does not exist`,
		},

		{
			name: "Source/FileStorage/Users - no file connector is specified",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Path:                   "my_file.csv",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			err:                     "actions on file storage connections must have a connector",
		},

		{
			name: "Source/App/Users - cannot specify a connector",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Connector: "CSV",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
			err:                     "actions on App connections cannot have a connector",
		},

		{
			name: "Source/FileStorage/Users - connector does not exist",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Connector:              "NotExistentConnector",
				Path:                   "my_file.csv",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			err:                     `connector "NotExistentConnector" does not exist`,
		},
		{
			name: "Source/FileStorage/Users - connector has wrong type",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Connector:              "Dummy",
				Path:                   "my_file.csv",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.AppType,
			err:                     "type of the action's connector must be File, got App",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.target == 0 {
				t.Fatal("invalid test: target cannot be 0")
			}
			if test.connectionRole == 0 {
				t.Fatal("invalid test: connectionRole cannot be 0")
			}
			if test.connectionConnectorType == 0 {
				t.Fatal("invalid test: connectionConnectorType cannot be 0")
			}
			v := validationState{}
			v.connection.role = test.connectionRole
			v.connection.connector.typ = test.connectionConnectorType
			v.connector.typ = test.connectorType
			v.connector.hasSheets = test.connectorHasSheets
			v.connector.hasUI = test.connectorHasUI
			v.provider = test.provider
			err := validateActionToSet(test.action, test.target, v)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if gotErr != test.err {
				if gotErr == "" {
					t.Fatalf("expected validation error %q, got no errors", test.err)
				}
				if test.err == "" {
					t.Fatalf("no validation errors expected, got %q", gotErr)
				}
				t.Fatalf("expected validation error %q, got %q", test.err, gotErr)
			}
		})
	}

}

// testProvider is a transformers.Provider which implements the minimum set of
// functionalities to be used in the validateActionToSet tests.
type testProvider struct{}

var _ transformers.Provider = testProvider{}

func (testProvider) Call(ctx context.Context, name, version string, inSchema, outSchema types.Type, values []map[string]any) ([]transformers.Result, error) {
	panic("not implemented")
}
func (testProvider) Close(ctx context.Context) error { panic("not implemented") }
func (testProvider) Create(ctx context.Context, name, source string) (string, error) {
	panic("not implemented")
}
func (testProvider) Delete(ctx context.Context, name string) error {
	panic("not implemented")
}
func (testProvider) SupportLanguage(language state.Language) bool {
	return language == state.JavaScript || language == state.Python
}
func (testProvider) Update(ctx context.Context, name, source string) (string, error) {
	panic("not implemented")
}

func Test_unusedProperties(t *testing.T) {
	cases := []struct {
		schema   types.Type
		paths    []string
		expected []string
	}{
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			paths: []string{"first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "last_name", Type: types.Text()},
			}),
			paths: []string{"first_name", "last_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
			}),
			expected: []string{"first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
			}),
			expected: []string{"email", "first_name"},
		},
	}
	for _, cas := range cases {
		got := unusedProperties(cas.schema, cas.paths)
		if !reflect.DeepEqual(cas.expected, got) {
			t.Fatalf("expecting %#v, got %#v", cas.expected, got)
		}
	}
}

func Test_validateLastChangeTimeFormat(t *testing.T) {
	tests := []struct {
		format string
		err    string
	}{
		// Valid.
		{format: "ISO8601"},
		{format: "Excel"},
		{format: "%Y-%m-%d %H:%M:%S"},
		{format: "%Y-%m-%d"},
		{format: "%Y"},
		{format: "%"},

		// Invalid.
		{format: "", err: "last change time format is empty"},
		{format: "iso8601", err: `last change time format "iso8601" is not valid`},
		{format: "excel", err: `last change time format "excel" is not valid`},
		{format: "Y-m-d", err: `last change time format "Y-m-d" is not valid`},
		{format: "%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y%Y", err: "last change time format is longer than 64 runes"},
		{format: "%Y-%m-%d\x00%H:%M:%S", err: "last change time format contains the NUL rune"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			got := validateLastChangeTimeFormat(test.format)
			var gotStr string
			if got != nil {
				gotStr = got.Error()
			}
			if test.err != gotStr {
				t.Fatalf("expecting %q, got %q", test.err, gotStr)
			}
		})
	}
}
