// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"fmt"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/tools/validation"
)

// SemanticValidationError describes a value that does not satisfy its property's semantic.
type SemanticValidationError struct {
	Path string
	Msg  string
}

// Error returns the validation error message.
func (err *SemanticValidationError) Error() string {
	return fmt.Sprintf("property %q %s", err.Path, err.Msg)
}

// ValidateSemantics checks the Country and Phone constraints of the non-null
// values present in value, including nested objects, array elements and map values.
// Alpha-2 Country values must be assigned or formerly assigned codes; alpha-3
// values are checked only for UTF-8 validity and their three-byte limit. Phone
// values must be valid UTF-8 strings of at most 16 bytes and 16 characters.
// It returns a *SemanticValidationError if validation fails.
//
// This is not a full type validator: it does not check required properties,
// nullability, other semantics, or constraints unrelated to these semantics.
// p.Type must be valid and non-generic. p.Name identifies the value in errors;
// it may be empty when validating a whole object.
//
// No context is needed: traversal is bounded by the already materialized value
// and the finite nesting of p.Type, and performs no I/O.
func ValidateSemantics(p types.Property, value any) error {

	if !p.Type.Valid() || p.Type.Generic() {
		return &SemanticValidationError{Path: p.Name, Msg: "has an invalid or generic type"}
	}
	if value == nil {
		return nil
	}

	switch p.Type.Kind() {

	case types.ObjectKind:

		object, ok := value.(map[string]any)
		if !ok {
			return &SemanticValidationError{Path: p.Name, Msg: "is not an object"}
		}
		for _, property := range p.Type.Properties().All() {
			v, ok := object[property.Name]
			if !ok {
				continue
			}
			if p.Name != "" {
				property.Name = p.Name + "." + property.Name
			}
			err := ValidateSemantics(property, v)
			if err != nil {
				return err
			}
		}

	case types.ArrayKind:

		array, ok := value.([]any)
		if !ok {
			return &SemanticValidationError{Path: p.Name, Msg: "is not an array"}
		}
		elem := p
		elem.Type = p.Type.Elem()
		for i, v := range array {
			elem.Name = fmt.Sprintf("%s[%d]", p.Name, i)
			err := ValidateSemantics(elem, v)
			if err != nil {
				return err
			}
		}

	case types.MapKind:

		object, ok := value.(map[string]any)
		if !ok {
			return &SemanticValidationError{Path: p.Name, Msg: "is not a map"}
		}
		elem := p
		elem.Type = p.Type.Elem()
		for key, v := range object {
			// Bound user-provided keys before including them in errors.
			elem.Name = fmt.Sprintf("%s[%.64q]", p.Name, key)
			err := ValidateSemantics(elem, v)
			if err != nil {
				return err
			}
		}

	default:

		if p.Type.Semantic() == types.NoSemantic {
			return nil
		}
		switch p.Type.Semantic() {
		case types.CountrySemantic:
			s, ok := value.(string)
			if !ok || len(s) > int(p.Type.CountryFormat()) || !utf8.ValidString(s) {
				return &SemanticValidationError{Path: p.Name, Msg: "contains an invalid country value"}
			}
			if p.Type.CountryFormat() == types.ISO3166Alpha2 && !validation.IsValidCountryCodeAlpha2(s) {
				return &SemanticValidationError{Path: p.Name, Msg: "contains an invalid country code"}
			}
		case types.PhoneSemantic:
			s, ok := value.(string)
			if !ok {
				return &SemanticValidationError{Path: p.Name, Msg: "is not a valid UTF-8 string"}
			}
			// The byte limit also bounds the number of UTF-8 characters.
			if len(s) > 16 {
				return &SemanticValidationError{Path: p.Name, Msg: "exceeds the 16-byte limit"}
			}
			if !utf8.ValidString(s) {
				return &SemanticValidationError{Path: p.Name, Msg: "is not a valid UTF-8 string"}
			}
		}

	}

	return nil
}
