// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/tools/types"
)

var testName = regexp.MustCompile(`(?m)^(GOOD: |BAD: )(\w+)/(\w+)/(\w+) - (.+)$`)

func Test_validatePipeline(t *testing.T) {

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

		// The PipelineToSet to validate.
		pipeline PipelineToSet

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

		// Pipelines that are correct.

		{
			name: "GOOD: Source/API/User - with mapping",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Source/API/User - with mapping and filter",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Source/API/User - with transformation function",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/API/User - incremental",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Incremental: true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/Database/User - with mapping",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
		},
		{
			name: "GOOD: Source/Database/User - incremental",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
		},
		{
			name: "GOOD: Source/FileStorage/User - with mapping",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/FileStorage/User - with mapping and filter",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/FileStorage/User - incremental",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
				Incremental:          true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/SDK/User - with mapping",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/User - with constant mapping",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/User - with function",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(event: dict) -> dict:`,
							`    return {`,
							`        "email_out": event["traits"]["email_in"],`,
							`    }`}, "\n"),
						InPaths:  []string{"traits"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/SDK/User - with constant function",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Source/SDK/Event - valid pipeline",
			pipeline: PipelineToSet{
				Name: "Import events",
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/SDK/Event - with filters",
			pipeline: PipelineToSet{
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
			target:                  state.TargetEvent,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
		},
		{
			name: "GOOD: Source/Webhook/User - with mapping",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Webhook,
		},
		{
			name: "GOOD: Source/Webhook/Event - valid pipeline",
			pipeline: PipelineToSet{
				Name: "Import events",
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Source,
			connectionConnectorType: state.Webhook,
		},
		{
			name: "GOOD: Destination/API/User - with mapping",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/User - update on duplicates allowed",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "first_name", Type: types.String()},
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
				UpdateOnDuplicates: true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/User - with transformation",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/API/User - with mapping and filters",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/Event - with a mapping",
			pipeline: PipelineToSet{
				Name:     "Dispatch events to api",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/Event - with a constant mapping",
			pipeline: PipelineToSet{
				Name:     "Dispatch events to api",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/Event - with a transformation function",
			pipeline: PipelineToSet{
				Name:     "Dispatch events to api",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetEvent,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/API/Event - with a constant transformation function",
			pipeline: PipelineToSet{
				Name:     "Dispatch events to api",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetEvent,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/Database/User - with mapping",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "name_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "name_out", Type: types.String(), Nullable: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
		},
		{
			name: "GOOD: Destination/Database/User - with transformation function",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/FileStorage/User - no placeholders",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
					{Name: "last_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "csv",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Destination/FileStorage/User - with placeholder",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
					{Name: "last_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "csv",
				Path:      "my_output_users - ${now}.csv",
				OrderBy:   "email",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Destination/FileStorage/User - with filter",
			pipeline: PipelineToSet{
				Name: "Export users",
				Filter: &Filter{
					Logical: OpAnd,
					Conditions: []FilterCondition{
						{Property: "first_name", Operator: OpIs, Values: []string{"Bob"}},
					},
				},
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
					{Name: "last_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Type{},
				Format:    "csv",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
		},
		{
			name: "GOOD: Source/API/User - input schema can contain meta properties",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
					{Name: "__id__", Type: types.Int(32)},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in __id__",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Source/API/User - InPaths refers to second-level property of input schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
					{Name: "additional_properties", Type: types.Object([]types.Property{
						{Name: "a", Type: types.String()},
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},

		// Pipelines that are invalid.

		{
			name: "BAD: Source/API/User - empty name",
			pipeline: PipelineToSet{
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "name is empty",
		},
		{
			name: "BAD: Source/API/User - pipeline name contains invalid UTF-8 encoded characters",
			pipeline: PipelineToSet{
				Name: "hello\xc5world",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "name contains invalid UTF-8 encoded characters",
		},
		{
			name: "BAD: Source/API/User - name is too long",
			pipeline: PipelineToSet{
				Name: "qwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyqwertyq",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "name is longer than 60 runes",
		},
		{
			name: "BAD: Source/API/User - mapping a not existent property",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"not_existent_property": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `invalid mapping: property path "not_existent_property" does not exist`,
		},
		{
			name: "BAD: Source/API/User - invalid input schema with mapping",
			pipeline: PipelineToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `input schema is required by the mapping`,
		},
		{
			name: "BAD: Source/API/User - invalid output schema with mapping",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "output schema is required by the mapping",
		},
		{
			name: "BAD: Source/API/User - invalid input schema with transformation function",
			pipeline: PipelineToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     `input schema is required by the transformation function`,
		},
		{
			name: "BAD: Source/API/User - invalid output schema with transformation function",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "output schema is required by the transformation function",
		},
		{
			name: "BAD: Source/API/User - empty source code in transformation function",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						InPaths:  []string{"email_in"},
						OutPaths: []string{"email_out"},
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "source of transformation function is empty",
		},
		{
			name: "BAD: Source/API/User - transformation language is empty",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "transformation language is empty",
		},
		{
			name: "BAD: Source/API/User - transformation language is invalid",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     `transformation language "SomeWeirdLanguage" is not valid`,
		},
		{
			name: "BAD: Source/FileStorage/User - no format is specified",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			err:                     "pipelines on file storage connections must have a format",
		},
		{
			name: "BAD: Source/API/User - cannot specify a connector",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format: "csv",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "pipelines on API connections cannot have a format",
		},
		{
			name: "BAD: Source/FileStorage/User - connector does not exist",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			err:                     `format "NotExistentFormat" does not exist`,
		},
		{
			name: "BAD: Source/FileStorage/User - connector has wrong type",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.API,
			err:                     "format does not refer to a file connector",
		},
		{
			name: "BAD: Source/Database/User - no identity column specified",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Query:                "SELECT id, timestamp, email_in FROM my_table",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is mandatory",
		},
		{
			name: "BAD: Source/Database/User - identity column not found in input schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column \"id\" not found within input schema",
		},
		{
			name: "BAD: Source/Database/User - incremental without last change time column",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "incremental requires a last change time column",
		},
		{
			name: "BAD: Source/Database/User - identity column has invalid type",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Array(types.String())},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column \"id\" has kind array instead of int, uint uuid, json, or string",
		},
		{
			name: "BAD: Source/API/User - cannot specify an identity column",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				IdentityColumn: "my_id_column",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "pipeline cannot specify an identity column",
		},
		{
			name: "BAD: Source/API/User - with both mapping and transformation function",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "pipeline cannot have both transformation mapping and function",
		},
		{
			name: "BAD: Source/FileStorage/User - path too long",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 strings.Repeat("a", 1025),
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "path is longer than 1024 runes",
		},

		{
			name: "BAD: Source/FileStorage/User - path cannot contain a placeholder",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 "my_file-${now}.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "placeholders syntax is not supported by source pipelines",
		},
		{
			name: "BAD: Source/API/User - cannot specify a path",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Path: "my-file-path",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "API pipelines cannot have a path",
		},
		{
			name: "BAD: Source/API/User - cannot specify a sheet",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Sheet: "sheet1",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "API pipelines cannot have a sheet",
		},
		{
			name: "BAD: Source/FileStorage/User - invalid input schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:               "csv",
				Path:                 "my_file.csv",
				IdentityColumn:       "id",
				LastChangeTimeColumn: "timestamp",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "input schema is required by the mapping",
		},
		{
			name: "BAD: Source/FileStorage/User - incremental without last change time column",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Format:         "csv",
				Path:           "my_file.csv",
				IdentityColumn: "id",
				Incremental:    true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "incremental requires a last change time column",
		},
		{
			name: "BAD: Destination/Database/User - table name contains invalid UTF-8 encoded characters",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my\xc5z_users_table",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table name contains invalid UTF-8 encoded characters",
		},
		{
			name: "BAD: Destination/Database/User - table name contains the NUL byte",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_\x00_table",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table name contains the NUL byte",
		},
		{
			name: "BAD: Destination/FileStorage/User - invalid compression",
			pipeline: PipelineToSet{
				Name:     "Export users",
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "first_name", Type: types.String()},
					{Name: "last_name", Type: types.String()},
				}),
				Format:      "csv",
				Path:        "my_output_users.csv",
				Compression: "BadCompression",
				OrderBy:     "email",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "compression \"BadCompression\" is not valid",
		},
		{
			name: "BAD: Destination/Database/User - table name is required",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `table name cannot be empty for destination database pipelines`,
		},
		{
			name: "BAD: Source/API/User - output schema is not an object",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Int(32),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "out schema, if provided, must be an object",
		},
		{
			name: "BAD: Source/API/User - input schema is not an object",
			pipeline: PipelineToSet{
				Name:     "Import users",
				InSchema: types.Int(32),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "input schema, if provided, must be an object",
		},
		{
			name: "BAD: Source/API/User - output schema contains a nullable property",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.Object([]types.Property{
						{Name: "email", Type: types.String(), ReadOptional: true, Nullable: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"x.email": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `output pipeline schema property "x.email" cannot have Nullable set to true`,
		},
		{
			name: "BAD: Source/API/User - output schema contains a required property",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true, ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `output pipeline schema property "email_out" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Destination/API/User - input schema contains a nullable property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true, Nullable: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     `input pipeline schema property "email_in" cannot have Nullable set to true`,
		},
		{
			name: "BAD: Destination/API/User - input schema contains a required property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), CreateRequired: true, ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     `input pipeline schema property "email_in" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Source/API/User - output schema cannot contain meta properties",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
					{Name: "__id__", Type: types.Int(32), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
						"__id__":    "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `output pipeline schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/API/User - input schema cannot contain meta properties",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "__id__", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     `input pipeline schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/API/User - incremental is not supported",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				Incremental: true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "incremental cannot be true for destination pipelines",
		},
		{
			name: "BAD: Destination/API/Event - input schema must be invalid",
			pipeline: PipelineToSet{
				Name: "Dispatch events to api",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "input schema must be invalid for pipelines that send events to apps",
		},
		{
			name: "BAD: Source/SDK/User - input schema must be invalid",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "input schema must be invalid for pipelines that import identities from events",
		},
		{
			name: "BAD: Source/SDK/User - property 'mpid' does not exist in the event schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				OutSchema: types.Object([]types.Property{
					{Name: "userID", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"userID": "mpid",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "invalid mapping: property \"mpid\" does not exist",
		},
		{
			name: "BAD: Source/Webhook/User - input schema must be invalid",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "traits.email",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Webhook,
			err:                     "input schema must be invalid for pipelines that import identities from events",
		},
		{
			name: "BAD: Destination/Database/User - missing database table key",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key cannot be empty for destination database pipelines",
		},
		{
			name: "BAD: Destination/Database/User - table key not in schema",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "some_property",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key \"some_property\" not found within output schema",
		},
		{
			name: "BAD: Destination/Database/User - output property is required for creation",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "my_array_prop", Type: types.Array(types.String()), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `output pipeline schema property "my_array_prop" cannot have CreateRequired set to true`,
		},
		{
			name: "BAD: Destination/Database/User - output property is required for update",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true, UpdateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `output pipeline schema property "email_out" cannot have UpdateRequired set to true`,
		},
		{
			name: "BAD: Destination/Database/User - table key is nullable",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true, Nullable: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     `table key property "email_out" in output pipeline schema cannot have Nullable set to true`,
		},
		{
			name: "BAD: Destination/Database/User - table key has wrong type",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "my_array_prop", Type: types.Array(types.String()), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "my_array_prop",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "type array(string) cannot be used as table key",
		},
		{
			name: "BAD: Destination/Database/User - unmapped property in input schema",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
					{Name: "a", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "input schema contains an unused property: a",
		},
		{
			name: "BAD: Destination/Database/User - unused property in output schema",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "first_name", Type: types.String()},
					{Name: "x", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "output schema contains an unused property: x",
		},
		{
			name: "BAD: Destination/Database/User - table key is not a valid property name",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "some-invalid-property-name",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "table key is not a valid property name",
		},
		{
			name: "BAD: Destination/Database/User - an expression must be mapped to the table key",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "a", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
					{Name: "b", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"b": "a",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "an expression must be mapped to the table key",
		},
		{
			name: "BAD: Destination/Database/User - transformation function does not transform to the table key",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "my_key", Type: types.String(), CreateRequired: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
			err:                     "the out properties of the transformation function must contain the table key",
		},
		{
			name: "BAD: Source/API/User - table key cannot be specified",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableKey: "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "table key is not allowed",
		},
		{
			name: "BAD: Source/API/User - transformation function with unused property in input schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
					{Name: "tax_code", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "input schema contains an unused property: tax_code",
		},
		{
			name: "BAD: Source/API/User - transformation function with unused property in output schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
					{Name: "last_name", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "output schema contains an unused property: last_name",
		},
		{
			name: "GOOD: Source/API/Group - target Group is not supported, but this should be checked before validating the pipeline, not by the pipeline validation itself",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetGroup,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
		},
		{
			name: "GOOD: Destination/API/User - in matching property can be a property path",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "additional", Type: types.Object([]types.Property{
						{Name: "first_name", Type: types.String(), ReadOptional: true},
					}), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "app_id", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "additional.first_name",
					Out: "app_id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "GOOD: Destination/API/User - out matching property can be a property path",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "additional", Type: types.Object([]types.Property{
						{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
		},
		{
			name: "BAD: Source/API/User - input schema cannot contain a property with a prefilled value",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), Prefilled: "Your Email"},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `input pipeline schema property "email_in" has a prefilled value, but pipeline schema properties cannot have prefilled values`,
		},
		{
			name: "BAD: Source/API/User - output schema cannot contain a property with a prefilled value",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), Prefilled: "Your Email"},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `output pipeline schema property "email_out" has a prefilled value, but pipeline schema properties cannot have prefilled values`,
		},
		{
			name: "BAD: Source/API/User - output schema - which refers to users - cannot contain conflicting properties",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
					{Name: "email", Type: types.Object([]types.Property{
						{Name: "out", Type: types.String(), ReadOptional: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
						"email.out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `two output pipeline schema properties would have the same column name "email_out" in the data warehouse, case-insensitively`,
		},
		{
			name: "BAD: Source/API/User - output schema - which refers to users - cannot have a property with type array(object)",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
					{Name: "many_values_in", Type: types.Array(types.Object([]types.Property{
						{Name: "a", Type: types.String()},
					}))},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
					{Name: "many_values_out", Type: types.Array(types.Object([]types.Property{
						{Name: "a", Type: types.String(), ReadOptional: true},
					})), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out":       "email_in",
						"many_values_out": "many_values_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     `output pipeline schema property "many_values_out" cannot have type array(object)`,
		},
		{
			name: "BAD: Destination/API/User - filter refers to a property not in input schema",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "filter is not valid: property path \"__id__\" does not exist",
		},
		{
			name: "BAD: Destination/API/User - filter refers to a meta property in input schema",
			pipeline: PipelineToSet{
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
					{Name: "__id__", Type: types.String(), ReadOptional: true},
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     `input pipeline schema property "__id__" is a meta property`,
		},
		{
			name: "BAD: Destination/FileStorage/User - no input schema",
			pipeline: PipelineToSet{
				Name:      "Export users",
				OutSchema: types.Type{},
				Format:    "csv",
				Path:      "my_output_users.csv",
				OrderBy:   "email",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.FileStorage,
			formatType:              state.File,
			formatTargets:           state.UsersFlag,
			formatHasSettings:       false,
			formatHasSheets:         false,
			err:                     "input schema must be valid when exporting profiles to file",
		},
		{
			name: "BAD: Destination/API/User - output matching property transformed using mapping",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "address", Type: types.Object([]types.Property{
						{Name: "email_out", Type: types.String()},
					})},
					{Name: "first_name", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name":        "first_name",
						"address.email_out": "email_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "address.email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - output matching property, with simple name, transformed with mapping",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - parent of an output matching property transformed with mapping",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
					{Name: "address_in", Type: types.Map(types.String()), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "address", Type: types.Object([]types.Property{
						{Name: "email_out", Type: types.String()},
					})},
					{Name: "first_name", Type: types.String()},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"first_name": "first_name",
						"address":    "address_in",
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "email_in",
					Out: "address.email_out",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - output matching property transformed with function",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "personal", Type: types.Object([]types.Property{
						{Name: "id", Type: types.Int(32)},
					})},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`        "personal.id": user["in"]`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in", "id"},
						OutPaths: []string{"email_out", "personal.id"},
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "id",
					Out: "personal.id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - output matching property, with simple name, transformed with function",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email_out", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - parent of an output matching property transformed with with function",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "id", Type: types.Int(32), ReadOptional: true},
					{Name: "name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "personal", Type: types.Object([]types.Property{
						{Name: "id", Type: types.Int(32)},
						{Name: "name", Type: types.String()},
					})},
				}),
				Transformation: &Transformation{
					Function: &TransformationFunction{
						Language: "Python",
						Source: strings.Join([]string{
							`def transform(user: dict) -> dict:`,
							`    return {`,
							`        "email_out": user["email_in"],`,
							`        "personal": { "name" : user["name"] }`,
							`    }`}, "\n"),
						InPaths:  []string{"email_in", "name"},
						OutPaths: []string{"email_out", "personal"},
					},
				},
				ExportMode: CreateOrUpdate,
				Matching: Matching{
					In:  "id",
					Out: "personal.id",
				},
				UpdateOnDuplicates: false,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - output matching property not in out schema",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "output matching property \"email_out\" not found within the output schema",
		},
		{
			name: "BAD: Source/Database/User - filters are not allowed",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "filters are not allowed",
		},
		{
			name: "BAD: Source/SDK/Event - cannot provide input schema",
			pipeline: PipelineToSet{
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
					{Name: "anonymousId", Type: types.String()},
				}),
				Name: "Import events into the data warehouse",
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "input schema must be invalid for pipelines that import events into data warehouse",
		},
		{
			name: "BAD: Source/SDK/Event - cannot provide output schema",
			pipeline: PipelineToSet{
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.String()},
				}),
				Name: "Import events into the data warehouse",
			},
			target:                  state.TargetEvent,
			connectionRole:          state.Source,
			connectionConnectorType: state.SDK,
			err:                     "output schema must be invalid when importing events into data warehouse",
		},
		{
			name: "BAD: Source/API/User - InPaths refers to a not-existent second-level property of input schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
					{Name: "additional_properties", Type: types.Object([]types.Property{
						{Name: "a", Type: types.String()},
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "input property \"additional_properties.zzzz\" of transformation function does not exist in schema",
		},
		{
			name: "BAD: Destination/API/User - with mapping transformation that overwrites the out matching property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "mapping cannot map over the output matching property",
		},
		{
			name: "BAD: Destination/API/User - with transformation function that overwrites the out matching property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			provider:                testProvider{},
			err:                     "transformation function cannot transform over the output matching property",
		},
		{
			name: "BAD: Source/API/User - with constant mapping (not allowed) and filter",
			pipeline: PipelineToSet{
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
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "\"a@b\"",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "transformation must map at least one property",
		},
		{
			name: "BAD: Destination/Database/User - only the table key is mapped",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "my_users_table",
				TableKey:  "email_out",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			err:                     "in addition to the table key, there must be at least one other mapped column",
		},
		{
			name: "BAD: Destination/Database/User - only the table key is transformed",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), CreateRequired: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.Database,
			provider:                testProvider{},
			err:                     "the out properties of the transformation function must contain at least one other property in addition to the table key",
		},
		{
			name: "BAD: Source/API/User - unused property in output schema",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "x", Type: types.Object([]types.Property{
						{Name: "y", Type: types.String(), ReadOptional: true},
						{Name: "z", Type: types.String(), ReadOptional: true},
					}), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"x.y": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "output schema contains an unused property: x.z",
		},
		{
			name: "BAD: Source/API/User - with filter but no input schema",
			pipeline: PipelineToSet{
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
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "input schema is required by the filter",
		},
		{
			name: "BAD: Source/API/User - with non-nil mapping but no properties mapped",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{},
				},
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "transformation mapping must have mapped properties",
		},
		{
			name: "BAD: Destination/API/User - missing input matching property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "input matching property cannot be empty if output matching property is not empty",
		},
		{
			name: "BAD: Destination/API/User - missing output matching property",
			pipeline: PipelineToSet{
				Name: "Export users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String(), ReadOptional: true},
					{Name: "first_name", Type: types.String(), ReadOptional: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String()},
					{Name: "first_name", Type: types.String()},
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
			target:                  state.TargetUser,
			connectionRole:          state.Destination,
			connectionConnectorType: state.API,
			err:                     "output matching property cannot be empty if input matching property is not empty",
		},
		{
			name: "BAD: Source/Database/User - identity column is not a valid property name",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is not a valid property name",
		},
		{
			name: "BAD: Source/Database/User - identity column is too long",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "timestamp", Type: types.DateTime()},
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
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
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.Database,
			err:                     "identity column is longer than 1024 runes",
		},
		{
			name: "BAD: Source/API/User - table name is not allowed",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				TableName: "users",
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "table name is not allowed",
		},
		{
			name: "BAD: Source/API/User - update on duplicates is not allowed",
			pipeline: PipelineToSet{
				Name: "Import users",
				InSchema: types.Object([]types.Property{
					{Name: "email_in", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email_out", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &Transformation{
					Mapping: map[string]string{
						"email_out": "email_in",
					},
				},
				UpdateOnDuplicates: true,
			},
			target:                  state.TargetUser,
			connectionRole:          state.Source,
			connectionConnectorType: state.API,
			err:                     "update on duplicates is not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateTestOnPipeline(test.name, test.connectionRole, test.connectionConnectorType, test.target, test.err); err != nil {
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
			err := validatePipelineToSet(test.pipeline, v)
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

// validateTestOnPipeline is used internally by tests to validate the coherence of
// a test on a pipeline.
func validateTestOnPipeline(name string, connectionRole state.Role, connectorType state.ConnectorType, target state.Target, expectedErr string) error {
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
// set of functionalities to be used in the validatePipelineToSet tests.
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
	tests := []struct {
		schema   types.Type
		paths    []string
		expected string
	}{
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
			}),
			paths: []string{"first_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
				{Name: "last_name", Type: types.String()},
			}),
			paths: []string{"first_name", "last_name"},
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
			}),
			expected: "first_name",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
				{Name: "email", Type: types.String()},
			}),
			paths:    []string{"first_name"},
			expected: "email",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
				{Name: "email", Type: types.String()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.String()},
					{Name: "zip_code", Type: types.String()},
				})},
			}),
			paths:    []string{"first_name", "email"},
			expected: "address.street",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
				{Name: "email", Type: types.String()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.String()},
					{Name: "zip_code", Type: types.String()},
				})},
			}),
			paths:    []string{"first_name", "email", "address.street"},
			expected: "address.zip_code",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String()},
				{Name: "email", Type: types.String()},
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street", Type: types.String()},
					{Name: "zip_code", Type: types.String()},
				})},
			}),
			paths:    []string{"first_name", "email", "address.street", "address.zip_code"},
			expected: "",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
			}),
			expected: "x1.y",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
			}),
			paths:    []string{"x1", "x2"},
			expected: "",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
			}),
			paths:    []string{"x1"},
			expected: "x2.y",
		},
		{
			schema: types.Object([]types.Property{
				{Name: "x1", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "y", Type: types.String()},
					{Name: "z", Type: types.String()},
					{Name: "a", Type: types.Object([]types.Property{
						{Name: "b", Type: types.String()},
						{Name: "c", Type: types.String()},
					})},
				})},
			}),
			paths:    []string{"x2.a", "x1"},
			expected: "x2.y",
		},
	}
	for _, test := range tests {
		got, ok := unusedPropertyPath(test.schema, test.paths)
		if (test.expected != "") != ok {
			t.Fatalf("expected %t, got %t", test.expected != "", ok)
		}
		if test.expected != got {
			t.Fatalf("expected %q, got %q", test.expected, got)
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
				{Name: "x", Type: types.String()},
				{Name: "xy", Type: types.String()},
				{Name: "x_y", Type: types.String()},
				{Name: "x_z", Type: types.String()},
				{Name: "z", Type: types.String()},
			}),
			paths: []string{"x_y", "x", "xy", "x_z", "z"},
		},
		{
			name: "Paths cannot be nil",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
			}),
			paths:         nil,
			expectedError: "input properties of transformation function cannot be null",
		},
		{
			name: "Paths cannot be empty",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
			}),
			paths:         []string{},
			expectedError: "there are no input properties in transformation function",
		},
		{
			name: "Paths can be empty when dispatching events",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
			}),
			paths:                []string{},
			dispatchEventsToApps: true,
		},
		{
			name: "Referencing a second-level property",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
			}),
			paths: []string{"x.a"},
		},
		{
			name: "Duplicated property path",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
			}),
			paths:         []string{"x.a", "x.a"},
			expectedError: "transformation function input property path \"x.a\" is repeated",
		},
		{
			name: "Two identical paths (first level)",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.String()},
			}),
			paths:         []string{"x", "x"},
			expectedError: "transformation function input property path \"x\" is repeated",
		},
		{
			name: "Two identical paths (second level)",
			io:   "input",
			schema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
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
					{Name: "a", Type: types.String()},
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
					{Name: "b", Type: types.String()},
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
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
					{Name: "b", Type: types.String()},
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
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
					{Name: "b", Type: types.String()},
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
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
					{Name: "b", Type: types.String()},
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
					})},
					{Name: "array", Type: types.Array(types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
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
					{Name: "b", Type: types.String()},
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
					})},
					{Name: "map", Type: types.Map(types.Object([]types.Property{
						{Name: "f", Type: types.String()},
						{Name: "g", Type: types.String()},
						{Name: "h", Type: types.String()},
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
				{Name: "x", Type: types.String()},
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
