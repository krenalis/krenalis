// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// Purpose represents the purpose of a record transformation.
type Purpose int

const (
	None Purpose = iota
	Create
	Update
)

// TransformationError represents an error that occurs when transforming
// attributes.
type TransformationError struct {
	msg string
}

func (err TransformationError) Error() string {
	return err.msg
}

// ValidationError represents an error that occurs when validating attributes.
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
	outSchema   types.Type
}

type mappingExpr struct {
	path           string
	expr           *Expression
	properties     []string
	dt             types.Type // destination type.
	nullable       bool
	createRequired bool
	updateRequired bool
	timeLayouts    *state.TimeLayouts
}

// New returns a new mapping that transforms values according to the provided
// expressions. inSchema and outSchema represent the input and output schemas,
// respectively.
//
// If inPlace is true, a transformation is permitted to modify array, object,
// and map values directly within the value being transformed. Conversions that
// change a container's type or format temporal values copy containers to
// preserve source values shared by multiple expressions or elements.
//
// If layouts is not nil, it specifies the layouts used to format datetime,
// date, and time values as strings.
//
// The source type can be the invalid type if expressions do not contain paths.
// Expressions are compiled in alphabetical order of their destination paths.
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
	// Sort and validate the destination paths before compiling expressions.
	me := make([]mappingExpr, len(expressions))
	i := 0
	for path := range expressions {
		me[i].path = path
		i++
	}
	err := sortMappingExpressions(me)
	if err != nil {
		return nil, err
	}
	properties := outSchema.Properties()
	for i := range me {
		path := me[i].path
		p, err := properties.ByPath(path)
		if err != nil {
			return nil, err
		}
		me[i].expr, me[i].properties, err = Compile(expressions[path], inSchema, p.Type)
		if err != nil {
			return nil, err
		}
		me[i].dt = p.Type
		me[i].nullable = p.Nullable
		me[i].createRequired = p.CreateRequired
		me[i].updateRequired = p.UpdateRequired
		me[i].timeLayouts = layouts
	}
	return &Mapping{expressions: me, inPlace: inPlace, outSchema: outSchema}, nil
}

// InPaths returns the input property paths, i.e., the property paths found in
// the expressions, sorted alphabetically. The returned properties are
// guaranteed to be unique. If no properties are present, it returns an empty
// slice.
//
// If the expressions contain map or JSON indexing, InPaths does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (mapping *Mapping) InPaths() []string {
	p := map[string]struct{}{}
	for _, expr := range mapping.expressions {
		for _, name := range expr.properties {
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

// Transform transforms attributes that must conform to the mapping's source
// schema and returns a result conforming to its output schema.
//
// purpose specifies the reason for the transformation. If Create or Update,
// then all the properties required for creation or the update must be present
// in the returned value.
//
// If an expression evaluates to nil and the corresponding property cannot be
// null, a JSON property receives JSON null; other properties are omitted from
// the returned result, provided this is allowed by the purpose.
//
// If an error occurs during attribute transformation or final validation, a
// TransformationError or ValidationError is returned.
func (mapping *Mapping) Transform(attributes map[string]any, purpose Purpose) (map[string]any, error) {
	out := make(map[string]any, len(mapping.expressions))
	for _, e := range mapping.expressions {
		v, vt, err := e.expr.Eval(attributes)
		if err != nil {
			switch err := err.(type) {
			case TransformationError:
				return nil, TransformationError{fmt.Sprintf("%s while mapping to «%s»", err.msg, code(e.path))}
			case ValidationError:
				return nil, ValidationError{fmt.Sprintf("%s while mapping to «%s»", err.msg, code(e.path))}
			}
			return nil, TransformationError{fmt.Sprintf("%s while mapping to «%s»", err, code(e.path))}
		}
		if v != nil || (!e.nullable && e.dt.Kind() == types.JSONKind) {
			// A nil result for a non-nullable JSON property must become JSON null.
			nullable := v != nil
			v, err = convert(v, vt, e.dt, nullable, mapping.inPlace, e.timeLayouts, purpose)
			if err != nil {
				err = errValidationConversion(err, code(e.expr.source), e.dt)
				return nil, ValidationError{fmt.Sprintf("%s while mapping to «%s»", err, code(e.path))}
			}
		}
		if v == nil && !e.nullable {
			if e.createRequired && purpose == Create {
				return nil, ValidationError{fmt.Sprintf("«%s» is null but it is required for creation while mapping to «%s»", code(e.expr.source), code(e.path))}
			} else if e.updateRequired && purpose == Update {
				return nil, ValidationError{fmt.Sprintf("«%s» is null but it is required for update while mapping to «%s»", code(e.expr.source), code(e.path))}
			}
			continue
		}
		storeValue(out, e.path, v)
	}
	// Paths such as parent.x construct objects after their values are converted.
	// Check their required properties without reconverting formatted time values.
	if purpose == Create || purpose == Update {
		err := validateRequired(out, mapping.outSchema, purpose, "")
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// sortMappingExpressions sorts me by path and returns an error if any two paths
// are equal or if one is a prefix of another, immediately followed by a dot.
func sortMappingExpressions(me []mappingExpr) error {
	slices.SortFunc(me, func(a, b mappingExpr) int {
		return cmp.Compare(a.path, b.path)
	})
	for i, expr := range me[1:] {
		if prev := me[i]; strings.HasPrefix(expr.path, prev.path) &&
			(len(expr.path) == len(prev.path) || expr.path[len(prev.path)] == '.') {
			return fmt.Errorf("paths %q and %q have the same prefix", prev.path, expr.path)
		}
	}
	return nil
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

// validateRequired checks the presence of required properties in v and its
// containers according to typ and purpose. It returns a ValidationError with
// the missing property's path. Absent or null optional ancestors do not require
// their descendants. Scalar values may already have been formatted for output.
func validateRequired(v any, typ types.Type, purpose Purpose, path string) error {
	if v == nil {
		return nil
	}
	switch typ.Kind() {
	case types.ObjectKind:
		object := v.(map[string]any)
		for _, p := range typ.Properties().All() {
			propertyPath := p.Name
			if path != "" {
				propertyPath = path + "." + p.Name
			}
			value, ok := object[p.Name]
			if !ok {
				if purpose == Create && p.CreateRequired {
					return ValidationError{fmt.Sprintf("«%s» is missing but it is required for creation", code(propertyPath))}
				}
				if purpose == Update && p.UpdateRequired {
					return ValidationError{fmt.Sprintf("«%s» is missing but it is required for update", code(propertyPath))}
				}
				continue
			}
			err := validateRequired(value, p.Type, purpose, propertyPath)
			if err != nil {
				return err
			}
		}
	case types.ArrayKind:
		for i, value := range v.([]any) {
			err := validateRequired(value, typ.Elem(), purpose, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return err
			}
		}
	case types.MapKind:
		for key, value := range v.(map[string]any) {
			err := validateRequired(value, typ.Elem(), purpose, fmt.Sprintf("%s[%q]", path, key))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
