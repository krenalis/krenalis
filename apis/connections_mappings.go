//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2023 Open2b
//

package apis

import (
	"chichi/apis/types"
	"fmt"
	"reflect"
	"strings"
)

// Mapping represents a mapping from a kind of properties to another.
//
// In particular, if the mapping refers to a source connection, it is a mapping
// from the connection properties to a property of the Golden Record; otherwise,
// if it refers to a destination connection, it is a mapping from the Golden
// Record properties to a connection property.
//
// A mapping with just one input property and no source code is a "one to one"
// mapping (without transformation) from a property to another.
type Mapping struct {

	// id is the identifier of the mapping.
	id int

	// connection is the connection.
	connection *Connection

	// in is the schema of the input properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// connection, otherwise, if it is a destination, it contains the properties
	// of the Golden Record.
	//
	// In case of "one to one" mappings, this schema contains just one property.
	in types.Type

	// PredefinedFunc is the ID of the predefined transformation function of
	// this mapping, otherwise is zero if this mapping is not a predefined
	// transformation mapping.
	predefinedFunc int

	// sourceCode is the source code of the transformation function, which
	// should be something like:
	//
	//    def transform(user):
	//        return {
	//            "FirstName": user["firstname"],
	//        }
	//
	// This is the empty string for "one to one" mappings.
	sourceCode string

	// out is the schema of the output properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// Golden Record, otherwise, if it is a destination, it contains the
	// properties of the connection.
	out types.Type
}

// MappingInfo describes a mapping as returned by Get and List.
type MappingInfo struct {
	ID             int
	In             types.Type // just one property if it refers to a "one to one" mapping.
	SourceCode     string     // empty string if it refers to a "one to one" mapping.
	PredefinedFunc int        // zero if not a predefined transformation mapping.
	Out            types.Type // just one property if it refers to a "one to one" mapping.
}

// MappingToCreate represents a mapping to create.
//
// A mapping with just one input property and no source code is a "one to one"
// mapping (without transformation) from a property to another.
type MappingToCreate struct {
	// In is the schema of the input properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// connection, otherwise, if it is a destination, it contains the properties
	// of the Golden Record.
	//
	// In case of "one to one" mappings, this schema contains just one property.
	In types.Type

	// PredefinedFunc is the ID of the predefined transformation function of
	// this mapping, otherwise is zero if this mapping is not a predefined
	// transformation mapping.
	PredefinedFunc int

	// SourceCode is the source code of the transformation function, which
	// should be something like:
	//
	//    def transform(user):
	//        return {
	//            "FirstName": user["firstname"],
	//        }
	//
	// In case of "one to one" mappings, this is the empty string.
	SourceCode string

	// out is the schema of the output properties of the mapping. If the
	// connection is a source then the properties are the properties of the
	// Golden Record, otherwise, if it is a destination, it contains the
	// properties of the connection.
	Out types.Type
}

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
	// Input schema.
	In types.Type
	// F is the function to call.
	F any
	// Output schema.
	Out types.Type
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
		In: types.Object([]types.Property{
			{Name: "in", Label: "In", Type: types.Text()},
		}),
		F: strings.TrimSpace,
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
		F: func(s string) (string, string) {
			parts := strings.Split(s, " ")
			return parts[0], parts[1]
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
		F: strings.ToUpper,
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
		F: strings.ToLower,
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
		if (i + 1) != pf.ID {
			panic(fmt.Sprintf("BUG: invalid ID (%d != %d)", (i + 1), pf.ID))
		}
	}
}

// predefinedFuncByID returns the predefined function with the given ID and
// true, if exists, otherwise returns 'PredefinedFunc{}' and false.
func predefinedFuncByID(id int) (PredefinedFunc, bool) {
	if id > len(PredefinedMappingFuncs) {
		return PredefinedFunc{}, false
	}
	return PredefinedMappingFuncs[id-1], true
}

// callPredefinedFuncByID calls the predefined function with the given ID,
// passing to it the given arguments and returning its output values.
func callPredefinedFuncByID(id int, in []any) []any {
	f, _ := predefinedFuncByID(id)
	inRVs := make([]reflect.Value, len(in))
	for i := range inRVs {
		inRVs[i] = reflect.ValueOf(in[i])
	}
	outRVs := reflect.ValueOf(f.F).Call(inRVs)
	out := make([]any, len(outRVs))
	for i := range outRVs {
		out[i] = outRVs[i].Interface()
	}
	return out
}
