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

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

// Purpose represents the purpose of a record transformation.
type Purpose int

const (
	None Purpose = iota
	Create
	Update
)

// TransformationError represents an error that occurs when transforming
// properties.
type TransformationError struct {
	msg string
}

func (err TransformationError) Error() string {
	return err.msg
}

// ValidationError represents an error that occurs when validating properties.
type ValidationError struct {
	msg string
}

func (err ValidationError) Error() string {
	return err.msg
}

// Mapping represents a mapping transformer.
type Mapping struct {
	inPlace     bool
	expressions []mappingExpr
}

type mappingExpr struct {
	path           string
	expr           *Expression
	nullable       bool
	createRequired bool
	updateRequired bool
}

// New returns a new mapping that transforms values according to the provided
// expressions. inSchema and outSchema represent the input and output schemas,
// respectively.
//
// If inPlace is true, a transformation is permitted to modify array, object,
// and map values directly within the value being transformed.
//
// If layouts is not nil, it specifies the layouts used to format datetime,
// date, and time values as strings.
//
// The source type can be the invalid type if expressions do not contain paths.
//
// It returns a types.PathNotExistError error if a path in expressions does not
// exist in the source schema.
func New(expressions map[string]string, inSchema, outSchema types.Type, inPlace bool, layouts *state.TimeLayouts) (*Mapping, error) {
	if len(expressions) == 0 {
		return nil, errors.New("there are no expressions")
	}
	if k := inSchema.Kind(); k != types.ObjectKind && k != types.InvalidKind {
		return nil, errors.New("inSchema is not an object and is not the invalid schema")
	}
	if k := outSchema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("outSchema is the invalid schema")
		}
		return nil, errors.New("outSchema is not an object")
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
		mappingExpressions[i].expr, err = Compile(expr, inSchema, p.Type, layouts)
		mappingExpressions[i].nullable = p.Nullable
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
	return &Mapping{expressions: mappingExpressions, inPlace: inPlace}, nil
}

// InPaths returns the input property paths, i.e., the property paths found in
// the expressions, sorted alphabetically. The returned properties are
// guaranteed to be unique. If no property are present, it returns an empty
// slice.
//
// If the expressions contain a map or json indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (mapping *Mapping) InPaths() []string {
	p := map[string]struct{}{}
	for _, expr := range mapping.expressions {
		for _, name := range expr.expr.properties {
			p[name] = struct{}{}
		}
	}
	if len(p) == 0 {
		return []string{}
	}
	paths := make([]string, len(p))
	i := 0
	for path := range p {
		paths[i] = path
		i++
	}
	slices.Sort(paths)
	return paths
}

// OutPaths returns the output property paths sorted by path.
func (mapping *Mapping) OutPaths() []string {
	paths := make([]string, 0, len(mapping.expressions))
	for _, expr := range mapping.expressions {
		paths = append(paths, expr.path)
	}
	slices.Sort(paths)
	return paths
}

// Transform transforms properties, that must conform to the expression's source
// schema, and returns the result that conforms to the expression's output
// schema.
//
// purpose specifies the reason for the transformation. If Create or Update,
// then all the properties required for creation or the update must be present
// in the returned value.
//
// If an expression evaluates to nil and the corresponding property cannot be
// null, that property will be omitted from the returned result, provided this
// is allowed by the purpose.
//
// If an error occurs during property transformation or final validation, a
// TransformationError or ValidationError is returned.
func (mapping *Mapping) Transform(properties map[string]any, purpose Purpose) (map[string]any, error) {
	out := make(map[string]any, len(mapping.expressions))
	for _, e := range mapping.expressions {
		v, err := e.expr.Eval(properties, mapping.inPlace, purpose)
		if err != nil {
			return nil, err
		}
		if v == nil && !e.nullable {
			if e.createRequired && purpose == Create || e.updateRequired && purpose == Update {
				return nil, ValidationError{fmt.Sprintf("required property %q cannot be null", e.path)}
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
