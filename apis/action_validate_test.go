//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/types"
)

var testName = regexp.MustCompile(`(?m)^(GOOD: |BAD: )(\w+)/(\w+)/(\w+) - (.+)$`)

func Test_validateActionToSet(t *testing.T) {

	tests := []struct {
		// name is the name of the test.
		// Must have the form:
		//
		// "GOOD: Role/Type/Target - description"
		//
		//   or
		//
		// "BAD: Role/Type/Target - description"
		name string

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
			name: "GOOD: Source/App/Users - with mapping",
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
			name: "GOOD: Source/App/Users - with transformation function",
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
			name: "GOOD: Source/Database/Users - with mapping",
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
			name: "GOOD: Source/FileStorage/Users - with mapping",
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
			name: "GOOD: Source/Website/Users - with mapping",
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
			name: "GOOD: Source/Website/Events - valid action",
			action: ActionToSet{
				Name: "Import events",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.WebsiteType,
		},
		{
			name: "GOOD: Destination/App/Users - with mapping",
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
			name: "GOOD: Destination/App/Users - with transformation",
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
			name: "GOOD: Destination/App/Events - with a mapping",
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
		{
			name: "GOOD: Destination/App/Events - with a transformation function",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: Transformation{
					Function: &TransformationFunction{
						Source: strings.Join([]string{
							`def transform(event: dict) -> dict:`,
							`    return {`,
							`        "email_out": event["traits"]["email"],`,
							`    }`}, "\n"),
						Language:      "Python",
						InProperties:  []string{"traits"},
						OutProperties: []string{"email_out"},
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.AppType,
			provider:                testProvider{},
		},
		// TODO(Gianluca): it's strange that in the export table there must be
		// an "id" column, but then it is not necessary for this column to be in
		// the action's output schema. See the issue
		// https://github.com/open2b/chichi/issues/807.
		{
			name: "GOOD: Destination/Database/Users - with mapping",
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
				TableName: "my_users_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.DatabaseType,
		},
		{
			name: "GOOD: Destination/Database/Users - with transformation function",
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
				TableName: "my_users_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.DatabaseType,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/FileStorage/Users - no placeholders",
			action: ActionToSet{
				Name:     "Export users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				Connector:                "CSV",
				Path:                     "my_output_users.csv",
				FileOrderingPropertyPath: "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
		},
		{
			name: "GOOD: Destination/FileStorage/Users - with placeholder",
			action: ActionToSet{
				Name:     "Export users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				Connector:                "CSV",
				Path:                     "my_output_users - ${now}.csv",
				FileOrderingPropertyPath: "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
		},

		// Actions that are invalid.

		{
			name: "BAD: Source/App/Users - empty name",
			action: ActionToSet{
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
			err:                     "name is empty",
		},
		{
			name: "BAD: Source/App/Users - action name is not UTF-8 encoded",
			action: ActionToSet{
				Name: "hello\xc5world",
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
			err:                     "name is not UTF-8 encoded",
		},
		{
			name: "BAD: Source/App/Users - name is too long",
			action: ActionToSet{
				Name: "qwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyq",
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
			err:                     "name is longer than 60 runes",
		},
		{
			name: "BAD: Source/App/Users - mapping a not existent property",
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
			name: "BAD: Source/App/Users - invalid input schema with mapping",
			action: ActionToSet{
				Name: "Import users",
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
			err:                     `input schema is required by the mapping`,
		},
		{
			name: "BAD: Source/App/Users - invalid output schema with mapping",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
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
			err:                     "output schema is required by the mapping",
		},
		{
			name: "BAD: Source/App/Users - invalid input schema with transformation function",
			action: ActionToSet{
				Name: "Import users",
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
			err:                     `input schema is required by the transformation`,
		},
		{
			name: "BAD: Source/App/Users - invalid output schema with transformation function",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
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
			err:                     "output schema is required by the transformation",
		},
		{
			name: "BAD: Source/App/Users - empty source code in transformation function",
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
			err:                     "function transformation source is empty",
		},
		{
			name: "BAD: Source/App/Users - transformation language is empty",
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
						InProperties:  []string{"email_in"},
						OutProperties: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			provider:                testProvider{},
			err:                     "transformation language is empty",
		},
		{
			name: "BAD: Source/App/Users - transformation language is invalid",
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
						Language:      "SomeWeirdLanguage",
						InProperties:  []string{"email_in"},
						OutProperties: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			provider:                testProvider{},
			err:                     `transformation language "SomeWeirdLanguage" is not valid`,
		},
		{
			name: "BAD: Source/FileStorage/Users - no file connector is specified",
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
			name: "BAD: Source/App/Users - cannot specify a connector",
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
			name: "BAD: Source/FileStorage/Users - connector does not exist",
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
			name: "BAD: Source/FileStorage/Users - connector has wrong type",
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
		{
			name: "BAD: Source/Database/Users - no identity property specified",
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
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.DatabaseType,
			err:                     "identity property is mandatory",
		},
		{
			name: "BAD: Source/Database/Users - identity property not found in input schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
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
				Query:                  "SELECT timestamp, email_in FROM my_table",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.DatabaseType,
			err:                     "identity property \"id\" not found within input schema",
		},
		{
			name: "BAD: Source/Database/Users - identity property has invalid type",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Array(types.Text())},
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
			err:                     "identity property \"id\" has kind Array instead of Int, Uint, UUID, JSON, or Text",
		},
		{
			name: "BAD: Source/App/Users - cannot specify an identity property",
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
				IdentityProperty: "my_id_property",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			err:                     "action cannot specify an identity property",
		},
		{
			name: "BAD: Source/App/Users - with both mapping and transformation function",
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
			err:                     "action cannot have both mappings and transformation",
		},
		{
			name: "BAD: Source/FileStorage/Users - path too long",
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
				Path:                   strings.Repeat("a", 1025),
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
			err:                     "path is longer than 1024 runes",
		},

		{
			name: "BAD: Source/FileStorage/Users - path cannot contain a placeholder",
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
				Path:                   "my_file-${now}.csv",
				IdentityProperty:       "id",
				LastChangeTimeProperty: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
			err:                     "placeholders syntax is not supported by source actions",
		},
		{
			name: "BAD: Source/App/Users - filters are not allowed",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: "all",
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: "is",
							Value:    "a@b",
						},
					},
				},
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
			err:                     "filters are not allowed",
		},
		{
			name: "BAD: Source/App/Users - cannot specify a path",
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
				Path: "my-file-path",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			err:                     "App actions cannot have a path",
		},
		{
			name: "BAD: Source/App/Users - cannot specify a sheet",
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
				Sheet: "sheet1",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			err:                     "App actions cannot have a sheet",
		},
		{
			name: "BAD: Source/FileStorage/Users - invalid input schema",
			action: ActionToSet{
				Name: "Import users",
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
			err:                     "input schema is required by the mapping",
		},
		{
			name: "BAD: Destination/Database/Users - table name is not UTF-8 encoded",
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
				TableName: "my\xc5z_users_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.DatabaseType,
			err:                     "table name is not UTF-8 encoded",
		},
		{
			name: "BAD: Destination/Database/Users - table name contains the NUL rune",
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
				TableName: "my_users_\x00_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.DatabaseType,
			err:                     "table name contains NUL rune",
		},
		{
			name: "BAD: Destination/FileStorage/Users - invalid compression",
			action: ActionToSet{
				Name:     "Export users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
					{Name: "last_name", Type: types.Text()},
				}),
				Connector:                "CSV",
				Path:                     "my_output_users.csv",
				Compression:              "BadCompression",
				FileOrderingPropertyPath: "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorageType,
			connectorType:           state.FileType,
			connectorHasUI:          false,
			connectorHasSheets:      false,
			err:                     "compression \"BadCompression\" is not valid",
		},
		{
			name: "BAD: Destination/Database/Users - table name is required",
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
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.DatabaseType,
			err:                     "table name cannot be empty for destination database actions",
		},
		{
			name: "BAD: Source/App/Users - output schema is not an Object",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Int(32),
				Transformation: Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.AppType,
			err:                     "out schema, if provided, must be an object",
		},
		{
			name: "BAD: Source/App/Users - input schema is not an Object",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Int(32),
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
			err:                     "input schema, if provided, must be an object",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateTestOnAction(test.name, test.connectionRole, test.connectionConnectorType, test.target, test.err); err != nil {
				t.Fatalf("test %q is badly written: %s", test.name, err)
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

var testNames = map[string]struct{}{}

// validateTestOnAction is used internally by tests to validate the coherence of
// a test on an action.
func validateTestOnAction(name string, connectionRole state.Role, connectorType state.ConnectorType, target state.Target, expectedErr string) error {
	matches := testName.FindStringSubmatch(name)
	if len(matches) != 6 {
		return errors.New("the test name does not match the 'testName' regexp")
	}
	switch matches[1] {
	case "GOOD: ":
		if expectedErr != "" {
			return errors.New("test whose name starts with 'GOOD: ' cannot expect an error")
		}
	case "BAD: ":
		if expectedErr == "" {
			return errors.New("test whose name starts with 'BAD: ' must expect a validation error")
		}
	default:
		panic(fmt.Sprintf("unexpected: %q", matches[1]))
	}
	nameRole := matches[2]
	switch nameRole {
	case "Source":
		if connectionRole != state.Source {
			return fmt.Errorf("test name specifies role %q, but test have role %q", nameRole, connectionRole)
		}
	case "Destination":
		if connectionRole != state.Destination {
			return fmt.Errorf("test name specifies role %q, but test have role %q", nameRole, connectionRole)
		}
	default:
		return fmt.Errorf("invalid role in test name: %q", nameRole)
	}
	nameType := matches[3]
	if nameType != connectorType.String() {
		return fmt.Errorf("test name specifies a connector type %q, but test have connector type %q", nameType, connectorType.String())
	}
	nameTarget := matches[4]
	if nameTarget != target.String() {
		return fmt.Errorf("test name specifies a target %q, but test indicates target %q", nameTarget, target.String())
	}
	if _, ok := testNames[name]; ok {
		return errors.New("cannot have more than one test with the same name")
	}
	testNames[name] = struct{}{}
	return nil
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
