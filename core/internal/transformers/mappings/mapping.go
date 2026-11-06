// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

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
	expressions    []mappingExpr
	createRequired *requiredProperties
	updateRequired *requiredProperties
}

type mappingExpr struct {
	path            string
	expr            *Expression
	properties      []string
	dt              types.Type // destination type.
	nullable        bool
	convert         bool // non-null results need value conversion
	encodeNilAsJSON bool // encode nil as JSON null for non-nullable JSON destinations
}

// New returns a new mapping that transforms values according to the provided
// expressions. inSchema and outSchema represent the input and output schemas,
// respectively.
//
// The source type can be the invalid type if expressions do not contain paths.
//
// matchingOut is an output property supplied separately by the exporter, or
// empty if there is none. It must have type string, int, or uuid, and neither
// it nor an ancestor may be mapped. The exporter must supply a non-nil value
// on creation and on update when the property is UpdateRequired. Required
// checks account for ancestors created by that insertion without changing the
// output.
//
// It returns a types.PathNotExistError error if a path in expressions does not
// exist in the source schema or matchingOut does not exist in outSchema.
func New(expressions map[string]string, inSchema, outSchema types.Type, matchingOut string) (*Mapping, error) {

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
	// Validate the output matching property.
	var matching types.Property
	if matchingOut != "" {
		if !types.IsValidPropertyPath(matchingOut) {
			return nil, errors.New("output matching property path is not valid")
		}
		var err error
		matching, err = outSchema.Properties().ByPath(matchingOut)
		if err != nil {
			return nil, err
		}
		switch matching.Type.Kind() {
		case types.StringKind, types.IntKind, types.UUIDKind:
			// These types can be used for matching by the exporter.
		default:
			return nil, errors.New("output matching property must have type string, int, or uuid")
		}
		for path := range expressions {
			if path == matchingOut || strings.HasPrefix(matchingOut, path+".") {
				return nil, errors.New("mapping cannot map over the output matching property")
			}
		}
	}
	// Compile the expressions.
	me := make([]mappingExpr, len(expressions))
	i := 0
	properties := outSchema.Properties()
	for path, expr := range expressions {
		p, err := properties.ByPath(path)
		if err != nil {
			return nil, err
		}
		me[i].path = path
		me[i].expr, me[i].properties, err = Compile(expr, inSchema, p.Type)
		if err != nil {
			return nil, err
		}
		me[i].dt = p.Type
		me[i].nullable = p.Nullable
		me[i].convert = needsConversion(me[i].expr.parts, p.Type)
		me[i].encodeNilAsJSON = !me[i].convert && needsJSONNormalization(p.Type, p.Nullable)
		i++
	}
	err := sortMappingExpressions(me)
	if err != nil {
		return nil, err
	}

	// Compile required-property checks, skipping those guaranteed by the source
	// schema for values that do not need conversion.
	mapping := &Mapping{
		expressions:    me,
		createRequired: newRequiredProperties(outSchema, types.Type{}, Create),
		updateRequired: newRequiredProperties(outSchema, types.Type{}, Update),
	}
	for _, e := range me {
		if !e.convert {
			path := strings.Split(e.path, ".")
			var st types.Type
			if parts := e.expr.parts; len(parts) == 1 && parts[0].args == nil {
				st = typeOf(parts)
			}
			mapping.createRequired = mapping.createRequired.removeGuaranteedChecks(path, e.dt, st, Create)
			mapping.updateRequired = mapping.updateRequired.removeGuaranteedChecks(path, e.dt, st, Update)
		}
	}
	if matchingOut != "" {
		path := strings.Split(matchingOut, ".")
		mapping.createRequired = mapping.createRequired.providePath(path)
		if matching.UpdateRequired {
			mapping.updateRequired = mapping.updateRequired.providePath(path)
		}
	}

	return mapping, nil
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

// Transform transforms attributes, that must conform to the expression's source
// schema, and returns the result that conforms to the expression's output
// schema.
//
// Transform does not modify attributes. The result may share arrays, objects,
// and maps with attributes. Changes to shared slices and maps affect both
// values. To modify either value independently, callers must first make a deep
// copy of that value.
//
// purpose specifies the reason for the transformation. If Create or Update,
// then all the properties required for creation or the update must be present
// in the returned value.
//
// If an expression evaluates to nil and the corresponding property cannot be
// null, that property will be omitted from the returned result, provided this
// is allowed by the purpose. For non-nullable JSON properties, nil becomes
// JSON null instead.
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
		// Nil becomes JSON null only for a non-nullable JSON destination.
		nullable := e.nullable || !e.dt.IsJSON()
		if v != nil && (e.convert || e.encodeNilAsJSON) || v == nil && !nullable {
			if e.encodeNilAsJSON {
				v, _, err = convertEqual(v, e.dt, nullable, None)
			} else {
				v, err = convert(v, vt, e.dt, nullable, None)
			}
			if err != nil {
				var msg string
				switch err {
				case errRangeConversion:
					msg = fmt.Sprintf("number «%s» is too small or too large for the «%s» type while mapping to «%s»", code(e.expr.source), e.dt, code(e.path))
				case errMinConversion:
					var n any
					switch e.dt.Kind() {
					case types.IntKind:
						if e.dt.IsUnsigned() {
							n, _ = e.dt.UnsignedRange()
						} else {
							n, _ = e.dt.IntRange()
						}
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
						if e.dt.IsUnsigned() {
							_, n = e.dt.UnsignedRange()
						} else {
							_, n = e.dt.IntRange()
						}
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
					case types.IPKind:
						to = "an IP address"
					}
					msg = fmt.Sprintf("«%s» is not parsable as %s while mapping to «%s»", code(e.expr.source), to, code(e.path))
				case errYearRangeConversion:
					msg = fmt.Sprintf("year of «%s» is not in range [1,9999] while mapping to «%s»", code(e.expr.source), code(e.path))
				case errEnumConversion:
					msg = fmt.Sprintf("«%s» is not one of the allowed values while mapping to «%s»", code(e.expr.source), code(e.path))
				case errCountryAlpha2Conversion:
					msg = fmt.Sprintf("«%s» is not a 2-letter country code while mapping to «%s»", code(e.expr.source), code(e.path))
				case errCountryAlpha3Conversion:
					msg = fmt.Sprintf("«%s» is not a 3-letter country code while mapping to «%s»", code(e.expr.source), code(e.path))
				case errPhoneConversion:
					msg = fmt.Sprintf("«%s» is not a valid phone number while mapping to «%s»", code(e.expr.source), code(e.path))
				case errPatternConversion:
					msg = fmt.Sprintf("«%s» does not match «/%s/» while mapping to «%s»", code(e.expr.source), e.dt.Pattern(), code(e.path))
				case errMaxBytesConversion:
					n, _ := e.dt.MaxBytes()
					msg = fmt.Sprintf("«%s» exceeds the %d-byte limit while mapping to «%s»", code(e.expr.source), n, code(e.path))
				case errMaxLengthConversion:
					n, _ := e.dt.MaxLength()
					msg = fmt.Sprintf("«%s» exceeds the %d-char limit while mapping to «%s»", code(e.expr.source), n, code(e.path))
				default:
					msg = fmt.Sprintf("«%s» is not convertible to the «%s» type while mapping to «%s»", code(e.expr.source), e.dt.String(), code(e.path))
				}
				return nil, ValidationError{msg}
			}
		}
		if v == nil && !e.nullable {
			continue
		}
		storeValue(out, e.path, v)
	}

	// Check required properties after building the complete output.
	if purpose == Create || purpose == Update {
		required, operation := mapping.createRequired, "creation"
		if purpose == Update {
			required, operation = mapping.updateRequired, "update"
		}
		if path := required.missingPath(out); path != "" {
			path = strings.TrimPrefix(path, ".")
			msg := fmt.Sprintf("property «%s» is missing but it is required for %s", code(path), operation)
			return nil, ValidationError{msg}
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
