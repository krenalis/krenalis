//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package transformations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"chichi/apis/transformations/micropython"
)

type Transformations struct{}

// NewPool returns a new transformations pool.
// TODO(Gianluca): allow to set the minimum and maximum number of VMs which can
// be run contemporary.
func NewPool() *Transformations {
	return &Transformations{}
}

const stackSize = 10 * 1024 * 1024

// Run runs the Python code, passing to it the given parameters and returning
// the resulting values.
//
// In particular, code must be the source of a Python function named 'f' which
// takes parameters and returns one or more values.
//
// For example, code may be:
//
//	  def f(a, b):
//		    return a + b
//
// type annotations may be optionally provided (they serve just as documentation
// and will be ignored when interpreting the code):
//
//	  def f(a: int, b: int) -> int:
//		    return a + b
//
// TODO(Gianluca): note that this method is deprecated (in favour of the
// function that handles the new transformations) and will be removed in the
// future.
func (t *Transformations) Run(ctx context.Context, code string, params []any) ([]any, error) {

	// Initialize a new MicroPython VM that writes the stdout to a buffer.
	stdout := &bytes.Buffer{}
	vm, err := micropython.NewVM(ctx, stackSize, stdout, true)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate a MicroPython VM: %s", err)
	}
	defer vm.Close()

	// Wrap the 'f' function and run it.
	src := &bytes.Buffer{}
	src.WriteString("import json\n")
	src.WriteString(code + "\n\n")
	src.WriteString("out = f(*json.loads(" + `"""`)
	err = json.NewEncoder(src).Encode(params)
	if err != nil {
		return nil, fmt.Errorf("cannot encode parameters: %s", err)
	}
	src.WriteString(`"""))`)
	src.WriteString("\nprint(json.dumps(out if isinstance(out, tuple) else (out,)))")
	err = vm.RunSourceCode(src.Bytes())
	if err != nil {
		log.Printf("[info] this is the source code that failed:\n\n%s", src.String())
		log.Printf("[info] you can copy-paste this code in a service like https://www.programiz.com/python-programming/online-compiler/ to get more information about the error")
		return nil, fmt.Errorf("cannot run Python source code: %s", err)
	}

	// Decode the stdout as JSON and return it.
	var out []any
	err = json.NewDecoder(stdout).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("cannot decode JSON printed by MicroPython: %s", err)
	}

	return out, nil
}

// Run2 runs the Python transformation function declared in transformSource,
// applying it to user and returning the resulting properties map.
//
// The transformation function should be called 'transform', should take a
// single input parameter (a dictionary) and return a single output parameter (a
// dictionary).
//
// An example of a valid transformation function is:
//
//	def transform(user: dict) -> dict:
//	    return user
//
// Note that type annotations added to the function definition will be simply
// ignored by this method, as they serve just as documentation.
//
// TODO(Gianluca): this method will be renamed to 'Run' when the support for the
// current custom mappings will be dropped from chichi.
func (t *Transformations) Run2(ctx context.Context, transformSource string, user map[string]any) (map[string]any, error) {

	// Initialize a new MicroPython VM that writes the stdout to a buffer.
	stdout := &bytes.Buffer{}
	vm, err := micropython.NewVM(ctx, stackSize, stdout, true)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate a MicroPython VM: %s", err)
	}
	defer vm.Close()

	// Wrap the 'transform' function and run it.
	src := &bytes.Buffer{}
	src.WriteString("import json\n")
	src.WriteString(transformSource + "\n\n")
	src.WriteString("out = transform(json.loads(" + `"""`)
	err = json.NewEncoder(src).Encode(user)
	if err != nil {
		return nil, fmt.Errorf("cannot encode parameters: %s", err)
	}
	src.WriteString(`"""))`)
	src.WriteString("\nprint(json.dumps(out))")
	err = vm.RunSourceCode(src.Bytes())
	if err != nil {
		log.Printf("[info] this is the source code that failed:\n\n%s", src.String())
		log.Printf("[info] you can copy-paste this code in a service like https://www.programiz.com/python-programming/online-compiler/ to get more information about the error")
		return nil, fmt.Errorf("cannot run Python source code: %s", err)
	}

	// Decode the stdout as JSON and return it.
	var out map[string]any
	err = json.NewDecoder(stdout).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("cannot decode JSON printed by MicroPython: %s", err)
	}

	return out, nil

}
