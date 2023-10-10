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

	"chichi/apis/state"
)

var (
	ErrExist        = errors.New("function already exists")
	ErrNotExist     = errors.New("function does not exist")
	ErrPendingState = errors.New("function is in a pending state")
)

// An ExecutionError error is returned by the Transformer.CallFunction method
// when an error occurs executing the function.
type ExecutionError struct {
	Msg string
}

// NewExecutionError returns a new execution error with message msg.
func NewExecutionError(msg string) *ExecutionError {
	return &ExecutionError{Msg: msg}
}

func (err *ExecutionError) Error() string {
	return err.Msg
}

type Result struct {
	Value map[string]any
	Error ValueError
}

type ValueError string

func (err ValueError) Error() string {
	return string(err)
}

// A Transformer represents a transformer.
//
// A function name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_-]
//   - terminate with ".js", for JavaScript functions, or with ".py" for Python
//     functions
type Transformer interface {

	// CallFunction calls the function with the given name and version, with the
	// given values to transform, and returns the results. If an error occurs during
	// execution, it returns an *ExecutionError error. If the function does not
	// exist, it returns the ErrNotExist error. If the function is in a pending
	// state, it returns the ErrPendingState error.
	CallFunction(ctx context.Context, name, version string, values []map[string]any) ([]Result, error)

	// Close the transformer.
	Close(ctx context.Context) error

	// CreateFunction creates a new function with the given name and source, and
	// returns its version, which has a length in the range [1, 128]. name should
	// have an extension of either ".js" or ".py" depending on the source code's
	// language. If a function with the same name already exists, it returns the
	// ErrExist error.
	CreateFunction(ctx context.Context, name, source string) (string, error)

	// DeleteFunction deletes the function with the given name.
	// If a function with the given name does not exist, it does nothing.
	DeleteFunction(ctx context.Context, name string) error

	// SupportLanguage reports whether language is supported as a language.
	// It panics if language is not valid.
	SupportLanguage(language state.Language) bool

	// UpdateFunction updates the source of the function with the given name, and
	// returns a new version, which has a length in the range [1, 128]. If the
	// function does not exist, it returns the ErrNotExist error.
	UpdateFunction(ctx context.Context, name, source string) (string, error)
}

// ValidFunctionName reports whether name is a valid function name.
func ValidFunctionName(name string) bool {
	if len(name) < 4 {
		return false
	}
	name, ext := name[:len(name)-3], name[len(name)-3:]
	if ext != ".js" && ext != ".py" {
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
