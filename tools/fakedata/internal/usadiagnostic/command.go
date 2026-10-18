// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package usadiagnostic connects the repository-only CLI to private samplers.
package usadiagnostic

import (
	"context"
	"fmt"
)

// Options selects an offline diagnostic operation; publication is forbidden.
type Options struct {
	Operation      string
	Root           string
	Directory      string
	ModelDirectory string
	Country        string
	Currency       string
	Profile        string
	Order          string
	People         int
	Publish        bool
}

// runner is assigned once during package initialization, before concurrent use.
var runner func(context.Context, Options) error

// Register installs the private implementation during package initialization.
func Register(implementation func(context.Context, Options) error) {
	if runner != nil || implementation == nil {
		panic("USA diagnostic implementation must be registered exactly once")
	}
	runner = implementation
}

// Run executes the registered repository-only operation.
func Run(ctx context.Context, options Options) error {
	if ctx == nil || runner == nil {
		return fmt.Errorf("USA diagnostic context and implementation are required")
	}
	return runner(ctx, options)
}
