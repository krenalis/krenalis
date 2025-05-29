//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/types"
)

var testName = regexp.MustCompile(`(?m)^(GOOD: |BAD: )(\w+)/(\w+)/(\w+) - (.+)$`)

func Test_validateAction(t *testing.T) {

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

		formatType        state.ConnectorType
		formatTargets     state.ConnectorTargets
		formatHasSettings bool
		formatHasSheets   bool

		provider transformers.FunctionProvider

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
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Source/App/Users - with mapping and filter",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Source/App/Users - with transformation function",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/App/Users - incremental",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Incremental: true,
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
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
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
		},
		{
			name: "GOOD: Source/Database/Users - incremental",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
				Incremental:          true,
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
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
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/FileStorage/Users - with mapping and filter",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
						{
							Property: "id",
							Operator: OpIsNot,
							Values:   []string{"1234567890"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/FileStorage/Users - incremental",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
				Incremental:          true,
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/SDK/Users - with mapping",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/Users - with constant mapping",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/Users - with function",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"user"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/SDK/Users - with constant function",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": "a@",`,
							`    }`}, "\n"),
						InPaths:  []string{},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/SDK/Events - valid action",
			action: ActionToSet{
				Name: "Import events",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/Events - with filters",
			action: ActionToSet{
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "anonymousId",
							Operator: OpIsNot,
							Values:   []string{"abc"},
						},
					},
				},
				Name: "Import events into the data warehouse",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Destination/App/Users - with mapping",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Destination/App/Users - with transformation",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "id",
					Out: "id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/App/Users - with mapping and filters",
			action: ActionToSet{
				Name: "Export users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "id", Type: types.Int(32)},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "id",
					Out: "id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Destination/App/Events - with a mapping",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Destination/App/Events - with a constant mapping",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Destination/App/Events - with a transformation function",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(event: dict) -> dict:`,
							`    return {`,
							`        "email_out": event["traits"]["email"],`,
							`    }`}, "\n"),
						InPaths:  []string{"traits"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/App/Events - with a constant transformation function",
			action: ActionToSet{
				Name:     "Dispatch events to app",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(event: dict) -> dict:`,
							`    return {`,
							`        "email_out": "a@b",`,
							`    }`}, "\n"),
						InPaths:  []string{},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/Database/Users - with mapping",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "name_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "name_out", Type: types.Text(), Nullable: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
						"name_out":  "name_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
		},
		{
			name: "GOOD: Destination/Database/Users - with transformation function",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`        "first_name": user["first_name"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in", "first_name"},
						OutPaths: []string{"email_out", "first_name"},
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/FileStorage/Users - no placeholders",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
					{Name: "last_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "CSV",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Destination/FileStorage/Users - with placeholder",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
					{Name: "last_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "CSV",
				Path:      "my_output_users - ${now}.csv",
				OrderBy:   "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Destination/FileStorage/Users - with filter",
			action: ActionToSet{
				Name: "Export users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{Property: "first_name", Operator: OpIs, Values: []string{"Bob"}},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
					{Name: "last_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "CSV",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/App/Users - input schema can contain meta properties",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
					{Name: "__id__", Type: types.Int(32)},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in __id__",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
		},
		{
			name: "GOOD: Source/App/Users - InPaths refers to second-level property of input schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
					{Name: "additional_properties", Type: types.Object([]types.Property{
						{Name: "a", Type: types.Text()},
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths: []string{
							"email_in",
							"additional_properties.a",
							"additional_properties.b",
							"additional_properties.c",
						},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "name is empty",
		},
		{
			name: "BAD: Source/App/Users - action name contains invalid UTF-8 encoded characters",
			action: ActionToSet{
				Name: "hello\xc5world",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "name contains invalid UTF-8 encoded characters",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"not_existent_property": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `invalid mapping: property path "not_existent_property" does not exist`,
		},
		{
			name: "BAD: Source/App/Users - invalid input schema with mapping",
			action: ActionToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `input schema is required by the mapping`,
		},
		{
			name: "BAD: Source/App/Users - invalid output schema with mapping",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "output schema is required by the mapping",
		},
		{
			name: "BAD: Source/App/Users - invalid input schema with transformation function",
			action: ActionToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     `input schema is required by the transformation function`,
		},
		{
			name: "BAD: Source/App/Users - invalid output schema with transformation function",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "output schema is required by the transformation function",
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
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "source of transformation function is empty",
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
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
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
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "SomeWeirdLanguage",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     `transformation language "SomeWeirdLanguage" is not valid`,
		},
		{
			name: "BAD: Source/FileStorage/Users - no format is specified",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			err:                     "actions on file storage connections must have a format",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format: "CSV",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "actions on App connections cannot have a format",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "NotExistentFormat",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			err:                     `format "NotExistentFormat" does not exist`,
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "Dummy",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.App,
			err:                     "format does not refer to a file connector",
		},
		{
			name: "BAD: Source/Database/Users - no identity column specified",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is mandatory",
		},
		{
			name: "BAD: Source/Database/Users - identity column not found in input schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT timestamp, email_in FROM my_table",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column \"id\" not found within input schema",
		},
		{
			name: "BAD: Source/Database/Users - incremental without last change time column",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:          "SELECT id, email_in FROM my_table",
				IdentityColumn: "id",
				Incremental:    true,
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "incremental requires a last change time column",
		},
		{
			name: "BAD: Source/Database/Users - identity column has invalid type",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Array(types.Text())},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column \"id\" has kind array instead of int, uint uuid, json, or text",
		},
		{
			name: "BAD: Source/App/Users - cannot specify an identity column",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				IdentityColumn: "my_id_column",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "action cannot specify an identity column",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "action cannot have both transformation mapping and function",
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 strings.Repeat("a", 1025),
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
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
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 "my_file-${now}.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "placeholders syntax is not supported by source actions",
		},
		{
			name: "BAD: Source/App/Users - cannot specify a path",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Path: "my-file-path",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
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
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Sheet: "sheet1",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "App actions cannot have a sheet",
		},
		{
			name: "BAD: Source/FileStorage/Users - invalid input schema",
			action: ActionToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "CSV",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "input schema is required by the mapping",
		},
		{
			name: "BAD: Source/FileStorage/Users - incremental without last change time column",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:         "CSV",
				Path:           "my_file.csv",
				IdentityColumn: "id",
				Incremental:    true,
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "incremental requires a last change time column",
		},
		{
			name: "BAD: Destination/Database/Users - table name contains invalid UTF-8 encoded characters",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my\xc5z_users_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table name contains invalid UTF-8 encoded characters",
		},
		{
			name: "BAD: Destination/Database/Users - table name contains the NUL byte",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_\x00_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table name contains the NUL byte",
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
				Format:      "CSV",
				Path:        "my_output_users.csv",
				Compression: "BadCompression",
				OrderBy:     "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "compression \"BadCompression\" is not valid",
		},
		{
			name: "BAD: Destination/Database/Users - table name is required",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `table name cannot be empty for destination database actions`,
		},
		{
			name: "BAD: Source/App/Users - output schema is not an object",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Int(32),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "out schema, if provided, must be an object",
		},
		{
			name: "BAD: Source/App/Users - input schema is not an object",
			action: ActionToSet{
				Name:     "Import users",
				InSchema: types.Int(32),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "input schema, if provided, must be an object",
		},
		{
			name: "BAD: Source/App/Users - output schema contains a nullable property",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.Object([]types.Property{
						{Name: "email", Type: types.Text(), ReadOptional: true, Nullable: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"x.email": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `output action schema property "x.email" cannot have Nullable set to true`,
		},
		{
			name: "BAD: Source/App/Users - output schema contains a required property",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true, ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `output action schema property "email_out" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Destination/App/Users - input schema contains a nullable property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true, Nullable: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     `input action schema property "email_in" cannot have Nullable set to true`,
		},
		{
			name: "BAD: Destination/App/Users - input schema contains a required property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), CreateRequired: true, ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     `input action schema property "email_in" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Source/App/Users - output schema cannot contain meta properties",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
					{Name: "__id__", Type: types.Int(32), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
						"__id__":    "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `output action schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/App/Users - input schema cannot contain meta properties",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "__id__", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in __id__",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     `input action schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/App/Users - incremental is not supported",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Incremental: true,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "incremental cannot be true for destination actions",
		},
		{
			name: "BAD: Destination/App/Events - input schema must be invalid",
			action: ActionToSet{
				Name: "Dispatch events to app",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Events,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "input schema must be invalid for actions that dispatch events to apps",
		},
		{
			name: "BAD: Source/SDK/Users - input schema must be invalid",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "input schema must be invalid for actions that import user identities from events",
		},
		{
			name: "BAD: Destination/Database/Users - missing database table key",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key cannot be empty for destination database actions",
		},
		{
			name: "BAD: Destination/Database/Users - table key not in schema",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "some_property",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key \"some_property\" not found within output schema",
		},
		{
			name: "BAD: Destination/Database/Users - output property is required for creation",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "my_array_prop", Type: types.Array(types.Text()), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `output action schema property "my_array_prop" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Destination/Database/Users - output property is required for update",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true, UpdateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `output action schema property "email_out" cannot have UpdateRequired set to true`,
		},
		{
			name: "BAD: Destination/Database/Users - table key is nullable",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true, Nullable: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `table key property "email_out" in output action schema cannot have Nullable set to true`,
		},
		{
			name: "BAD: Destination/Database/Users - table key has wrong type",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "my_array_prop", Type: types.Array(types.Text()), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "my_array_prop",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "type array cannot be used as table key",
		},
		{
			name: "BAD: Destination/Database/Users - unmapped property in input schema",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
					{Name: "a", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out":  "email_in",
						"first_name": "first_name",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "input schema contains unused properties: a",
		},
		{
			name: "BAD: Destination/Database/Users - unused property in output schema",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "first_name", Type: types.Text()},
					{Name: "x", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out":  "email_in",
						"first_name": "first_name",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "output schema contains unused properties: x",
		},
		{
			name: "BAD: Destination/Database/Users - table key is not a valid property name",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "some-invalid-property-name",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key is not a valid property name",
		},
		{
			name: "BAD: Destination/Database/Users - an expression must be mapped to the table key",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "a", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
					{Name: "b", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"b": "a",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "an expression must be mapped to the table key",
		},
		{
			name: "BAD: Destination/Database/Users - transformation function does not transform to the table key",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "my_key", Type: types.Text(), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
				TableName: "my_users_table",
				TableKey:  "my_key",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
			err:                     "the out properties of the transformation function must contain the table key",
		},
		{
			name: "BAD: Source/App/Users - table key cannot be specified",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableKey: "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "table key is not allowed",
		},
		{
			name: "BAD: Source/App/Users - transformation function with unused property in input schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
					{Name: "tax_code", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "input schema contains unused properties: tax_code",
		},
		{
			name: "BAD: Source/App/Users - transformation function with unused property in output schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
					{Name: "last_name", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "output schema contains unused properties: last_name",
		},
		{
			name: "GOOD: Source/App/Groups - target Groups is not supported, but this should be checked before validating the action, not by the action validation itself",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Groups,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
		},
		{
			name: "BAD: Source/App/Users - input schema cannot contain a property with a placeholder",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), Placeholder: "Your Email"},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `input action schema property "email_in" has a placeholder, but action schema properties cannot have placeholders`,
		},
		{
			name: "BAD: Source/App/Users - output schema cannot contain a property with a placeholder",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), Placeholder: "Your Email"},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `output action schema property "email_out" has a placeholder, but action schema properties cannot have placeholders`,
		},
		{
			name: "BAD: Source/App/Users - output schema - which refers to users - cannot contain conflicting properties",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
					{Name: "email", Type: types.Object([]types.Property{
						{Name: "out", Type: types.Text(), ReadOptional: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
						"email.out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `two output action schema properties would have the same column name "email_out" in the data warehouse, case-insensitively`,
		},
		{
			name: "BAD: Source/App/Users - output schema - which refers to users - cannot have a property with type array(object)",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
					{Name: "many_values_in", Type: types.Array(types.Object([]types.Property{
						{Name: "a", Type: types.Text()},
					}))},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
					{Name: "many_values_out", Type: types.Array(types.Object([]types.Property{
						{Name: "a", Type: types.Text(), ReadOptional: true},
					})), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out":       "email_in",
						"many_values_out": "many_values_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     `output action schema property "many_values_out" cannot have type array(object)`,
		},
		{
			name: "BAD: Destination/App/Users - filter refers to a property not in input schema",
			action: ActionToSet{
				Name: "Export users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "__id__",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "filter is not valid: property path \"__id__\" does not exist",
		},
		{
			name: "BAD: Destination/App/Users - filter refers to a meta property in input schema",
			action: ActionToSet{
				Name: "Export users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "__id__",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "__id__", Type: types.Text(), ReadOptional: true},
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     `input action schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/FileStorage/Users - no input schema",
			action: ActionToSet{
				Name:      "Export users",
				OutSchema: types.Type{},
				Format:    "CSV",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "input schema must be valid when exporting users to file",
		},
		{
			name: "BAD: Destination/App/Users - output matching property transformed with mapping",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
						"email_out":  "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/App/Users - output matching property transformed with mapping with function",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_out", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`        "in": user["in"]`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in", "id"},
						OutPaths: []string{"email_out", "id"},
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "id",
					Out: "id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Destination/App/Users - output matching property not in out schema",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "output matching property \"email_out\" not found within the output schema",
		},
		{
			name: "BAD: Source/Database/Users - filters are not allowed",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
						{
							Property: "id",
							Operator: OpIsNot,
							Values:   []string{"1234567890"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "filters are not allowed",
		},
		{
			name: "BAD: Source/SDK/Events - cannot provide input schema",
			action: ActionToSet{
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "anonymousId",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "anonymousId", Type: types.Text()},
				}),
				Name: "Import events into the data warehouse",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "input schema must be invalid for actions that import events into data warehouse",
		},
		{
			name: "BAD: Source/SDK/Events - cannot provide output schema",
			action: ActionToSet{
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.Text()},
				}),
				Name: "Import events into the data warehouse",
			},
			target:                  state.Events,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "output schema must be invalid when importing events into data warehouse",
		},
		{
			name: "BAD: Source/App/Users - InPaths refers to a not-existent second-level property of input schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
					{Name: "additional_properties", Type: types.Object([]types.Property{
						{Name: "a", Type: types.Text()},
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths: []string{
							"email_in",
							"additional_properties.a",
							"additional_properties.b",
							"additional_properties.zzzz",
							"additional_properties.c",
						},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "input property \"additional_properties.zzzz\" of transformation function does not exist in schema",
		},
		{
			name: "BAD: Destination/App/Users - with mapping transformation that overwrites the out matching property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/App/Users - with transformation function that overwrites the out matching property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email"},
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "email",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Destination/App/Users - in matching property cannot be a property path",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "additional", Type: types.Object([]types.Property{
						{Name: "first_name", Type: types.Text()},
						{Name: "last_name", Type: types.Text()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "additional", Type: types.Object([]types.Property{
						{Name: "first_name", Type: types.Text()},
						{Name: "last_name", Type: types.Text()},
					})},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "additional.first_name",
					Out: "email_in",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "matching properties cannot be property paths, can only be property names",
		},
		{
			name: "BAD: Destination/App/Users - out matching property cannot be a property path",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "additional", Type: types.Object([]types.Property{
						{Name: "first_name", Type: types.Text()},
						{Name: "last_name", Type: types.Text()},
					})},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "additional.first_name",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			provider:                testProvider{},
			err:                     "matching properties cannot be property paths, can only be property names",
		},
		{
			name: "BAD: Source/App/Users - with constant mapping (not allowed) and filter",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "transformation must map at least one property",
		},
		{
			name: "BAD: Destination/Database/Users - only the table key is mapped",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "in addition to the table key, there must be at least one other mapped column",
		},
		{
			name: "BAD: Destination/Database/Users - only the table key is transformed",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
			err:                     "the out properties of the transformation function must contain at least one other property in addition to the table key",
		},
		{
			name: "BAD: Source/App/Users - unused property in output schema",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.Object([]types.Property{
						{Name: "y", Type: types.Text(), ReadOptional: true},
						{Name: "z", Type: types.Text(), ReadOptional: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"x.y": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "output schema contains unused properties: x.z",
		},
		{
			name: "BAD: Source/App/Users - with filter but no input schema",
			action: ActionToSet{
				Name: "Import users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{
							Property: "email_in",
							Operator: OpIsNot,
							Values:   []string{"a@b"},
						},
					},
				},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "input schema is required by the filter",
		},
		{
			name: "BAD: Source/App/Users - with non-nil mapping but no properties mapped",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{},
				},
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "transformation mapping must have mapped properties",
		},
		{
			name: "BAD: Destination/App/Users - missing input matching property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					Out: "email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "input matching property cannot be empty if output matching property is not empty",
		},
		{
			name: "BAD: Destination/App/Users - missing output matching property",
			action: ActionToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text(), ReadOptional: true},
					{Name: "first_name", Type: types.Text(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text()},
					{Name: "first_name", Type: types.Text()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In: "email_in",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.Users,
			connectionRole:          state.Destination,
			connectionConnectorType: state.App,
			err:                     "output matching property cannot be empty if input matching property is not empty",
		},
		{
			name: "BAD: Source/Database/Users - identity column is not a valid property name",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       "id column",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is not a valid property name",
		},
		{
			name: "BAD: Source/Database/Users - identity column is too long",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				IdentityColumn:       strings.Repeat("c", 1025),
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is longer than 1024 runes",
		},
		{
			name: "BAD: Source/App/Users - table name is not allowed",
			action: ActionToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.Text(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "users",
			},
			target:                  state.Users,
			connectionRole:          state.Source,
			connectionConnectorType: state.App,
			err:                     "table name is not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateTestOnAction(test.name, test.connectionRole, test.connectionConnectorType, test.target, test.err); err != nil {
				t.Fatalf("test %q is badly written: %s", test.name, err)
			}
			v := validationState{}
			v.target = test.target
			v.connection.role = test.connectionRole
			v.connection.connector.typ = test.connectionConnectorType
			v.format.typ = test.formatType
			v.format.targets = test.formatTargets
			v.format.hasSheets = test.formatHasSheets
			v.format.hasSettings = test.formatHasSettings
			v.provider = test.provider
			err := validateActionToSet(test.action, v)
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

// testProvider is a transformers.FunctionProvider which implements the minimum
// set of functionalities to be used in the validateActionToSet tests.
type testProvider struct{}

var _ transformers.FunctionProvider = testProvider{}

func (testProvider) Call(ctx context.Context, id, version string, inSchema, outSchema types.Type, preserveJSON bool, records []transformers.Record) error {
	panic("not implemented")
}
func (testProvider) Close(ctx context.Context) error { panic("not implemented") }
func (testProvider) Create(ctx context.Context, name string, language state.Language, source string) (string, string, error) {
	panic("not implemented")
}
func (testProvider) Delete(ctx context.Context, id string) error {
	panic("not implemented")
}
func (testProvider) SupportLanguage(language state.Language) bool {
	return language == state.JavaScript || language == state.Python
}
func (testProvider) Update(ctx context.Context, id, source string) (string, error) {
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
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
					{Name: "zip_code", Type: types.Text()},
				})},
			}),
			expected: []string{"address.street", "address.zip_code", "email", "first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
					{Name: "zip_code", Type: types.Text()},
				})},
			}),
			paths:    []string{"email"},
			expected: []string{"address.street", "address.zip_code", "first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
					{Name: "zip_code", Type: types.Text()},
				})},
			}),
			paths:    []string{"address.street"},
			expected: []string{"address.zip_code", "email", "first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
					{Name: "zip_code", Type: types.Text()},
				})},
			}),
			paths:    []string{"address", "first_name", "email"},
			expected: nil,
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.Text()},
					{Name: "zip_code", Type: types.Text()},
				})},
			}),
			paths:    []string{"address.zip_code", "email", "first_name"},
			expected: []string{"address.street"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
			}),
			expected: []string{"x1.a.b", "x1.a.c", "x1.y", "x1.z", "x2.a.b", "x2.a.c", "x2.y", "x2.z"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
			}),
			paths:    []string{"x1", "x2"},
			expected: nil,
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
			}),
			paths:    []string{"x1"},
			expected: []string{"x2.a.b", "x2.a.c", "x2.y", "x2.z"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.Text()},
					{Name: "z", Type: types.Text()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.Text()},
						{Name: "c", Type: types.Text()},
					})},
				})},
			}),
			paths:    []string{"x2.a", "x1"},
			expected: []string{"x2.y", "x2.z"},
		},
	}
	for _, cas := range cases {
		got := unusedPropertyPaths(cas.schema, cas.paths)
		if !reflect.DeepEqual(cas.expected, got) {
			t.Fatalf("expected %#v, got %#v", cas.expected, got)
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
		{format: "%Y-%m-%d\x00%H:%M:%S", err: "last change time format contains the NUL byte"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			got := validateLastChangeTimeFormat(test.format)
			var gotStr string
			if got != nil {
				gotStr = got.Error()
			}
			if test.err != gotStr {
				t.Fatalf("expected %q, got %q", test.err, gotStr)
			}
		})
	}
}

func Test_validateTransformationFunctionPaths(t *testing.T) {

	tests := []struct {
		name                 string
		io                   string
		schema               types.Type
		paths                []string
		dispatchEventsToApps bool
		expectedError        string
	}{
		{
			name: "Referencing a top-level property",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
				{Name: "xy", Type: types.Text()},
				{Name: "x_y", Type: types.Text()},
				{Name: "x_z", Type: types.Text()},
				{Name: "z", Type: types.Text()},
			}),
			paths: []string{"x_y", "x", "xy", "x_z", "z"},
		},
		{
			name: "Paths cannot be nil",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
			paths:         nil,
			expectedError: "input properties of transformation function cannot be null",
		},
		{
			name: "Paths cannot be empty",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
			paths:         []string{},
			expectedError: "there are no input properties in transformation function",
		},
		{
			name: "Paths can be empty when dispatching events",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
			paths:                []string{},
			dispatchEventsToApps: true,
		},
		{
			name: "Referencing a second-level property",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
			}),
			paths: []string{"x.a"},
		},
		{
			name: "Duplicated property path",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
			}),
			paths:         []string{"x.a", "x.a"},
			expectedError: "transformation function input property path \"x.a\" is repeated",
		},
		{
			name: "Two identical paths (first level)",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
			paths:         []string{"x", "x"},
			expectedError: "transformation function input property path \"x\" is repeated",
		},
		{
			name: "Two identical paths (second level)",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
			}),
			paths:         []string{"x.a", "x.a"},
			expectedError: "transformation function input property path \"x.a\" is repeated",
		},
		{
			name: "Sub path error #1",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
			}),
			paths:         []string{"x.a", "x"},
			expectedError: "transformation function input paths cannot contain both \"x\" and its sub-property path \"x.a\"",
		},
		{
			name: "Sub path error #2",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					})},
				})},
			}),
			paths:         []string{"a.b", "a.c", "a.e.f", "a.e.g", "a.e"},
			expectedError: "transformation function input paths cannot contain both \"a.e\" and its sub-property path \"a.e.f\"",
		},
		{
			name: "Sub path error #3",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					})},
				})},
			}),
			paths:         []string{"a.b", "a.c", "a.e", "a.e.f", "a.e.g"},
			expectedError: "transformation function input paths cannot contain both \"a.e\" and its sub-property path \"a.e.f\"",
		},
		{
			name: "Property path not in schema",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					})},
				})},
			}),
			paths:         []string{"a.b", "a.c", "a.e.f", "a.e.z", "a.e.g"},
			expectedError: "input property \"a.e.z\" of transformation function does not exist in schema",
		},
		{
			name: "Property path refers to an array sub-property",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					})},
					{Name: "array", Type: types.Array(types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					}))},
				})},
			}),
			paths:         []string{"a.b", "a.c", "a.e.f", "a.array.g", "a.e.g"},
			expectedError: "input property \"a.array.g\" of transformation function does not exist in schema",
		},
		{
			name: "Property path refers to an array sub-property",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "a", Type: types.Object([]types.Property{
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					})},
					{Name: "map", Type: types.Map(types.Object([]types.Property{
						{Name: "f", Type: types.Text()},
						{Name: "g", Type: types.Text()},
						{Name: "h", Type: types.Text()},
					}))},
				})},
			}),
			paths:         []string{"a.b", "a.c", "a.e.f", "a.array.g", "a.e.g"},
			expectedError: "input property \"a.array.g\" of transformation function does not exist in schema",
		},
		{
			name: "Invalid property path",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Text()},
			}),
			paths:         []string{"x", "x.invalid-path.z"},
			expectedError: "transformation function input property path \"x.invalid-path.z\" is not valid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErr := validateTransformationFunctionPaths(test.io, test.schema, test.paths, test.dispatchEventsToApps)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedError {
				t.Fatalf("expected error %q, got %q", test.expectedError, gotErrStr)
			}
		})
	}

}
