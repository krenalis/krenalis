// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	"github.com/meergo/meergo/tools/prometheus"
	"github.com/meergo/meergo/tools/types"
)

// Purpose represents the purpose of a record transformation.
type Purpose int

const (
	Import Purpose = iota // used when importing a user or group into the data warehouse.
	Create                // used when creating a user or group in an application or sending an event to an application.
	Update                // used when updating a user or group in an application.
)

// Record represents a record to transform.
type Record struct {
	Purpose    Purpose // defaults to Import.
	Attributes map[string]any
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
	pipeline  int
	provider  FunctionProvider
	inSchema  types.Type
	outSchema types.Type
	mapping   *mappings.Mapping
	function  *state.TransformationFunction
}

// New returns a new transformer that transforms values for the provided
// pipeline. provider is the transformer provider used for transformation
// functions and should be nil for mappings. layouts, if not nil, represents the
// layouts used to format datetime, date, and time values as strings.
//
// It only accesses the ID, InSchema, OutSchema, and Transformation fields of
// pipeline.
//
// It returns a types.PathNotExistError error if a path in the mapping does not
// exist in the source schema.
func New(pipeline *state.Pipeline, provider FunctionProvider, layouts *state.TimeLayouts) (*Transformer, error) {

	if m := pipeline.Transformation.Mapping; m != nil {
		inPlace := pipeline.Target != state.TargetEvent
		mapping, err := mappings.New(m, pipeline.InSchema, pipeline.OutSchema, inPlace, layouts)
		if err != nil {
			return nil, err
		}
		t := Transformer{
			pipeline:  pipeline.ID,
			inSchema:  pipeline.InSchema,
			outSchema: pipeline.OutSchema,
			mapping:   mapping,
		}
		// Set CreateRequired to true for the output schema first level properties of a destination database.
		if isDestinationDatabase := pipeline.TableName != ""; isDestinationDatabase {
			t.outSchema = setCreateRequired(t.outSchema)
		}
		return &t, nil
	}

	if f := pipeline.Transformation.Function; f != nil {
		t := Transformer{
			pipeline:  pipeline.ID,
			provider:  provider,
			outSchema: schemaSubset(pipeline.OutSchema, pipeline.Transformation.OutPaths),
			function:  f,
		}
		if len(pipeline.Transformation.InPaths) > 0 {
			t.inSchema = schemaSubset(pipeline.InSchema, pipeline.Transformation.InPaths)
		}
		// Set CreateRequired to true for the output schema first level properties of a destination database.
		if isDestinationDatabase := pipeline.TableName != ""; isDestinationDatabase {
			t.outSchema = setCreateRequired(t.outSchema)
		}
		return &t, nil
	}

	return nil, errors.New("there is no transformation")
}

// Transform transforms the provided records and updates their attributes.
// Record attributes, before transformation, are expected to conform to the
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

	prometheus.Increment("Transformer.Transform.calls", 1)
	prometheus.Increment("Transformer.Transform.passed_records", len(records))

	// Transform using the mapping.
	if t.mapping != nil {
		for i, record := range records {
			attributes, err := t.mapping.Transform(record.Attributes, mappings.Purpose(record.Purpose))
			if err != nil {
				switch e := err.(type) {
				case mappings.TransformationError:
					err = RecordTransformationError{msg: e.Error()}
				case mappings.ValidationError:
					err = RecordValidationError{msg: e.Error()}
				default:
					return err
				}
				record.Attributes = nil
				records[i].Err = err
				continue
			}
			records[i].Attributes = attributes
			if i%100 != 0 {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
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
		if err == errInvalidResponseFormat {
			err = errors.New("cannot execute transformation: transformer has unexpectedly returned an invalid JSON response")
		}
		return err
	}

	return nil
}

// schemaSubset returns a subset of schema containing only the property paths
// specified in paths, preserving their original order and upper hierarchy
// in schema. paths must be alphabetically ordered. This function panics if
// schema is not an object type.
func schemaSubset(schema types.Type, paths []string) types.Type {
	return types.Prune(schema, func(path string) bool {
		for {
			if _, ok := slices.BinarySearch(paths, path); ok {
				return true
			}
			i := strings.LastIndexByte(path, '.')
			if i == -1 {
				return false
			}
			path = path[:i]
		}
	})
}

// setCreateRequired returns a copy of schema with all first-level properties'
// CreateRequired attribute set to true.
func setCreateRequired(schema types.Type) types.Type {
	properties := schema.Properties().Slice()
	for i := range properties {
		properties[i].CreateRequired = true
	}
	return types.Object(properties)
}
