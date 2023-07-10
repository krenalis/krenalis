//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import (
	"errors"

	"chichi/connector/types"
)

// numArguments reports the number of arguments for each expression function.
// It is used to initialize the slice of arguments before parsing them.
var numArguments = map[string]int{
	"coalesce": 2,
}

// checkCoalesce type checks a call to 'coalesce' with the given arguments.
func checkCoalesce(args [][]part, schema, dt types.Type, nullable bool) (types.Type, error) {
	if len(args) < 1 {
		return types.Type{}, errors.New("'coalesce' function requires at least one argument")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, dt, nullable)
		if err != nil {
			return types.Type{}, err
		}
	}
	return types.Type{}, nil
}
