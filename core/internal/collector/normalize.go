// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// jsonValidationError identifies invalid event data and its property path.
type jsonValidationError struct {
	path string
	err  error
}

// Error returns the validation error with its property path.
func (err *jsonValidationError) Error() string {
	if err.path == "" {
		return err.err.Error()
	}
	return fmt.Sprintf("%s: %s", err.path, err.err)
}

// PropertyPath returns the path of the value being validated.
func (err *jsonValidationError) PropertyPath() string {
	return err.path
}

// Unwrap returns the underlying validation error.
func (err *jsonValidationError) Unwrap() error {
	return err.err
}

func (err *jsonValidationError) withProperty(name string) *jsonValidationError {
	path := name
	if err.path != "" {
		path += "." + err.path
	}
	return &jsonValidationError{path: path, err: err.err}
}

// normalizeJSON validates and normalizes event JSON against a catalog schema,
// preserving undeclared properties. The schema follows the profile-schema
// restrictions, so arrays and maps contain scalars or opaque JSON values.
// Input representations and scalar normalization follow types.Decode/Marshal;
// additional conversions are deferred. Invalid input returns a ValidationError
// whose property path starts with name.
func normalizeJSON(name string, v json.Value, t types.Type) (json.Value, error) {
	if !t.Valid() || t.Generic() {
		return nil, errors.New("core/internal/collector: invalid JSON normalization schema")
	}
	if !json.Valid(v) {
		return nil, &jsonValidationError{path: name, err: errors.New("invalid JSON")}
	}
	normalized, err := normalizeJSONValue(v, t)
	if err != nil {
		if err, ok := errors.AsType[*jsonValidationError](err); ok {
			return nil, err.withProperty(name)
		}
		return nil, err
	}
	return normalized, nil
}

// normalizeJSONValue normalizes a valid JSON value with a concrete schema.
func normalizeJSONValue(v json.Value, t types.Type) (json.Value, error) {

	if !t.IsJSON() && v.IsNull() {
		return nil, &jsonValidationError{err: errors.New("value cannot be null")}
	}
	if !t.IsObject() {
		value, err := types.Decode[any](bytes.NewReader(v), t)
		if err != nil {
			return nil, &jsonValidationError{err: err}
		}
		return types.Marshal(value, t)
	}

	var object map[string]json.Value
	err := json.Unmarshal(v, &object)
	if err != nil {
		return nil, &jsonValidationError{err: err}
	}
	var out json.Buffer
	out.WriteByte('{')
	for _, property := range t.Properties().All() {
		value, ok := object[property.Name]
		if !ok {
			if !property.ReadOptional {
				return nil, &jsonValidationError{path: property.Name, err: errors.New("required property is missing")}
			}
			continue
		}
		if !value.IsNull() || !property.Nullable {
			value, err = normalizeJSONValue(value, property.Type)
			if err != nil {
				if err, ok := errors.AsType[*jsonValidationError](err); ok {
					return nil, err.withProperty(property.Name)
				}
				return nil, err
			}
		}
		err = out.EncodeKeyValue(property.Name, value)
		if err != nil {
			return nil, err
		}
		delete(object, property.Name)
	}

	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		err = out.EncodeKeyValue(name, object[name])
		if err != nil {
			return nil, err
		}
	}
	out.WriteByte('}')

	return out.Value()
}
