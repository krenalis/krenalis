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
)

// PredefinedFunc is a predefined function which can be used in mappings.
type PredefinedFunc struct {
	// ID.
	ID int
	// Name.
	Name string
	// A short description.
	Description string
	// An icon, which may be used in the UI.
	Icon string
	// Input parameters labels.
	In []string
	// F is the function to call.
	F any
	// Output parameters labels.
	Out []string
}

// Define the IDs of the predefined functions.
const (
	TrimSpace = iota + 1
	SplitName
	UpperCase
	LowerCase
)

var PredefinedMappingFuncs = []PredefinedFunc{
	{
		ID:          TrimSpace,
		Name:        "Trim whitespace",
		Description: "Trim whitespace at the start and the end of a string",
		Icon:        "scissors",
		In:          []string{"In"},
		F:           strings.TrimSpace,
		Out:         []string{"Out"},
	},
	{
		ID:          SplitName,
		Name:        "Split name",
		Description: "Split a full name in its 'name' and 'last name' components",
		Icon:        "signpost-split",
		In:          []string{"Full name"},
		F: func(s string) (string, string) {
			parts := strings.Split(s, " ")
			return parts[0], parts[1]
		},
		Out: []string{"First name", "Last name"},
	},
	{
		ID:          UpperCase,
		Name:        "Upper case",
		Description: "Change string case to upper case",
		Icon:        "arrow-up",
		In:          []string{"In"},
		F:           strings.ToUpper,
		Out:         []string{"Out"},
	},
	{
		ID:          LowerCase,
		Name:        "Lower case",
		Description: "Change string case to lower case",
		Icon:        "arrow-down",
		In:          []string{"In"},
		F:           strings.ToLower,
		Out:         []string{"Out"},
	},
}

func init() {
	// TODO(Gianluca): this check, which ensures consistency between the
	// predefined functions and their IDs, should be moved in tests when we will
	// have them.
	for i, pf := range PredefinedMappingFuncs {
		if (i + 1) != pf.ID {
			panic(fmt.Sprintf("BUG: invalid ID (%d != %d)", (i + 1), pf.ID))
		}
	}
}

// callPredefinedFuncByID calls the predefined function with the given ID,
// passing to it the given arguments and returning its output values.
func callPredefinedFuncByID(id int, in []any) []any {
	f := PredefinedMappingFuncs[id-1].F
	inRVs := make([]reflect.Value, len(in))
	for i := range inRVs {
		inRVs[i] = reflect.ValueOf(in[i])
	}
	outRVs := reflect.ValueOf(f).Call(inRVs)
	out := make([]any, len(outRVs))
	for i := range outRVs {
		out[i] = outRVs[i].Interface()
	}
	return out
}
