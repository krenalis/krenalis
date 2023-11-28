package transformers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"chichi/apis/state"
	"chichi/apis/transformers/mappings"
	"chichi/connector/types"
)

// Transformer represents a transformer.
type Transformer struct {
	inSchema, outSchema types.Type
	mapping             *mappings.Mapping
	transformation      state.Transformation
	function            Function
	action              int
}

// New returns a new transformer that transforms properties of inSchema to
// outSchema using the given mapping and, in case a transformation is provided,
// also uses such transformation. layouts represents, if not null, the layouts
// used to format DateTime, Date, and Time values as strings.
func New(inSchema, outSchema types.Type, transformation state.Transformation, action int, function Function, layouts *state.Layouts) (*Transformer, error) {

	if !outSchema.Valid() {
		return nil, errors.New("output schema is not valid")
	}

	m := Transformer{
		inSchema:       inSchema,
		outSchema:      outSchema,
		transformation: transformation,
		function:       function,
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

// Transform transforms values and returns the result or an error if values
// cannot be transformed.
func (transformer *Transformer) Transform(ctx context.Context, values map[string]any) (map[string]any, error) {

	// Transform using the mapping.
	if transformer.mapping != nil {
		return transformer.mapping.Transform(values)
	}

	// Transform using a function.
	funcName := transformationFunctionName(transformer.action, transformer.transformation.Function.Language)
	results, err := transformer.function.Call(ctx, funcName, transformer.transformation.Function.Version, transformer.inSchema, transformer.outSchema, []map[string]any{values})
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
