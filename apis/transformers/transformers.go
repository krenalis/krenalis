package transformers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers/mappings"
	"github.com/open2b/chichi/types"
)

// Transformer represents a transformer.
type Transformer struct {
	inSchema, outSchema types.Type
	mapping             *mappings.Mapping
	transformation      state.Transformation
	provider            Provider
	action              int
}

// New returns a new transformer that transforms properties from inSchema to
// outSchema using the given transformation for the action with the provided
// identifier. provider is the transformer provider to use for function
// transformations and is nil for mappings. If not nil, layouts represent the
// layouts used to format DateTime, Date, and Time values as strings.
//
// For mappings, it returns a types.PathNotExistError error if a path in
// expressions does not exist in the input schema.
func New(inSchema, outSchema types.Type, transformation state.Transformation, action int, provider Provider, layouts *state.TimeLayouts) (*Transformer, error) {

	if !outSchema.Valid() {
		return nil, errors.New("output schema is not valid")
	}

	m := Transformer{
		inSchema:       inSchema,
		outSchema:      outSchema,
		transformation: transformation,
		provider:       provider,
		action:         action,
	}

	// Mapping.
	if transformation.Mapping != nil {
		var err error
		m.mapping, err = mappings.New(transformation.Mapping, inSchema, outSchema, layouts)
		if err != nil {
			return nil, err
		}
	}

	return &m, nil
}

// Properties returns the properties in the mapping expressions. It calls the
// Properties method of the mapping. See this method for documentation.
// It panics if the transformation is not a mapping but a function.
func (transformer *Transformer) Properties() []string {
	if transformer.mapping == nil {
		panic("cannot get properties of a function transformation")
	}
	return transformer.mapping.Properties()
}

// Transform transforms the values and returns the result. values is expected to
// conform to the input schema. If a validation error occurs during the
// transformation, it returns an error implementing ValidationError of apis.
//
// If the evaluation of an expression results in a void value, the corresponding
// property will not be present in the returned value. If all evaluations of the
// expression result in a void value, an empty map is returned.
//
// For function transformers, it returns the ErrFunctionNotExist error if the
// function does not exist, and a FunctionExecutionError error if an error
// occurs during function execution.
func (transformer *Transformer) Transform(ctx context.Context, values map[string]any) (map[string]any, error) {

	// Transform using the mapping.
	if transformer.mapping != nil {
		return transformer.mapping.Transform(values)
	}

	// Transform using a function.
	funcName := transformationFunctionName(transformer.action, transformer.transformation.Function.Language)
	results, err := transformer.provider.Call(ctx, funcName, transformer.transformation.Function.Version, transformer.inSchema, transformer.outSchema, []map[string]any{values})
	if err != nil {
		if err, ok := err.(FunctionExecutionError); ok {
			return nil, FunctionExecutionError(fmt.Sprintf("%s: %s ", transformer.transformation.Function.Language.String(), err))
		}
		return nil, err
	}
	if err := results[0].Err; err != nil {
		return nil, err
	}
	out := results[0].Value

	return out, nil
}

// TransformValues transforms the provided values and returns the results. The
// values are expected to conform to the input schema. If an error occurs during
// the transformation of a single value, the error is stored in the Err field of
// the corresponding result. If the error is a validation error, it implements
// ValidationError of apis; otherwise it is a FunctionExecutionError error.
//
// For function transformers, it returns the ErrFunctionNotExist error if the
// function does not exist, and a FunctionExecutionError error if an error
// occurs during function execution.
func (transformer *Transformer) TransformValues(ctx context.Context, values []map[string]any) ([]Result, error) {

	// Transform using the mapping.
	if transformer.mapping != nil {
		results := make([]Result, len(values))
		for i, value := range values {
			value, err := transformer.mapping.Transform(value)
			if err != nil {
				results[i].Err = err
				continue
			}
			results[i].Value = value
			if i%100 != 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		return results, nil
	}

	// Transform using a function.
	funcName := transformationFunctionName(transformer.action, transformer.transformation.Function.Language)
	results, err := transformer.provider.Call(ctx, funcName, transformer.transformation.Function.Version, transformer.inSchema, transformer.outSchema, values)
	if err != nil {
		if err, ok := err.(FunctionExecutionError); ok {
			return nil, FunctionExecutionError(fmt.Sprintf("%s: %s ", transformer.transformation.Function.Language.String(), err))
		}
		return nil, err
	}

	return results, nil
}

// transformationFunctionName returns the name the transformation function for
// an action in the specified language.
//
// Keep in sync with the function having the same name in the apis package.
func transformationFunctionName(action int, language state.Language) string {
	var ext string
	switch language {
	case state.JavaScript:
		ext = ".js"
	case state.Python:
		ext = ".py"
	default:
		panic("unexpected language")
	}
	return "action-" + strconv.Itoa(action) + ext
}
