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

// Error represents an error resulting from a transformation, such as a syntax
// error in the function, or the use of a non-existent property.
type Error string

func (err Error) Error() string { return string(err) }

func errorf(format string, a ...any) error {
	return Error(fmt.Sprintf(format, a...))
}

// Transformer represents a transformer.
type Transformer struct {
	inSchema, outSchema types.Type
	mapping             *mappings.Mapping
	transformation      *state.Transformation
	function            Function
	action              int
}

// New returns a new transformer that transforms properties of inSchema to
// outSchema using the given mapping and, in case a transformation is provided,
// also uses such transformation. layouts represents, if not null, the layouts
// used to format DateTime, Date, and Time values as strings.
func New(inSchema, outSchema types.Type, mapping map[string]string, transformation *state.Transformation, action int, function Function, layouts *state.Layouts) (*Transformer, error) {

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
	if mapping != nil {
		var err error
		m.mapping, err = mappings.New(mapping, inSchema, outSchema, layouts)
		if err != nil {
			return nil, err
		}
	}

	return &m, nil
}

// Transform transforms values and returns the result or an error if values
// cannot be transformed.
func (m *Transformer) Transform(ctx context.Context, values map[string]any) (map[string]any, error) {

	// Transform using the mapping.
	if m.mapping != nil {
		return m.mapping.Transform(values)
	}

	// Transform using a function.
	funcName := transformationFunctionName(m.action, m.transformation.Language)
	results, err := m.function.Call(ctx, funcName, m.transformation.Version, m.inSchema, m.outSchema, []map[string]any{values})
	if err != nil {
		if err, ok := err.(*ExecutionError); ok {
			return nil, errorf("%s: %s ", m.transformation.Language.String(), err.Msg)
		}
		return nil, fmt.Errorf("error while execution the transformation: %s", err)
	}
	if err := results[0].Error; err != nil {
		if err, ok := err.(*ExecutionError); ok {
			return nil, errorf("%s: %s ", m.transformation.Language.String(), err.Msg)
		}
		return nil, errorf("%s", err)
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
