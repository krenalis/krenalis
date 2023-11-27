//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"errors"

	"chichi/connector/types"
)

var jsonArrayType = types.Array(types.JSON())

// numArguments reports the number of arguments for each expression function.
// It is used to initialize the slice of arguments before parsing them.
var numArguments = map[string]int{
	"and":      2,
	"array":    2,
	"coalesce": 2,
	"eq":       2,
	"when":     2,
}

// checkAnd type checks a call to 'and' with the given arguments.
func checkAnd(args [][]part, schema, dt types.Type, required, nullable bool) (types.Type, error) {
	if len(args) < 2 {
		return types.Type{}, errors.New("'and' function requires at least two argument")
	}
	booleanType := types.Boolean()
	for _, arg := range args {
		err := typeCheck(arg, schema, booleanType, required, nullable)
		if err != nil {
			return types.Type{}, err
		}
	}
	return booleanType, nil
}

// checkArray type checks a call to 'array' with the given arguments.
func checkArray(args [][]part, schema, dt types.Type, required, nullable bool) (types.Type, error) {
	for _, arg := range args {
		err := typeCheck(arg, schema, types.JSON(), required, false)
		if err != nil {
			return types.Type{}, err
		}
	}
	return jsonArrayType, nil
}

// checkCoalesce type checks a call to 'coalesce' with the given arguments.
func checkCoalesce(args [][]part, schema, dt types.Type, required, nullable bool) (types.Type, error) {
	if len(args) < 1 {
		return types.Type{}, errors.New("'coalesce' function requires at least one argument")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, dt, required, nullable)
		if err != nil {
			return types.Type{}, err
		}
	}
	return types.Type{}, nil
}

// checkEq type checks a call to 'eq' with the given arguments.
func checkEq(args [][]part, schema, dt types.Type, required, nullable bool) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'eq' function requires two arguments")
	}
	for _, arg := range args {
		err := typeCheck(arg, schema, types.Type{}, required, true)
		if err != nil {
			return types.Type{}, err
		}
	}
	t0 := typesOf(args[0])
	t1 := typesOf(args[1])
	if !t0.Valid() || !t1.Valid() {
		return types.Boolean(), nil
	}
	if !convertibleTo(t0, t1) {
		return types.Type{}, errors.New("first argument of 'eq(...)' is not convertible to the type of the second")
	}
	if !convertibleTo(t1, t0) {
		return types.Type{}, errors.New("second argument of 'eq(...)' is not convertible to the type of the first")
	}
	return types.Boolean(), nil
}

// checkWhen type checks a call to 'when' with the given arguments.
func checkWhen(args [][]part, schema, dt types.Type, required, nullable bool) (types.Type, error) {
	if len(args) != 2 {
		return types.Type{}, errors.New("'when' function requires two arguments")
	}
	if required {
		return types.Type{}, errors.New("'when' function cannot be used in a required expression")
	}
	err := typeCheck(args[0], schema, types.Boolean(), false, true)
	if err != nil {
		return types.Type{}, err
	}
	err = typeCheck(args[1], schema, dt, false, nullable)
	if err != nil {
		return types.Type{}, err
	}
	return dt, nil
}
