// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"regexp"
	"testing"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/types"
)

func Test_checkSchemaAlignment(t *testing.T) {

	var (
		createOnlyMode = state.CreateOnly
		updateOnlyMode = state.UpdateOnly
		createOrUpdate = state.CreateOrUpdate
	)

	tests := []struct {
		p1, p2 types.Property
		mode   *state.ExportMode
		err    string
	}{
		{p1: types.Property{Type: types.String()}, p2: types.Property{Type: types.String()}, mode: &createOnlyMode},
		{p1: types.Property{Type: types.String().WithMaxLength(100)}, p2: types.Property{Type: types.String().WithMaxLength(100)}},
		{p1: types.Property{Type: types.String().WithMaxByteLength(50)}, p2: types.Property{Type: types.String().WithMaxByteLength(50)}},
		{p1: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`))}, p2: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`))}, mode: &createOnlyMode},
		{p1: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`)).WithMaxByteLength(10)}, p2: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`)).WithMaxByteLength(10)}},
		{p1: types.Property{Type: types.Boolean()}, p2: types.Property{Type: types.Boolean()}},
		{p1: types.Property{Type: types.Int(32)}, p2: types.Property{Type: types.Int(32)}, mode: &createOnlyMode},
		{p1: types.Property{Type: types.Int(32).WithIntRange(-10, 100)}, p2: types.Property{Type: types.Int(32).WithIntRange(-10, 100)}},
		{p1: types.Property{Type: types.Uint(16).WithUintRange(0, 100)}, p2: types.Property{Type: types.Uint(16).WithUintRange(0, 100)}},
		{p1: types.Property{Type: types.Float(64)}, p2: types.Property{Type: types.Float(64)}},
		{p1: types.Property{Type: types.Float(32).AsReal()}, p2: types.Property{Type: types.Float(32).AsReal()}, mode: &createOnlyMode},
		{p1: types.Property{Type: types.Float(32).WithFloatRange(1.0, 5.67)}, p2: types.Property{Type: types.Float(32).WithFloatRange(1.0, 5.67)}},
		{p1: types.Property{Type: types.Decimal(10, 2)}, p2: types.Property{Type: types.Decimal(10, 2)}},
		{p1: types.Property{Type: types.DateTime()}, p2: types.Property{Type: types.DateTime()}},
		{p1: types.Property{Type: types.Date()}, p2: types.Property{Type: types.Date()}},
		{p1: types.Property{Type: types.Time()}, p2: types.Property{Type: types.Time()}},
		{p1: types.Property{Type: types.Year()}, p2: types.Property{Type: types.Year()}},
		{p1: types.Property{Type: types.UUID()}, p2: types.Property{Type: types.UUID()}, mode: &createOnlyMode},
		{p1: types.Property{Type: types.JSON()}, p2: types.Property{Type: types.JSON()}},
		{p1: types.Property{Type: types.Inet()}, p2: types.Property{Type: types.Inet()}},
		{p1: types.Property{Type: types.Array(types.Int(8))}, p2: types.Property{Type: types.Array(types.Int(8))}},
		{
			p1: types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), Nullable: true, Description: "a property"}})},
			p2: types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), Nullable: true}})},
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: true}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "a", Prefilled: "a", Type: types.Boolean(), CreateRequired: true}, {Name: "b", Type: types.Int(64)}})},
			mode: &createOnlyMode,
		},
		{
			p1: types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: true}})},
			p2: types.Property{Type: types.Object([]types.Property{{Name: "a", Prefilled: "a", Type: types.Boolean(), ReadOptional: true}, {Name: "b", Type: types.Int(64), CreateRequired: true}})},
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Int(32), CreateRequired: true, UpdateRequired: true}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Int(32), CreateRequired: true, UpdateRequired: true}, {Name: "b", Type: types.Int(32), ReadOptional: true}})},
			mode: &createOrUpdate,
		},
		{p1: types.Property{Type: types.Map(types.String().WithMaxLength(60))}, p2: types.Property{Type: types.Map(types.String().WithMaxLength(60))}},
		{p1: types.Property{Type: types.String()}, p2: types.Property{Type: types.String().WithMaxLength(100)}, err: `character length of the "foo" property's type has changed from unbounded to 100`},
		{p1: types.Property{Type: types.String().WithMaxLength(5)}, p2: types.Property{Type: types.String()}, err: `character length of the "foo" property's type has changed from 5 to unbounded`},
		{p1: types.Property{Type: types.String().WithMaxLength(50)}, p2: types.Property{Type: types.String().WithMaxLength(60)}, err: `character length of the "foo" property's type has changed from 50 to 60`},
		{p1: types.Property{Type: types.String()}, p2: types.Property{Type: types.String().WithMaxByteLength(500)}, err: `byte length of the "foo" property's type has changed from unbounded to 500`},
		{p1: types.Property{Type: types.String().WithMaxByteLength(8)}, p2: types.Property{Type: types.String()}, err: `byte length of the "foo" property's type has changed from 8 to unbounded`},
		{p1: types.Property{Type: types.String().WithMaxByteLength(1200)}, p2: types.Property{Type: types.String().WithMaxByteLength(1250)}, err: `byte length of the "foo" property's type has changed from 1200 to 1250`},
		{p1: types.Property{Type: types.String()}, p2: types.Property{Type: types.String().WithValues("b", "a")}, err: `"foo" property was previously unrestricted but is now limited to specific values`},
		{p1: types.Property{Type: types.String().WithValues("x", "y", "z")}, p2: types.Property{Type: types.String()}, err: `"foo" property was previously limited to specific values but is now unrestricted`},
		{p1: types.Property{Type: types.String().WithValues("x", "y", "z")}, p2: types.Property{Type: types.String().WithValues("z", "y")}, err: `"foo" property allowed value "x" but it is no longer allowed`},
		{p1: types.Property{Type: types.String().WithValues("x", "y", "z")}, p2: types.Property{Type: types.String().WithValues("y", "x")}, err: `"foo" property allowed value "z" but it is no longer allowed`},
		{p1: types.Property{Type: types.String().WithValues("x", "y")}, p2: types.Property{Type: types.String().WithValues("y", "z", "x")}, err: `"foo" property previously disallowed value "z" but it now allows it`},
		{p1: types.Property{Type: types.String()}, p2: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\w+`))}, err: `regular expression of the "foo" property's type has changed from none to "^\w+"`},
		{p1: types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`))}, p2: types.Property{Type: types.String()}, err: `regular expression of the "foo" property's type has changed from "^\d+" to none`},
		{
			p1:  types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\d+`))},
			p2:  types.Property{Type: types.String().WithRegexp(regexp.MustCompile(`^\w+`))},
			err: `regular expression of the "foo" property's type has changed from "^\d+" to "^\w+"`,
		},
		{p1: types.Property{Type: types.Boolean()}, p2: types.Property{Type: types.Int(32)}, err: `"foo" property's type has changed from boolean to int(32)`},
		{p1: types.Property{Type: types.Int(32)}, p2: types.Property{Type: types.Int(64)}, err: `"foo" property's type has changed from int(32) to int(64)`},
		{p1: types.Property{Type: types.Uint(16)}, p2: types.Property{Type: types.Uint(8)}, err: `"foo" property's type has changed from uint(16) to uint(8)`},
		{p1: types.Property{Type: types.Int(32).WithIntRange(-100, 200)}, p2: types.Property{Type: types.Int(32).WithIntRange(-90, 200)}, err: `range of the "foo" property's type has changed from [-100,200] to [-90,200]`},
		{p1: types.Property{Type: types.Uint(16).WithUintRange(0, 10)}, p2: types.Property{Type: types.Uint(16).WithUintRange(0, 20)}, err: `range of the "foo" property's type has changed from [0,10] to [0,20]`},
		{p1: types.Property{Type: types.Float(64)}, p2: types.Property{Type: types.Float(32)}, err: `"foo" property's type has changed from float(64) to float(32)`},
		{p1: types.Property{Type: types.Float(32).AsReal()}, p2: types.Property{Type: types.Float(32)}, err: `"foo" property's type has changed from real to non-real`},
		{p1: types.Property{Type: types.Float(64)}, p2: types.Property{Type: types.Float(64).AsReal()}, err: `"foo" property's type has changed from non-real to real`},
		{p1: types.Property{Type: types.Float(64).WithFloatRange(0, 1.229073661)}, p2: types.Property{Type: types.Float(64).WithFloatRange(0, 1.229073662)}, err: `range of the "foo" property's type has changed from [0,1.229073661] to [0,1.229073662]`},
		{p1: types.Property{Type: types.Decimal(10, 3)}, p2: types.Property{Type: types.Decimal(12, 3)}, err: `precision of the "foo" property's type has changed from 10 to 12`},
		{p1: types.Property{Type: types.Decimal(10, 3)}, p2: types.Property{Type: types.Decimal(10, 2)}, err: `scale of the "foo" property's type has changed from 3 to 2`},
		{
			p1:  types.Property{Type: types.Decimal(10, 3).WithDecimalRange(decimal.MustInt(5), decimal.MustParse("49.99"))},
			p2:  types.Property{Type: types.Decimal(10, 3).WithDecimalRange(decimal.MustParse("5.5"), decimal.MustInt(39))},
			err: `range of "foo" property's type has changed from [5,49.99] to [5.5,39]`,
		},
		{p1: types.Property{Type: types.Array(types.String())}, p2: types.Property{Type: types.Array(types.Int(32))}, err: `"foo[]" property's type has changed from string to int(32)`},
		{p1: types.Property{Type: types.Array(types.Array(types.UUID()))}, p2: types.Property{Type: types.Array(types.Array(types.String()))}, err: `"foo[][]" property's type has changed from uuid to string`},
		{p1: types.Property{Type: types.Array(types.Boolean()).WithMinElements(1)}, p2: types.Property{Type: types.Array(types.Boolean())}, err: `minimum number of "foo" property elements has been changed from 1 to 0`},
		{p1: types.Property{Type: types.Array(types.Boolean()).WithMinElements(10)}, p2: types.Property{Type: types.Array(types.Boolean()).WithMinElements(12)}, err: `minimum number of "foo" property elements has been changed from 10 to 12`},
		{p1: types.Property{Type: types.Array(types.UUID()).WithUnique()}, p2: types.Property{Type: types.Array(types.UUID())}, err: `"foo" property elements were initially required to be unique, but it is no longer required`},
		{p1: types.Property{Type: types.Array(types.Inet())}, p2: types.Property{Type: types.Array(types.Inet()).WithUnique()}, err: `"foo" property elements were not required to be unique, but now it is required`},
		{p1: types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Float(32)}})}, p2: types.Property{Type: types.JSON()}, err: `"foo" property's type has changed from object to json`},
		{p1: types.Property{Type: types.JSON()}, p2: types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Float(32)}})}, err: `"foo" property's type has changed from json to object`},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "x", Type: types.Boolean()}, {Name: "y", Type: types.JSON()}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "x", Type: types.Boolean()}})},
			err: `"foo.y" property no longer exists`,
		},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "z", Type: types.Array(types.Float(64))}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "z", Type: types.Array(types.Float(32))}})},
			err: `"foo.z[]" property's type has changed from float(64) to float(32)`,
		},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "z", Type: types.Object([]types.Property{{Name: "w", Type: types.UUID()}})}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "z", Type: types.Object([]types.Property{{Name: "y", Type: types.Int(8)}})}})},
			err: `"foo.z.w" property no longer exists`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: true}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), CreateRequired: false}})},
			mode: &createOnlyMode,
			err:  `"foo.a" property was previously required for creation but is no longer`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "b", Type: types.Boolean(), CreateRequired: false}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "b", Type: types.Boolean(), CreateRequired: true}})},
			mode: &createOnlyMode,
			err:  `"foo.b" property was not previously required for creation but it is now required`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: true}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: false}})},
			err:  `"foo.a" property was previously required for the update but is no longer`,
			mode: &createOrUpdate,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: false}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), UpdateRequired: true}})},
			err:  `"foo.a" property was not previously required for the update but it is now required`,
			mode: &createOrUpdate,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.String(), CreateRequired: true}})},
			mode: &createOnlyMode,
			err:  `"foo.d" property is required for creation but is not present in the schema`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.String(), UpdateRequired: true}})},
			mode: &updateOnlyMode,
			err:  `"foo.d" property is required for update but is not present in the schema`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.String(), CreateRequired: true}})},
			mode: &createOrUpdate,
			err:  `"foo.d" property is required for creation but is not present in the schema`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean()}, {Name: "d", Type: types.String(), UpdateRequired: true}})},
			mode: &createOrUpdate,
			err:  `"foo.d" property is required for update but is not present in the schema`,
		},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: true}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "a", Type: types.Boolean(), ReadOptional: false}})},
			err: `"foo.a" property was previously optional but it is now non-optional`,
		},
		{
			p1:   types.Property{Type: types.Object([]types.Property{{Name: "b", Type: types.Boolean(), ReadOptional: false}})},
			p2:   types.Property{Type: types.Object([]types.Property{{Name: "b", Type: types.Boolean(), ReadOptional: true}})},
			mode: &createOrUpdate,
		},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean(), Nullable: true}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "c", Type: types.Boolean(), Nullable: false}})},
			err: `"foo.c" property was previously nullable but it is no longer nullable`,
		},
		{
			p1:  types.Property{Type: types.Object([]types.Property{{Name: "d", Type: types.Boolean(), Nullable: false}})},
			p2:  types.Property{Type: types.Object([]types.Property{{Name: "d", Type: types.Boolean(), Nullable: true}})},
			err: `"foo.d" property was previously non-nullable but is now nullable`,
		},
		{p1: types.Property{Type: types.Map(types.String())}, p2: types.Property{Type: types.Map(types.Int(32))}, err: `"foo[]" property's type has changed from string to int(32)`},
		{p1: types.Property{Type: types.Map(types.Map(types.String()))}, p2: types.Property{Type: types.Map(types.Map(types.Date()))}, err: `"foo[][]" property's type has changed from string to date`},
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

		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean(), UpdateRequired: true}}), &createOnlyMode)
		if err != nil {
			t.Fatalf("expected no error, got error %q", err)
		}

		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean(), CreateRequired: true}}), &updateOnlyMode)
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

		expected = `"foo" property is required for update`
		err = CheckAlignment(types.Type{}, types.Object([]types.Property{{Name: "foo", Type: types.Boolean(), UpdateRequired: true}}), &updateOnlyMode)
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

		expected = `"foo" property no longer exists`
		err = CheckAlignment(types.Object([]types.Property{{Name: "foo", Type: types.Boolean()}}), types.Type{}, &updateOnlyMode)
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
			s1 := types.Object([]types.Property{{Name: "foo", Type: test.p1.Type, CreateRequired: test.p1.CreateRequired, UpdateRequired: test.p1.UpdateRequired}})
			s2 := types.Object([]types.Property{{Name: "foo", Type: test.p2.Type, CreateRequired: test.p2.CreateRequired, UpdateRequired: test.p2.UpdateRequired}})
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
