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

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"
)

// Purpose represents the purpose of a record transformation.
type Purpose int

const (
	None Purpose = iota
	Create
	Update
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
	path           string
	expr           *Expression
	createRequired bool
	updateRequired bool
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
		p, err := types.PropertyByPath(outSchema, path)
		if err != nil {
			return nil, err
		}
		mappingExpressions[i].path = path
		mappingExpressions[i].expr, err = Compile(expr, inSchema, p.Type, p.Nullable, layouts)
		mappingExpressions[i].createRequired = p.CreateRequired
		mappingExpressions[i].updateRequired = p.UpdateRequired
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
// expressions, sorted alphabetically. The returned properties are guaranteed to
// be unique. If no property are present, it returns an empty slice.
//
// If the expressions contain a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (mapping *Mapping) InProperties() []string {
	p := map[string]struct{}{}
	for _, expr := range mapping.expressions {
		for _, name := range expr.expr.properties {
			p[name] = struct{}{}
		}
	}
	if len(p) == 0 {
		return []string{}
	}
	properties := make([]string, len(p))
	i := 0
	for name := range p {
		properties[i] = name
		i++
	}
	slices.Sort(properties)
	return properties
}

// OutProperties returns the output properties sorted by path.
func (mapping *Mapping) OutProperties() []string {
	properties := make([]string, 0, len(mapping.expressions))
	for _, expr := range mapping.expressions {
		properties = append(properties, expr.path)
	}
	slices.Sort(properties)
	return properties
}

// Transform transforms properties, that must conform to the expression's source
// schema, and returns the result that conforms to the expression's output
// schema.
//
// purpose specifies the reason for the transformation. If Create or Update,
// then all the properties required for creation or the update must be present
// in the returned value.
//
// If the evaluation of an expression results in a void value, the corresponding
// property will not be present in the returned value. If all evaluations of the
// expression result in a void value, an empty map is returned.
//
// If the resulting value cannot be converted to the destination type, it
// returns an error value implementing the ValidationError interface of apis.
//
// Transform might replace JSON properties in the properties map with their
// unmarshalled values.
func (mapping *Mapping) Transform(properties map[string]any, purpose Purpose) (map[string]any, error) {
	out := make(map[string]any, len(mapping.expressions))
	for _, e := range mapping.expressions {
		v, err := e.expr.Eval(properties, purpose)
		if err != nil {
			if err, ok := err.(*invalidConversionError); ok {
				return nil, &validationError{
					path: e.path,
					msg:  err.Error(),
				}
			}
			return nil, err
		}
		if v == Void {
			if e.createRequired && purpose == Create || e.updateRequired && purpose == Update {
				return nil, &validationError{
					path: e.path,
					msg:  "expression is required, but the evaluation returned undefined",
				}
			}
			continue
		}
		storeValue(out, e.path, v)
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
