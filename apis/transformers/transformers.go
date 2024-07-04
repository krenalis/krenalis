package transformers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers/mappings"
	"github.com/open2b/chichi/types"
)

// Transformer represents a transformer.
type Transformer struct {
	inSchema  types.Type
	outSchema types.Type
	mapping   *mappings.Mapping
	function  *state.TransformationFunction
	action    int
	provider  Provider
}

// New returns a new transformer that transforms properties from inSchema to
// outSchema using the given transformation for the action with the provided
// identifier. provider is the transformer provider to use for function
// transformations and is nil for mappings. If not nil, layouts represent the
// layouts used to format DateTime, Date, and Time values as strings.
//
// For functions, only the property names listed in 'transformation.Function'
// are processed from the given schemas.
func New(inSchema, outSchema types.Type, transformation state.Transformation, action int, provider Provider, layouts *state.TimeLayouts) *Transformer {

	transformer := Transformer{
		action:   action,
		provider: provider,
	}

	if m := transformation.Mapping; m != nil {
		transformer.inSchema = inSchema
		transformer.outSchema = outSchema
		mapping, err := mappings.New(transformation.Mapping, inSchema, outSchema, layouts)
		if err != nil {
			panic(fmt.Sprintf("unexpected error building a mapping: %s", err))
		}
		transformer.mapping = mapping
	} else if fn := transformation.Function; fn != nil {
		transformer.function = fn
		if len(fn.InProperties) > 0 {
			transformer.inSchema = schemaSubset(inSchema, fn.InProperties)
		}
		transformer.outSchema = schemaSubset(outSchema, fn.OutProperties)
	} else {
		panic(errors.New("there is no transformation"))
	}

	return &transformer
}

// InProperties returns the input properties of the transformer.
//
// For functions, it returns the property paths. If the transformation involves
// dispatching events to apps, the returned slice may be empty. In all other
// cases, it is never empty.
//
// For mappings, it returns the properties found in the expression, sorted by
// their appearance order in the expressions. The returned properties are
// guaranteed to be unique. If no property are present, it returns nil.
//
// If the expressions contain a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (transformer *Transformer) InProperties() []string {
	if transformer.mapping != nil {
		return transformer.mapping.InProperties()
	}
	return slices.Clone(transformer.function.InProperties)
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
	funcName := transformationFunctionName(transformer.action, transformer.function.Language)
	results, err := transformer.provider.Call(ctx, funcName, transformer.function.Version, transformer.inSchema, transformer.outSchema, []map[string]any{values})
	if err != nil {
		if err, ok := err.(FunctionExecutionError); ok {
			return nil, FunctionExecutionError(fmt.Sprintf("%s: %s ", transformer.function.Language.String(), err))
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
	funcName := transformationFunctionName(transformer.action, transformer.function.Language)
	results, err := transformer.provider.Call(ctx, funcName, transformer.function.Version, transformer.inSchema, transformer.outSchema, values)
	if err != nil {
		if err, ok := err.(FunctionExecutionError); ok {
			return nil, FunctionExecutionError(fmt.Sprintf("%s: %s ", transformer.function.Language.String(), err))
		}
		return nil, err
	}

	return results, nil
}

// OutProperties returns the output properties of the transformer.
// The properties are sorted by their path, and there is at least one property.
func (transformer *Transformer) OutProperties() []string {
	if transformer.mapping != nil {
		return transformer.mapping.OutProperties()
	}
	return slices.Clone(transformer.function.OutProperties)
}

// schemaSubset returns a subset of schema containing only the properties
// specified in properties, preserving their original order in schema.
// The parameter io specifies whether the operation relates to "input" or
// "output" and is used solely for error messages.
// This function panics if schema is not an object type.
func schemaSubset(schema types.Type, properties []string) types.Type {
	has := make(map[string]struct{}, len(properties))
	for _, name := range properties {
		has[name] = struct{}{}
	}
	return types.SubsetFunc(schema, func(p types.Property) bool {
		_, ok := has[p.Name]
		return ok
	})
}

// transformationFunctionName returns the name of the transformation function
// for an action in the specified language.
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
