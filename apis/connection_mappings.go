//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2023 Open2b
//

package apis

import (
	"fmt"
	"reflect"
	"strings"

	"chichi/apis/state"
	"chichi/apis/types"
)

// Mapping represents a connection mapping.
type Mapping struct {
	InProperties   []string
	OutProperties  []string
	PredefinedFunc *PredefinedFunc
	CustomFunc     *MappingCustomFunc
}

type MappingCustomFunc struct {
	InTypes  []types.Type
	OutTypes []types.Type
	Source   string
}

type PredefinedFuncDefinition struct {

	// ID.
	ID PredefinedFunc

	// Name.
	Name string

	// A short textual description.
	Description string

	// Icon is the name of an icon for this predefined function, which may be
	// used in the UI.
	Icon string

	// Input schema. Must be valid.
	In types.Type

	// Func is the function to call. Can have multiple input and output
	// parameters.
	Func any

	// Output schema. Must be valid.
	Out types.Type
}

// PredefinedFunc represents the ID of a predefined function.
type PredefinedFunc int

const (
	TrimSpace PredefinedFunc = iota + 1
	SplitName
	UpperCase
	LowerCase
)

// PredefinedMappingFuncs is as set of predefined functions.
var PredefinedMappingFuncs = []PredefinedFuncDefinition{
	{
		ID:          TrimSpace,
		Name:        "Trim whitespace",
		Description: "Trim whitespace at the start and the end of a string",
		Icon:        "scissors",
		In: types.Object([]types.Property{
			{Name: "in", Label: "In", Type: types.Text()},
		}),
		Func: strings.TrimSpace,
		Out: types.Object([]types.Property{
			{Name: "out", Label: "Out", Type: types.Text()},
		}),
	},
	{
		ID:          SplitName,
		Name:        "Split name",
		Description: "Split a full name in its 'name' and 'last name' components",
		Icon:        "signpost-split",
		In: types.Object([]types.Property{
			{Name: "full_name", Label: "Full name", Type: types.Text()},
		}),
		Func: func(s string) (string, string) {
			parts := strings.SplitN(s, " ", 2)
			switch len(parts) {
			case 1:
				return parts[0], ""
			case 2:
				return parts[0], parts[1]
			default:
				return "", ""
			}
		},
		Out: types.Object([]types.Property{
			{Name: "first_name", Label: "First name", Type: types.Text()},
			{Name: "last_name", Label: "Last name", Type: types.Text()},
		}),
	},
	{
		ID:          UpperCase,
		Name:        "Upper case",
		Description: "Change string case to upper case",
		Icon:        "arrow-up",
		In: types.Object([]types.Property{
			{Name: "in", Label: "In", Type: types.Text()},
		}),
		Func: strings.ToUpper,
		Out: types.Object([]types.Property{
			{Name: "out", Label: "Out", Type: types.Text()},
		}),
	},
	{
		ID:          LowerCase,
		Name:        "Lower case",
		Description: "Change string case to lower case",
		Icon:        "arrow-down",
		In: types.Object([]types.Property{
			{Name: "in", Label: "In", Type: types.Text()},
		}),
		Func: strings.ToLower,
		Out: types.Object([]types.Property{
			{Name: "out", Label: "Out", Type: types.Text()},
		}),
	},
}

func init() {
	// TODO(Gianluca): this check, which ensures consistency between the
	// predefined functions and their IDs, should be moved in tests when we will
	// have them.
	for i, pf := range PredefinedMappingFuncs {
		if PredefinedFunc(i+1) != pf.ID {
			panic(fmt.Sprintf("BUG: invalid ID (%d != %d)", (i + 1), pf.ID))
		}
	}
}

// predefinedFuncDefinitionByID returns the definition of the predefined
// function with the given ID and true, if exists, otherwise returns
// 'PredefinedFuncDefinition{}' and false.
func predefinedFuncDefinitionByID(id state.PredefinedFunc) (PredefinedFuncDefinition, bool) {
	if int(id) > len(PredefinedMappingFuncs) {
		return PredefinedFuncDefinition{}, false
	}
	return PredefinedMappingFuncs[id-1], true
}

// callPredefinedFunc calls the given predefined function, passing to it the
// given arguments and returning its output values.
func callPredefinedFunc(f PredefinedFuncDefinition, in []any) []any {
	inRVs := make([]reflect.Value, len(in))
	for i := range inRVs {
		inRVs[i] = reflect.ValueOf(in[i])
	}
	outRVs := reflect.ValueOf(f.Func).Call(inRVs)
	out := make([]any, len(outRVs))
	for i := range outRVs {
		out[i] = outRVs[i].Interface()
	}
	return out
}
