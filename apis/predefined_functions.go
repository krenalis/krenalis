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

// PredefinedFuncID represents the ID of a predefined function.
type PredefinedFuncID int

const (
	TrimSpace PredefinedFuncID = iota + 1
	SplitName
	UpperCase
	LowerCase
)

var predefinedMappingFunctions = map[PredefinedFuncID]any{
	TrimSpace: strings.TrimSpace,
	SplitName: func(s string) (string, string) {
		parts := strings.Split(s, " ")
		return parts[0], parts[1]
	},
	UpperCase: strings.ToUpper,
	LowerCase: strings.ToLower,
}

// callPredefinedFunction calls the predefined function with the given ID,
// passing to it the given arguments and returning its output values.
func callPredefinedFunction(id PredefinedFuncID, in []any) []any {
	f, ok := predefinedMappingFunctions[id]
	if !ok {
		// TODO(Gianluca): determine the best way to handle errors: maybe they
		// should be checked during validation?
		panic(fmt.Sprintf("invalid id %d", id))
	}
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
