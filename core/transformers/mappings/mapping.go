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
	me := make([]mappingExpr, len(expressions))
	i := 0
	for path, expr := range expressions {
		p, err := types.PropertyByPath(outSchema, path)
		if err != nil {
			return nil, err
		}
		me[i].path = path
		me[i].expr, me[i].properties, err = Compile(expr, inSchema, p.Type)
		me[i].dt = p.Type
		me[i].nullable = p.Nullable
		me[i].createRequired = p.CreateRequired
		me[i].updateRequired = p.UpdateRequired
		me[i].timeLayouts = layouts
		if err != nil {
			return nil, err
		}
		i++
	}
	err := sortMappingExpressions(me)
	if err != nil {
		return nil, err
	}
	return &Mapping{expressions: me, inPlace: inPlace}, nil
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
		v, vt, err := e.expr.Eval(properties)
		if err != nil {
			switch err := err.(type) {
			case TransformationError:
				return nil, TransformationError{fmt.Sprintf("%s while mapping to «%s»", err.msg, code(e.path))}
			case ValidationError:
				return nil, ValidationError{fmt.Sprintf("%s while mapping to «%s»", err.msg, code(e.path))}
			}
			return nil, TransformationError{fmt.Sprintf("%s while mapping to «%s»", err, code(e.path))}
		}
		if v != nil {
			v, err = convert(v, vt, e.dt, true, mapping.inPlace, e.timeLayouts, purpose)
			if err != nil {
				var msg string
				switch err {
				case errRangeConversion:
					msg = fmt.Sprintf("number «%s» is not a «%s» value while mapping to «%s»", code(e.expr.source), e.dt, code(e.path))
				case errMinConversion:
					var n any
					switch e.dt.Kind() {
					case types.IntKind:
						n, _ = e.dt.IntRange()
					case types.UintKind:
						n, _ = e.dt.UintRange()
					case types.FloatKind:
						n, _ = e.dt.FloatRange()
					case types.DecimalKind:
						n, _ = e.dt.DecimalRange()
					}
					msg = fmt.Sprintf("number «%s» is less than %v while mapping to «%s»", code(e.expr.source), n, code(e.path))
				case errMaxConversion:
					var n any
					switch e.dt.Kind() {
					case types.IntKind:
						_, n = e.dt.IntRange()
					case types.UintKind:
						_, n = e.dt.UintRange()
					case types.FloatKind:
						_, n = e.dt.FloatRange()
					case types.DecimalKind:
						_, n = e.dt.DecimalRange()
					}
					msg = fmt.Sprintf("number «%s» is greater than %v while mapping to «%s»", code(e.expr.source), n, code(e.path))
				case errParseConversion:
					var to string
					switch e.dt.Kind() {
					case types.DateTimeKind:
						to = "a date time in ISO 8601 format"
					case types.DateKind:
						to = "a date in ISO 8601 format"
					case types.TimeKind:
						to = "a time in ISO 8601 format"
					case types.UUIDKind:
						to = "a UUID"
					case types.InetKind:
						to = "an IP address"
					}
					msg = fmt.Sprintf("«%s» is not parsable as %s while mapping to «%s»", code(e.expr.source), to, code(e.path))
				case errYearRangeConversion:
					msg = fmt.Sprintf("year of «%s» is not in range [1,9999] while mapping to «%s»", code(e.expr.source), code(e.path))
				case errEnumConversion:
					msg = fmt.Sprintf("«%s» is not one of the allowed values while mapping to «%s»", code(e.expr.source), code(e.path))
				case errRegexpConversion:
					msg = fmt.Sprintf("«%s» does not match «/%s/» while mapping to «%s»", code(e.expr.source), e.dt.Regexp(), code(e.path))
				case errByteLenConversion:
					n, _ := e.dt.ByteLen()
					msg = fmt.Sprintf("«%s» exceeds the %d-byte limit while mapping to «%s»", code(e.expr.source), n, code(e.path))
				case errCharLenConversion:
					n, _ := e.dt.CharLen()
					msg = fmt.Sprintf("«%s» exceeds the %d-char limit while mapping to «%s»", code(e.expr.source), n, code(e.path))
				default:
					msg = fmt.Sprintf("«%s» is not convertible to the «%s» type while mapping to «%s»", code(e.expr.source), e.dt, code(e.path))
				}
				return nil, ValidationError{msg}
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
