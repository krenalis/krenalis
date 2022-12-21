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

// Run runs the Python code, passing to it the given user and returning the
// resulting value.
//
// In particular, the code must be the source code of a Python function named
// 'transform' which takes a single parameter of type 'dict' with string as keys
// (which are the property names) and property values as values, and must return
// a value which will be put in the Golden Record.
//
// For example, code may be:
//
//	def transform(user):
//	    return user["firstname"]
//
// type annotations may be optionally provided (they serve just as documentation
// and will be ignored when interpreting the code):
//
//	def transform(user: dict) -> str:
//	    return user["firstname"]
func (t *Transformations) Run(ctx context.Context, code string, user map[string]any) (any, error) {

	// Initialize a new MicroPython VM that writes the stdout to a buffer.
	stdout := &bytes.Buffer{}
	vm, err := micropython.NewVM(ctx, stackSize, stdout, true)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate a MicroPython VM: %s", err)
	}
	defer vm.Close()

	// Inject code around the function code and run it.
	src := &bytes.Buffer{}
	src.WriteString(code)
	src.WriteString("\nimport json\n\nprint(json.dumps(transform(json.loads(" + `"""` + "\n")
	err = json.NewEncoder(src).Encode(user)
	if err != nil {
		return nil, fmt.Errorf("cannot encode parameters: %s", err)
	}
	src.WriteString(`"""))))`)
	err = vm.RunSourceCode(src.Bytes())
	if err != nil {
		log.Printf("this is the source code that failed:\n\n%s", src.String())
		log.Printf("you can copy-paste this code in a service like https://www.programiz.com/python-programming/online-compiler/ to get more information about the error")
		return nil, fmt.Errorf("cannot run Python source code: %s", err)
	}

	// Decode the stdout as JSON and return it.
	var out any
	err = json.NewDecoder(stdout).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("cannot decode JSON printed by MicroPython: %s", err)
	}

	return out, nil
}
