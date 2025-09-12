//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"context"
	"errors"
	"fmt"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	meergoMetrics "github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/types"
)

// Purpose represents the purpose of a record transformation.
type Purpose int

const (
	Import Purpose = iota // used when importing a user or group into the data warehouse.
	Create                // used when creating a user or group in an app or sending an event to an app.
	Update                // used when updating a user or group in an app.
)

// Record represents a record to transform.
type Record struct {
	Purpose    Purpose // defaults to Import.
	Properties map[string]any
	Err        error
}

// RecordTransformationError represents an error that occurs when transforming a
// record.
type RecordTransformationError struct {
	msg string
}

func (err RecordTransformationError) Error() string {
	return err.msg
}

// Transformer represents a transformer.
type Transformer struct {
	action    int
	provider  FunctionProvider
	inSchema  types.Type
	outSchema types.Type
	mapping   *mappings.Mapping
	function  *state.TransformationFunction
}

// New returns a new transformer that transforms values for the provided action.
// provider is the transformer provider used for transformation functions and
// should be nil for mappings. layouts, if not nil, represents the layouts used
// to format datetime, date, and time values as strings.
//
// It only accesses the ID, InSchema, OutSchema, and Transformation fields of
// action.
//
// It returns a types.PathNotExistError error if a path in the mapping does not
// exist in the source schema.
func New(action *state.Action, provider FunctionProvider, layouts *state.TimeLayouts) (*Transformer, error) {

	if m := action.Transformation.Mapping; m != nil {
		inPlace := action.Target != state.TargetEvent
		mapping, err := mappings.New(m, action.InSchema, action.OutSchema, inPlace, layouts)
		if err != nil {
			return nil, err
		}
		t := Transformer{
			action:    action.ID,
			inSchema:  action.InSchema,
			outSchema: action.OutSchema,
			mapping:   mapping,
		}
		// Set CreateRequired to true for the output schema first level properties of a destination database.
		if isDestinationDatabase := action.TableName != ""; isDestinationDatabase {
			t.outSchema = setCreateRequired(t.outSchema)
		}
		return &t, nil
	}

	if f := action.Transformation.Function; f != nil {
		t := Transformer{
			action:    action.ID,
			provider:  provider,
			outSchema: schemaSubset(action.OutSchema, action.Transformation.OutPaths),
			function:  f,
		}
		if len(action.Transformation.InPaths) > 0 {
			t.inSchema = schemaSubset(action.InSchema, action.Transformation.InPaths)
		}
		// Set CreateRequired to true for the output schema first level properties of a destination database.
		if isDestinationDatabase := action.TableName != ""; isDestinationDatabase {
			t.outSchema = setCreateRequired(t.outSchema)
		}
		return &t, nil
	}

	return nil, errors.New("there is no transformation")
}

// Transform transforms the provided records and updates their properties.
// Record properties, before transformation, are expected to conform to the
// input schema.
//
// If an error occurs during the transformation of a single record, either a
// RecordTransformationError or RecordValidationError is stored in the Err field
// of the corresponding record.
//
// For function transformers, if the function does not exist, the method returns
// ErrFunctionNotExist, and if an error occurs during function execution, it
// returns a FunctionExecError.
func (t *Transformer) Transform(ctx context.Context, records []Record) error {

	meergoMetrics.Increment("Transformer.Transform.calls", 1)
	meergoMetrics.Increment("Transformer.Transform.passed_records", len(records))

	// Transform using the mapping.
	if t.mapping != nil {
		for i, record := range records {
			properties, err := t.mapping.Transform(record.Properties, mappings.Purpose(record.Purpose))
			if err != nil {
				switch e := err.(type) {
				case mappings.TransformationError:
					err = RecordTransformationError{msg: e.Error()}
				case mappings.ValidationError:
					err = RecordValidationError{msg: e.Error()}
				default:
					return err
				}
				record.Properties = nil
				records[i].Err = err
				continue
			}
			records[i].Properties = properties
			if i%100 != 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		return nil
	}

	// Transform using the function.
	fn := t.function
	err := t.provider.Call(ctx, fn.ID, fn.Version, t.inSchema, t.outSchema, fn.PreserveJSON, records)
	if err != nil {
		if err, ok := err.(FunctionExecError); ok {
			err.msg = fmt.Sprintf("%s: %s ", t.function.Language.String(), err.msg)
		}
		return err
	}

	return nil
}

// schemaSubset returns a subset of schema containing only the property paths
// specified in properties, preserving their original order and upper hierarchy
// in schema. This function panics if schema is not an object type.
func schemaSubset(schema types.Type, paths []string) types.Type {
	has := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		has[path] = struct{}{}
	}
	return types.SubsetByPathFunc(schema, func(path string) bool {
		_, ok := has[path]
		return ok
	})
}

// setCreateRequired returns a copy of schema with all first-level properties'
// CreateRequired attribute set to true.
func setCreateRequired(schema types.Type) types.Type {
	properties := types.Properties(schema)
	for i := 0; i < len(properties); i++ {
		properties[i].CreateRequired = true
	}
	return types.Object(properties)
}
