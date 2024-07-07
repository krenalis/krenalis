//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// validationError implements the apis.ValidationError interface.
type validationError struct {
	path string
	msg  string
}

func (err *validationError) Error() string {
	return err.msg
}

func (err *validationError) PropertyPath() string {
	return err.path
}

// Mapping represents a mapping transformer.
type Mapping struct {
	expressions []mappingExpr
}

type mappingExpr struct {
	path string
	expr *Expression
}

// New returns a new mapping that transforms values according to the provided
// expressions. inSchema and outSchema represent the input and output schemas,
// respectively. If layouts is not nil, it specifies the layouts used to format
// DateTime, Date, and Time values as strings.
//
// The source type can be the invalid type if expressions do not contain paths.
// It returns a types.PathNotExistError error if a path in expressions does not
// exist in the source schema.
func New(expressions map[string]string, inSchema, outSchema types.Type, layouts *state.TimeLayouts) (*Mapping, error) {
	if len(expressions) == 0 {
		return nil, errors.New("there are no expressions")
	}
	if k := inSchema.Kind(); k != types.ObjectKind && k != types.InvalidKind {
		return nil, errors.New("inSchema is not an Object and is not the invalid schema")
	}
	if k := outSchema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("outSchema is the invalid schema")
		}
		return nil, errors.New("outSchema is not an Object")
	}
	// Compile the expressions.
	mappingExpressions := make([]mappingExpr, len(expressions))
	i := 0
	for path, expr := range expressions {
		p, err := outSchema.PropertyByPath(path)
		if err != nil {
			return nil, err
		}
		mappingExpressions[i].path = path
		mappingExpressions[i].expr, err = Compile(expr, inSchema, p.Type, p.Required, p.Nullable, layouts)
		if err != nil {
			return nil, err
		}
		i++
	}
	// Sort the expressions based on their paths and ensure that no two paths have the same prefix.
	slices.SortFunc(mappingExpressions, func(a, b mappingExpr) int {
		return cmp.Compare(a.path, b.path)
	})
	for i, expr := range mappingExpressions[1:] {
		if prev := mappingExpressions[i]; strings.HasPrefix(expr.path, prev.path) {
			return nil, fmt.Errorf("paths %q and %q have the same prefix", expr.path, prev.path)
		}
	}
	return &Mapping{expressions: mappingExpressions}, nil
}

// InProperties returns the input properties, i.e., the properties found in the
// expressions, sorted by their appearance order in the expressions. The
// returned properties are guaranteed to be unique. If no property are present,
// it returns nil.
//
// If the expressions contain a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (mapping *Mapping) InProperties() []string {
	var properties []string
	for _, expr := range mapping.expressions {
		properties = appendProperties(properties, expr.expr.parts)
	}
	return properties
}

// OutProperties returns the output properties sorted by path.
func (mapping *Mapping) OutProperties() []string {
	properties := make([]string, len(mapping.expressions))
	for _, expr := range mapping.expressions {
		properties = append(properties, expr.path)
	}
	slices.Sort(properties)
	return properties
}

// Transform transforms value, that must conform to the expression's source
// schema, and returns the result that conforms to the expression's output
// schema.
//
// If the evaluation of an expression results in a void value, the corresponding
// property will not be present in the returned value. If all evaluations of the
// expression result in a void value, an empty map is returned.
//
// If the resulting value cannot be converted to the destination type, it
// returns an error value implementing the ValidationError interface of apis.
func (mapping *Mapping) Transform(value map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(mapping.expressions))
	for _, e := range mapping.expressions {
		v, err := e.expr.Eval(value)
		if err != nil {
			if err, ok := err.(*invalidConversionError); ok {
				return nil, &validationError{
					path: e.path,
					msg:  err.Error(),
				}
			}
			return nil, err
		}
		if v != Void {
			storeValue(out, e.path, v)
		}
	}
	return out, nil
}

// storeValue stores v in value at the given path.
func storeValue(value map[string]any, path string, v any) {
	var ok bool
	var name string
	for {
		name, path, ok = strings.Cut(path, ".")
		if !ok {
			value[name] = v
			break
		}
		object, ok := value[name].(map[string]any)
		if !ok {
			object = map[string]any{}
			value[name] = object
		}
		value = object
	}
}
