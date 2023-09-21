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
)

var (
	ErrNotExist = errors.New("function does not exist")
	ErrExist    = errors.New("function already exists")
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
// A function name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_-]
type Transformer interface {

	// CallFunction calls the function with the given name an version, with the
	// given values to transform, and returns the results. If an error occurred
	// during its execution, it returns an ExecutionError error.
	CallFunction(ctx context.Context, name, version string, values []map[string]any) ([]Result, error)

	// Close the transformer.
	Close(ctx context.Context) error

	// CreateFunction creates a new function with the given name and source, and
	// returns its version, which has a length in the range [1, 128]. If a function
	// with the same name already exists, it returns the ErrExist error.
	CreateFunction(ctx context.Context, name, source string) (string, error)

	// DeleteFunction deletes the function with the given name.
	// If a function with the given name does not exist, it does nothing.
	DeleteFunction(ctx context.Context, name string) error

	// UpdateFunction updates the source of the function with the given name, and
	// returns a new version, which has a length in the range [1, 128]. If the
	// function does not exist, it returns the ErrNotExist error.
	UpdateFunction(ctx context.Context, name, source string) (string, error)
}
