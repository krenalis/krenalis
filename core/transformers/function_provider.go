//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"context"
	"errors"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

var ErrFunctionNotExist = errors.New("function does not exist")

// FunctionExecutionError represents an error resulting from the execution of a
// transformation function such as a syntax error in the function.
type FunctionExecutionError string

func (err FunctionExecutionError) Error() string { return string(err) }

// A FunctionProvider represents a function provider.
//
// A function name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_-]
//   - terminate with ".js", for JavaScript functions, or with ".py" for Python
//     functions
type FunctionProvider interface {

	// Call calls the function with the given identifier and version for each record
	// updating its Properties field with the result of each invocation. Record
	// properties are supposed to conform to inSchema. After the transformation,
	// Record properties conform to outSchema unless a transformation error
	// occurred, and in that case, the error is stored in the Record's Err field.
	//
	// It returns the ErrFunctionNotExist error if the function does not exist, and
	// a FunctionExecutionError if the execution fails.
	Call(ctx context.Context, id, version string, inSchema, outSchema types.Type, preserveJSON bool, records []Record) error

	// Close closes the function.
	Close(ctx context.Context) error

	// Create creates a new function with the given name, language, and source and
	// returns its identifier and version.
	Create(ctx context.Context, name string, language state.Language, source string) (string, string, error)

	// Delete deletes the function with the given identifier.
	// If a function with the given identifier does not exist, it does nothing.
	Delete(ctx context.Context, id string) error

	// SupportLanguage reports whether language is supported as a language.
	// It panics if language is not valid.
	SupportLanguage(language state.Language) bool

	// Update updates the source of the function with the given identifier and
	// returns a new version, which has a length in the range [1, 128].
	// If the function does not exist, it returns the ErrFunctionNotExist error.
	Update(ctx context.Context, id, source string) (string, error)
}

// ValidFunctionName reports whether name is a valid function name.
func ValidFunctionName(name string) bool {
	if name == "" || len(name) > 60 {
		return false
	}
	n := len(name)
	for i := 0; i < n; i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || i > 0 && (c == '-' || c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}
