//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// validationError implements the ValidationError interface of apis.
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

// Void represents the void value.
var Void = struct{}{}

// errVoid is returned by the 'when' function when the first argument is false,
// and in this case the destination property is not changed.
var errVoid = errors.New("void")

// invalidConversionError is the error returned by the Eval and Transform
// methods of when a value resulted from an evaluation cannot be converted to
// the destination type.
type invalidConversionError struct {
	value           any
	sourceType      types.Type
	destinationType types.Type
}

func (err *invalidConversionError) Error() string {
	switch err.value {
	case nil:
		return "cannot convert null to a non-nullable value"
	case Void:
		return "expression is required, but the evaluation returned no value"
	}
	return fmt.Sprintf("cannot convert %#v (type %s) to type %s", err.value, err.sourceType, err.destinationType)
}

// Expression represents a mapping expression used to transform data from a
// source to a destination. An Expression can contain strings, numbers, true,
// false, null, property paths and function calls.
type Expression struct {
	parts       []part     // expression parts.
	dt          types.Type // destination type.
	required    bool       // reports whether the resulting value is required and consequently it cannot be void.
	nullable    bool       // reports whether the resulting value can be nil.
	timeLayouts *state.TimeLayouts
}

// path represents a property path or a function name.
// See 'part.path' for documentation.
type path []string

// part represents an expression part within an Expression. An expression part
// can take different forms:
//
//   - value             example: "foo"
//   - path              example: x
//   - path(args)        example: add(x, 5)
//   - value path        example: 5 a.b
//   - value path(args)  example: "foo" coalesce(a.b, c)
//
// For instance, the Expression `"foo" x " " true a.b` is parsed as `"foo" x`,
// `" true" a.b`.
type part struct {
	// Value. If there is a path, value, if present, can only be of type Text.
	value any

	// Path or function name.
	// If it represents a function name, it consists of only the function name.
	// Otherwise, path elements follow these rules:
	//   - If it represents a map or JSON key, it starts with ':'.
	//   - If it was denoted with an indexing (e.g., a["b"]), it is enclosed in '[' and ']'.
	//   - If it was denoted by '?', it ends with '?'.
	// Examples of path elements: "x", "[x]" ":x", ":[$a]", ":x?", ":[x]?".
	path path

	// Function call arguments.
	args [][]part

	// If there is a path, it represents the type of the property or the type of the function call.
	// Otherwise, it represents the type of the value. For some function calls, as coalesce, it is
	// the invalid type, indicating that the call can return different types.
	typ types.Type
}

// Compile parses a map expression and returns an Expression value that can be
// used to execute the expression.
//
// schema is the schema of the paths in the expression, dt is the destination
// type, required indicates whether the result value of the evaluation is
// required (cannot be void), nullable indicates whether that value can be nil,
// and layouts represents, if not null, the layouts used to format DateTime,
// Date, and Time values as strings.
//
// An invalid schema can be passed to compile an expression without paths.
func Compile(expr string, schema types.Type, dt types.Type, required, nullable bool, layouts *state.TimeLayouts) (*Expression, error) {
	if expr == "" {
		return nil, errors.New("expression is empty")
	}
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("schema is non an object")
	}
	if !dt.Valid() {
		return nil, errors.New("destination type is not valid")
	}
	parts, src, err := parseExpression(expr)
	if err != nil {
		return nil, err
	}
	if src != "" {
		return nil, fmt.Errorf("unexpected character %v", strconv.QuoteRuneToGraphic(rune(src[0])))
	}
	err = typeCheck(parts, schema, dt, required, nullable)
	if err != nil {
		return nil, err
	}
	expression := &Expression{
		parts:       parts,
		dt:          dt,
		required:    required,
		nullable:    nullable,
		timeLayouts: layouts,
	}
	return expression, nil
}

// Eval evaluates the expression using the provided values for the properties,
// which must conform to the expression's source type, and returns the result
// that conforms to the expression's destination type or Void if the result is
// void.
func (expr *Expression) Eval(values map[string]any) (any, error) {
	v, st, err := eval(expr.parts, values, expr.timeLayouts)
	if err != nil {
		if err == errVoid {
			if !expr.required {
				return Void, nil
			}
			err = &invalidConversionError{Void, st, expr.dt}
		}
		return nil, err
	}
	if v != nil || !expr.nullable {
		c, err := convert(v, st, expr.dt, expr.nullable, expr.timeLayouts)
		if err != nil {
			if err == errInvalidConversion {
				err = &invalidConversionError{v, st, expr.dt}
			}
			return nil, err
		}
		v = c
	}
	return v, err
}

// Properties returns the properties found in the expression, sorted by their
// appearance order in the expression. The returned properties are guaranteed to
// be unique. If no property are present, it returns nil.
//
// If the expression contains a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (expr *Expression) Properties() []string {
	return appendProperties(nil, expr.parts)
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
// expressions. st and dt represent the source and destination types,
// respectively. If layouts is not null, it specifies the layouts used to
// format DateTime, Date, and Time values as strings.
//
// The source type can be the invalid type if expressions do not contain paths.
// It returns a types.PathNotExistError error if a path in expressions does not
// exist in the source schema.
func New(expressions map[string]string, st, dt types.Type, layouts *state.TimeLayouts) (*Mapping, error) {
	if len(expressions) == 0 {
		return nil, errors.New("there are no expressions")
	}
	if k := st.Kind(); k != types.ObjectKind && k != types.InvalidKind {
		return nil, errors.New("source is not an object and is not the invalid type")
	}
	if k := dt.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("destination type is the invalid type")
		}
		return nil, errors.New("destination type is not an object")
	}
	// Compile the expressions.
	mappingExpressions := make([]mappingExpr, len(expressions))
	i := 0
	for path, expr := range expressions {
		p, err := dt.PropertyByPath(path)
		if err != nil {
			return nil, err
		}
		mappingExpressions[i].path = path
		mappingExpressions[i].expr, err = Compile(expr, st, p.Type, p.Required, p.Nullable, layouts)
		if err != nil {
			return nil, err
		}
		i++
	}
	// Sort the expressions based on their paths
	// and ensure that no two paths have the same prefix.
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

// Properties returns the properties found in the expressions, sorted by their
// appearance order in the expressions. The returned properties are guaranteed
// to be unique. If no property are present, it returns nil.
//
// If the expressions contain a map or JSON indexing, Properties does not return
// the key. For example, for the expression x.y.z, it returns {"x"} if x is a
// JSON object, and returns {"x.z"} if x is a map of objects.
func (mapping *Mapping) Properties() []string {
	var properties []string
	for _, expr := range mapping.expressions {
		properties = appendProperties(properties, expr.expr.parts)
	}
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

// appendProperties appends the properties from expression to properties, if
// they do not already exist in it.
func appendProperties(properties []string, expression []part) []string {
	for _, expr := range expression {
		if expr.path == nil {
			continue
		}
		if expr.args == nil {
			var b strings.Builder
			for _, name := range expr.path {
				if name[0] != ':' {
					if name[0] == '[' {
						name = name[1 : len(name)-1]
					}
					if b.Len() > 0 {
						b.WriteByte('.')
					}
					b.WriteString(name)
				}
			}
			property := b.String()
			if !slices.Contains(properties, property) {
				properties = append(properties, property)
			}
			continue
		}
		for _, arg := range expr.args {
			properties = appendProperties(properties, arg)
		}
	}
	return properties
}

// eval evaluates expression and returns its value and type. values contains the
// property values. layouts represents, if not null, the layouts used to
// format DateTime, Date, and Time values as strings.
//
// If the result of the evaluation is void, it returns the errVoid error.
func eval(expression []part, values map[string]any, layouts *state.TimeLayouts) (any, types.Type, error) {

	// Evaluate the most common cases that does not require a buffer.
	if len(expression) == 1 {
		p := expression[0]
		if p.path == nil {
			return p.value, p.typ, nil
		}
		if p.value == nil {
			if len(p.path) == 1 {
				if p.args == nil {
					return values[p.path[0]], p.typ, nil
				}
				return evalCall(p, values, layouts)
			}
			v, err := valueOf(p.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			return v, p.typ, nil
		}
	}

	var v any
	var err error
	var vt types.Type
	var buf []byte

	for _, p := range expression {
		if s, _ := p.value.(string); s != "" {
			buf = append(buf, s...)
		}
		if p.path == nil {
			continue
		}
		if p.args == nil {
			v, err = valueOf(p.path, values)
			if err != nil {
				return nil, types.Type{}, err
			}
			vt = p.typ
		} else {
			v, vt, err = evalCall(p, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
		}
		buf, err = appendAsString(buf, v, vt)
		if err != nil {
			return nil, types.Type{}, err
		}
	}

	return string(buf), types.Text(), nil
}

// valueOf returns the value at the given path in values.
// It returns an error if the path does not exist.
func valueOf(path path, values map[string]any) (any, error) {
	var v any
	var ok bool
	last := len(path) - 1
	for i, name := range path {
		if name[0] == ':' {
			name = name[1:]
			if n := len(name) - 1; name[n] == '?' {
				name = name[:n]
			}
		}
		if name[0] == '[' {
			name = name[1 : len(name)-1]
		}
		v, ok = values[name]
		if !ok {
			return nil, nil
		}
		if i != last {
			values, ok = v.(map[string]any)
			if !ok {
				if name := path[i+1]; name[len(name)-1] == '?' {
					return nil, errVoid
				}
				var t string
				switch v.(type) {
				case nil:
					t = "JSON null"
				case bool:
					t = "a JSON boolean"
				case float64, json.Number:
					t = "a JSON number"
				case string:
					t = "a JSON string"
				default:
					t = "a JSON array"
				}
				return nil, fmt.Errorf("invalid %s: %s is not a JSON object, it is %s", stringifyPath(path[:i+2]), stringifyPath(path[:i+1]), t)
			}
		}
	}
	return v, nil
}

// stringifyPath returns path as a string.
func stringifyPath(path []string) string {
	s := path[0]
	for _, name := range path[1:] {
		if name[0] == ':' {
			name = name[1:]
		}
		question := name[len(name)-1] == '?'
		if question {
			name = name[:len(name)-1]
		}
		if name[0] == '[' {
			s += "[" + strconv.Quote(name[1:len(name)-1]) + "]"
		} else {
			s += "." + name
		}
		if question {
			s += "?"
		}
	}
	return s
}

// evalCall evaluates p representing a function call, and returns its value and
// type. values contains the property values. timeLayouts represents, if not null,
// the timeLayouts used to format DateTime, Date, and Time values as strings.
func evalCall(p part, values map[string]any, layouts *state.TimeLayouts) (any, types.Type, error) {
	switch name := p.path[0]; name {
	case "and":
		for _, arg := range p.args {
			v, _, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			if !v.(bool) {
				return false, types.Boolean(), nil
			}
		}
		return true, types.Boolean(), nil
	case "array":
		a := make([]any, len(p.args))
		for i, arg := range p.args {
			v, _, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			a[i] = v
		}
		return a, types.Array(types.JSON()), nil
	case "coalesce":
		for _, arg := range p.args {
			v, vt, err := eval(arg, values, layouts)
			if err != nil {
				return nil, types.Type{}, err
			}
			if v != nil {
				return v, vt, nil
			}
		}
		return nil, p.typ, nil
	case "eq":
		v0, t0, err := eval(p.args[0], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		v1, t1, err := eval(p.args[1], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		if !types.Equal(t0, t1) {
			v0, err = convert(v0, t0, t1, true, layouts)
			if err != nil {
				if err == errInvalidConversion {
					return false, types.Boolean(), nil
				}
				return nil, types.Type{}, err
			}
		}
		return reflect.DeepEqual(v0, v1), types.Boolean(), nil
	case "when":
		v0, _, err := eval(p.args[0], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		if !v0.(bool) {
			return nil, types.Type{}, errVoid
		}
		v1, t1, err := eval(p.args[1], values, layouts)
		if err != nil {
			return nil, types.Type{}, err
		}
		return v1, t1, nil
	}
	panic(fmt.Errorf("unknown function %q", p.path[0]))
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
