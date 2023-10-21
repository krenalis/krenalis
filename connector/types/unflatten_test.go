//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"testing"
)

func TestUnflatten(t *testing.T) {

	tests := []struct {
		Flatten   Type
		Unflatten Type
	}{
		{
			Flatten:   Boolean(),
			Unflatten: Boolean(),
		}, {
			Flatten:   Text().WithCharLen(10),
			Unflatten: Text().WithCharLen(10),
		}, {
			Flatten:   Text().WithValues("a", "b"),
			Unflatten: Text().WithValues("a", "b"),
		}, {
			Flatten:   Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
			Unflatten: Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
		}, {
			Flatten: Object([]Property{{Name: "user", Type: Object(
				[]Property{{Name: "address", Type: Object(
					[]Property{{Name: "street", Type: Text()}}),
					Nullable: true, Flat: true}}),
				Nullable: true, Flat: true}}),
			Unflatten: Object([]Property{{Name: "user", Type: Object(
				[]Property{{Name: "address", Type: Object(
					[]Property{{Name: "street", Type: Text()}}),
					Nullable: true}}),
				Nullable: true}}),
		}, {
			Flatten:   Array(Boolean()),
			Unflatten: Array(Boolean()),
		}, {
			Flatten:   Array(Object([]Property{{Name: "address", Type: Object([]Property{{Name: "street", Type: Text()}}), Required: true, Flat: true}})),
			Unflatten: Array(Object([]Property{{Name: "address", Type: Object([]Property{{Name: "street", Type: Text()}}), Required: true}})),
		}, {
			Flatten:   Map(Int()),
			Unflatten: Map(Int()),
		}, {
			Flatten:   Map(Object([]Property{{Name: "email", Type: Object([]Property{{Name: "street", Type: Text()}}), Nullable: true, Flat: true}})),
			Unflatten: Map(Object([]Property{{Name: "email", Type: Object([]Property{{Name: "street", Type: Text()}}), Nullable: true}})),
		},
	}

	for _, test := range tests {
		err := sameType(test.Unflatten, test.Flatten.Unflatten())
		if err != nil {
			t.Errorf("%s: %s", test.Flatten, err)
		}
	}

}
