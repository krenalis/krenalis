//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package schemas

import (
	"regexp"
	"testing"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

func Test_checkSchemaAlignment(t *testing.T) {

	var (
		createOnlyMode = state.CreateOnly
		updateOnlyMode = state.UpdateOnly
		createOrUpdate = state.CreateOrUpdate
	)

	tests := []struct {
		t1, t2 types.Type
		mode   *state.ExportMode
		err    string
	}{
		{t1: types.Boolean(), t2: types.Boolean()},
		{t1: types.Int(32), t2: types.Int(32), mode: &createOnlyMode},
		{t1: types.Int(32).WithIntRange(-10, 100), t2: types.Int(32).WithIntRange(-10, 100)},
		{t1: types.Uint(16).WithUintRange(0, 100), t2: types.Uint(16).WithUintRange(0, 100)},
		{t1: types.Float(64), t2: types.Float(64)},
		{t1: types.Float(32).AsReal(), t2: types.Float(32).AsReal(), mode: &createOnlyMode},
		{t1: types.Float(32).WithFloatRange(1.0, 5.67), t2: types.Float(32).WithFloatRange(1.0, 5.67)},
		{t1: types.Decimal(10, 2), t2: types.Decimal(10, 2)},
		{t1: types.DateTime(), t2: types.DateTime()},
		{t1: types.Date(), t2: types.Date()},
		{t1: types.Time(), t2: types.Time()},
		{t1: types.Year(), t2: types.Year()},
		{t1: types.UUID(), t2: types.UUID(), mode: &createOnlyMode},
		{t1: types.JSON(), t2: types.JSON()},
		{t1: types.Inet(), t2: types.Inet()},
		{t1: types.Text(), t2: types.Text(), mode: &createOnlyMode},
		{t1: types.Text().WithCharLen(100), t2: types.Text().WithCharLen(100)},
		{t1: types.Text().WithByteLen(50), t2: types.Text().WithByteLen(50)},
		{t1: types.Text().WithRegexp(regexp.MustCompile(`^\d+`)), t2: types.Text().WithRegexp(regexp.MustCompile(`^\d+`)), mode: &createOnlyMode},
		{t1: types.Text().WithRegexp(regexp.MustCompile(`^\d+`)).WithByteLen(10), t2: types.Text().WithRegexp(regexp.MustCompile(`^\d+`)).WithByteLen(10)},
		{t1: types.Array(types.Int(8)), t2: types.Array(types.Int(8))},
		{
			t1: types.Object([]types.Property{{Name: "a", Label: "A", Type: types.Boolean(), Nullable: true, Note: "a property"}}),
			t2: types.Object([]types.Property{{Name: "a", Label: "a", Type: types.Boolean(), Nullable: true}}),
		},
		{
			t1:   types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: true}}),
			t2:   types.Object([]types.Property{{Name: "a", Placeholder: "a", Type: types.Boolean(), CreateRequired: true}, {Name: "b", Type: types.Int(64)}}),
			mode: &createOnlyMode,
		},
		{
			t1: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: true}}),
			t2: types.Object([]types.Property{{Name: "a", Placeholder: "a", Type: types.Boolean(), ReadOptional: true}, {Name: "b", Type: types.Int(64), CreateRequired: true}}),
		},
		{
			t1:   types.Object([]types.Property{{Name: "a", Type: types.Int(32), CreateRequired: true, UpdateRequired: true}}),
			t2:   types.Object([]types.Property{{Name: "a", Type: types.Int(32), CreateRequired: true, UpdateRequired: true}, {Name: "b", Type: types.Int(32), ReadOptional: true}}),
			mode: &createOrUpdate,
		},
		{t1: types.Map(types.Text().WithCharLen(60)), t2: types.Map(types.Text().WithCharLen(60))},
		{t1: types.Boolean(), t2: types.Int(32), err: `"foo" property's type has changed from Boolean to Int(32)`},
		{t1: types.Int(32), t2: types.Int(64), err: `"foo" property's type has changed from Int(32) to Int(64)`},
		{t1: types.Uint(16), t2: types.Uint(8), err: `"foo" property's type has changed from Uint(16) to Uint(8)`},
		{t1: types.Int(32).WithIntRange(-100, 200), t2: types.Int(32).WithIntRange(-90, 200), err: `range of the "foo" property's type has changed from [-100,200] to [-90,200]`},
		{t1: types.Uint(16).WithUintRange(0, 10), t2: types.Uint(16).WithUintRange(0, 20), err: `range of the "foo" property's type has changed from [0,10] to [0,20]`},
		{t1: types.Float(64), t2: types.Float(32), err: `"foo" property's type has changed from Float(64) to Float(32)`},
		{t1: types.Float(32).AsReal(), t2: types.Float(32), err: `"foo" property's type has changed from real to non-real`},
		{t1: types.Float(64), t2: types.Float(64).AsReal(), err: `"foo" property's type has changed from non-real to real`},
		{t1: types.Float(64).WithFloatRange(0, 1.229073661), t2: types.Float(64).WithFloatRange(0, 1.229073662), err: `range of the "foo" property's type has changed from [0,1.229073661] to [0,1.229073662]`},
		{t1: types.Decimal(10, 3), t2: types.Decimal(12, 3), err: `precision of the "foo" property's type has changed from 10 to 12`},
		{t1: types.Decimal(10, 3), t2: types.Decimal(10, 2), err: `scale of the "foo" property's type has changed from 3 to 2`},
		{
			t1:  types.Decimal(10, 3).WithDecimalRange(decimal.NewFromInt(5), decimal.NewFromFloat(49.99)),
			t2:  types.Decimal(10, 3).WithDecimalRange(decimal.NewFromFloat(5.5), decimal.NewFromFloat(39.0)),
			err: `range of "foo" property's type has changed from [5,49.99] to [5.5,39]`,
		},
		{t1: types.Text(), t2: types.Text().WithCharLen(100), err: `character length of the "foo" property's type has changed from unbounded to 100`},
		{t1: types.Text().WithCharLen(5), t2: types.Text(), err: `character length of the "foo" property's type has changed from 5 to unbounded`},
		{t1: types.Text().WithCharLen(50), t2: types.Text().WithCharLen(60), err: `character length of the "foo" property's type has changed from 50 to 60`},
		{t1: types.Text(), t2: types.Text().WithByteLen(500), err: `byte length of the "foo" property's type has changed from unbounded to 500`},
		{t1: types.Text().WithByteLen(8), t2: types.Text(), err: `byte length of the "foo" property's type has changed from 8 to unbounded`},
		{t1: types.Text().WithByteLen(1200), t2: types.Text().WithByteLen(1250), err: `byte length of the "foo" property's type has changed from 1200 to 1250`},
		{t1: types.Text(), t2: types.Text().WithValues("b", "a"), err: `"foo" property was previously unrestricted but is now limited to specific values`},
		{t1: types.Text().WithValues("x", "y", "z"), t2: types.Text(), err: `"foo" property was previously limited to specific values but is now unrestricted`},
		{t1: types.Text().WithValues("x", "y", "z"), t2: types.Text().WithValues("z", "y"), err: `"foo" property allowed value "x" but it is no longer allowed`},
		{t1: types.Text().WithValues("x", "y", "z"), t2: types.Text().WithValues("y", "x"), err: `"foo" property allowed value "z" but it is no longer allowed`},
		{t1: types.Text().WithValues("x", "y"), t2: types.Text().WithValues("y", "z", "x"), err: `"foo" property previously disallowed value "z" but it now allows it`},
		{t1: types.Text(), t2: types.Text().WithRegexp(regexp.MustCompile(`^\w+`)), err: `regular expression of the "foo" property's type has changed from none to "^\w+"`},
		{t1: types.Text().WithRegexp(regexp.MustCompile(`^\d+`)), t2: types.Text(), err: `regular expression of the "foo" property's type has changed from "^\d+" to none`},
		{
			t1:  types.Text().WithRegexp(regexp.MustCompile(`^\d+`)),
			t2:  types.Text().WithRegexp(regexp.MustCompile(`^\w+`)),
			err: `regular expression of the "foo" property's type has changed from "^\d+" to "^\w+"`,
		},
		{t1: types.Array(types.Text()), t2: types.Array(types.Int(32)), err: `"foo[]" property's type has changed from Text to Int(32)`},
		{t1: types.Array(types.Array(types.UUID())), t2: types.Array(types.Array(types.Text())), err: `"foo[][]" property's type has changed from UUID to Text`},
		{t1: types.Array(types.Boolean()).WithMinElements(1), t2: types.Array(types.Boolean()), err: `minimum number of "foo" property elements has been changed from 1 to 0`},
		{t1: types.Array(types.Boolean()).WithMinElements(10), t2: types.Array(types.Boolean()).WithMinElements(12), err: `minimum number of "foo" property elements has been changed from 10 to 12`},
		{t1: types.Array(types.UUID()).WithUnique(), t2: types.Array(types.UUID()), err: `"foo" property elements were initially required to be unique, but it is no longer required`},
		{t1: types.Array(types.Inet()), t2: types.Array(types.Inet()).WithUnique(), err: `"foo" property elements were not required to be unique, but now it is required`},
		{t1: types.Object([]types.Property{{Name: "a", Type: types.Float(32)}}), t2: types.JSON(), err: `"foo" property's type has changed from Object to JSON`},
		{t1: types.JSON(), t2: types.Object([]types.Property{{Name: "a", Type: types.Float(32)}}), err: `"foo" property's type has changed from JSON to Object`},
		{
			t1:  types.Object([]types.Property{{Name: "x", Type: types.Boolean()}, {Name: "y", Type: types.JSON()}}),
			t2:  types.Object([]types.Property{{Name: "x", Type: types.Boolean()}}),
			err: `"foo.y" property no longer exists`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "z", Type: types.Array(types.Float(64))}}),
			t2:  types.Object([]types.Property{{Name: "z", Type: types.Array(types.Float(32))}}),
			err: `"foo.z[]" property's type has changed from Float(64) to Float(32)`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "z", Type: types.Object([]types.Property{{Name: "w", Type: types.UUID()}})}}),
			t2:  types.Object([]types.Property{{Name: "z", Type: types.Object([]types.Property{{Name: "y", Type: types.Int(8)}})}}),
			err: `"foo.z.w" property no longer exists`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: true}}),
			t2:   types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: false}}),
			mode: &createOnlyMode,
			err:  `"foo.a" property was previously required for creation but is no longer`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "b", Type: types.Boolean(), CreateRequired: false}}),
			t2:   types.Object([]types.Property{{Name: "b", Type: types.Boolean(), CreateRequired: true}}),
			mode: &createOnlyMode,
			err:  `"foo.b" property was not previously required for creation but it is now required`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: true}}),
			t2:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: false}}),
			err: `"foo.a" property was previously required for the update but is no longer`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: false}}),
			t2:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: true}}),
			err: `"foo.a" property was not previously required for the update but it is now required`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}}),
			t2:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.Text(), CreateRequired: true}}),
			mode: &createOnlyMode,
			err:  `"foo.d" property is required for creation but is not present in the schema`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}}),
			t2:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.Text(), UpdateRequired: true}}),
			mode: &updateOnlyMode,
			err:  `"foo.d" property is required for update but is not present in the schema`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}}),
			t2:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.Text(), CreateRequired: true}}),
			mode: &createOrUpdate,
			err:  `"foo.d" property is required for creation but is not present in the schema`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}}),
			t2:   types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.Text(), UpdateRequired: true}}),
			mode: &createOrUpdate,
			err:  `"foo.d" property is required for update but is not present in the schema`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: true}}),
			t2:  types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: false}}),
			err: `"foo.a" property was previously optional but it is now non-optional`,
		},
		{
			t1:   types.Object([]types.Property{{Name: "b", Type: types.Boolean(), ReadOptional: false}}),
			t2:   types.Object([]types.Property{{Name: "b", Type: types.Boolean(), ReadOptional: true}}),
			mode: &createOrUpdate,
			err:  `"foo.b" property was previously non-optional but it is now optional`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "c", Type: types.Boolean(), Nullable: true}}),
			t2:  types.Object([]types.Property{{Name: "c", Type: types.Boolean(), Nullable: false}}),
			err: `"foo.c" property was previously nullable but it is no longer nullable`,
		},
		{
			t1:  types.Object([]types.Property{{Name: "d", Type: types.Boolean(), Nullable: false}}),
			t2:  types.Object([]types.Property{{Name: "d", Type: types.Boolean(), Nullable: true}}),
			err: `"foo.d" property was previously non-nullable but is now nullable`,
		},
		{t1: types.Map(types.Text()), t2: types.Map(types.Int(32)), err: `"foo[]" property's type has changed from Text to Int(32)`},
		{t1: types.Map(types.Map(types.Text())), t2: types.Map(types.Map(types.Date())), err: `"foo[][]" property's type has changed from Text to Date`},
	}

	// Test when either t1 or t2 is invalid.
	t.Run("", func(t *testing.T) {
		err := CheckAlignment(types.Type{}, types.Type{}, nil)
		if err != nil {
			t.Fatalf("expected no error, got error %q", err)
		}
		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean()}}), nil)
		if err != nil {
			t.Fatalf("expected no error, got error %q", err)
		}
		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean()}}), &createOnlyMode)
		if err != nil {
			t.Fatalf("expected no error, got error %q", err)
		}
		expected := `"foo" property is required for creation`
		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean(), CreateRequired: true}}), &createOnlyMode)
		if err == nil {
			t.Fatalf("expected error %q, got no error", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected error %q, got error %q", expected, err)
		}
		expected = `"foo" property no longer exists`
		err = CheckAlignment(types.Object([]types.Property{{Name: "foo", Type: types.Boolean()}}), types.Type{}, &createOnlyMode)
		if err == nil {
			t.Fatalf("expected error %q, got no error", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected error %q, got error %q", expected, err)
		}
	})

	// Test all the other cases.
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			s1 := types.Object([]types.Property{{Name: "foo", Type: test.t1}})
			s2 := types.Object([]types.Property{{Name: "foo", Type: test.t2}})
			err := CheckAlignment(s1, s2, test.mode)
			if err != nil {
				if test.err == "" {
					t.Fatalf("expected no error, got error %q", err)
				}
				if test.err != err.Error() {
					t.Fatalf("expected error %q, got error %q", test.err, err)
				}
				return
			}
			if test.err != "" {
				t.Fatalf("expected error %q, got no error", test.err)
			}
		})
	}
}
