//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/shopspring/decimal"
)

func TestTypeSerialization(t *testing.T) {

	tests := []struct {
		Data string
		Type Type
	}{
		{
			Data: `null`,
			Type: Type{},
		},
		{
			Data: `{"name":"Text"}`,
			Type: Text(),
		}, {
			Data: `{"name":"Text","charLen":10}`,
			Type: Text(Chars(10)),
		}, {
			Data: `{"name":"Text","byteLen":24}`,
			Type: Text(Bytes(24)),
		}, {
			Data: `{"name":"Text","byteLen":80,"charLen":100}`,
			Type: Text(Chars(100), Bytes(80)),
		}, {
			Data: `{"name":"Text","enum":["a","b"]}`,
			Type: Text().WithEnum([]string{"a", "b"}),
		}, {
			Data: `{"name":"Int8","minimum":10}`,
			Type: Int8().WithIntRange(10, MaxInt8),
		}, {
			Data: `{"name":"Float","minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Float().WithFloatRange(-3.9936173, 8.00002312),
		}, {
			Data: `{"name":"Float32","minimum":3.99,"maximum":5.31}`,
			Type: Float32().WithFloatRange(3.99, 5.31),
		}, {
			Data: `{"name":"Decimal"}`,
			Type: Decimal(0, 0),
		}, {
			Data: `{"name":"Decimal","minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Decimal(0, 0).WithDecimalRange(
				decimal.RequireFromString("-3.9936173"),
				decimal.RequireFromString("8.00002312"),
			),
		}, {
			Data: `{"name":"Decimal","precision":10}`,
			Type: Decimal(10, 0),
		}, {
			Data: `{"name":"Decimal","precision":10,"scale":8}`,
			Type: Decimal(10, 8),
		}, {
			Data: `{"name":"DateTime","layout":"2006-01-02T15:04"}`,
			Type: DateTime("2006-01-02T15:04"),
		}, {
			Data: `{"name":"Array","itemType":{"name":"Text"}}`,
			Type: Array(Text()),
		}, {
			Data: `{"name":"Array","itemType":{"name":"Int"}}`,
			Type: Array(Int()),
		}, {
			Data: `{"name":"Array","minItems":2,"maxItems":8,"uniqueItems":true,"itemType":{"name":"Decimal"}}`,
			Type: Array(Decimal(0, 0)).WithMinItems(2).WithMaxItems(8).WithUnique(),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","type":{"name":"Text"}}],"flat":true}`,
			Type: Object([]Property{{Name: "email", Type: Text()}}).WithFlat(),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","type":{"name":"Text"}},{"name":"size","type":{"name":"Decimal"}}]}`,
			Type: Object([]Property{{Name: "email", Type: Text()}, {Name: "size", Type: Decimal(0, 0)}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","type":{"name":"Text"},"nullable":true}]}`,
			Type: Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
		},
	}

	for _, test := range tests {
		got, err := Parse(test.Data)
		if err != nil {
			t.Errorf("cannot unmarshal type %q: %s", test.Data, err)
			continue
		}
		if err = equalTypes(test.Type, got); err != nil {
			t.Errorf("%s: %s", test.Data, err)
			continue
		}
		b, err := test.Type.MarshalJSON()
		if err != nil {
			t.Errorf("%s: %s", test.Data, err)
			continue
		}
		if data := string(b); test.Data != data {
			t.Errorf("expecting %s, got %s", test.Data, data)
		}
	}

}

// equalTypes returns an error if t1 and t2 are not equal.
// It assumes that t1 is valid and validates t2.
func equalTypes(t1, t2 Type) error {
	// Type validity.
	if t1.Valid() != t2.Valid() {
		return fmt.Errorf("expected Valid() = %t, got %t", t1.Valid(), t2.Valid())
	}
	// Physical type.
	if t1.pt != t2.pt {
		if !t2.pt.Valid() {
			return fmt.Errorf("unknown physical type %d", t2.pt)
		}
		return fmt.Errorf("expected physical type %s, got %s", t1.pt, t2.pt)
	}
	// Logical type.
	if t1.lt != t2.lt {
		if t2.lt == 0 {
			return fmt.Errorf("expected logical type %s, got no logical type", t1.pt)
		}
		if !t2.lt.Valid() {
			return fmt.Errorf("unknown logical type %d", t2.pt)
		}
		return fmt.Errorf("expected logical type %s, got %s", t1.pt, t2.pt)
	}
	// Minimum and maximum.
	switch t1.pt {
	case PtInt, PtInt8, PtInt16, PtInt24:
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", t1.p, t1.s, t2.p, t2.s)
		}
	case PtUInt, PtUInt8, PtUInt16, PtUInt24:
		if t1.p != t2.p || t1.s != t2.s {
			return fmt.Errorf("expected range [%d,%d], got [%d,%d]", uint32(t1.p), uint32(t1.s), uint32(t2.p), uint32(t2.s))
		}
	case PtInt64, PtUInt64, PtFloat, PtFloat32:
		if t1.vl != t2.vl {
			if t1.vl == nil {
				return fmt.Errorf("expected no range, got %v", t2.vl)
			} else if t2.vl == nil {
				return fmt.Errorf("expected range %v, got no range", t1.vl)
			} else {
				return fmt.Errorf("expected range %v, got %v", t1.vl, t2.vl)
			}
		}
	case PtDecimal:
		if vl1, ok := t1.vl.(decimalRange); ok {
			vl2, ok := t2.vl.(decimalRange)
			if !ok || !vl1.min.Equal(vl2.min) || !vl1.max.Equal(vl2.max) {
				return fmt.Errorf("expected range %v, got range %v", t1.vl, t2.vl)
			}
		} else if t2.vl != nil {
			return fmt.Errorf("expected no-range, got range %v", t2.vl)
		}
	}
	// Precision, byte length or items minimum length.
	if t1.p != t2.p {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected precision %d, got %d", t1.p, t2.p)
		case PtText:
			return fmt.Errorf("expected byte length %d, got %d", t1.p, t2.p)
		case PtArray:
			return fmt.Errorf("expected items minimum length %d, got %d", t1.p, t2.p)
		}
		return fmt.Errorf("expected p == 0, got %d", t2.p)
	}
	// Scale, character length or items maximum length.
	if t1.s != t2.s {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected scale %d, got %d", t1.s, t2.s)
		case PtText:
			return fmt.Errorf("expected character length %d, got %d", t1.s, t2.s)
		case PtArray:
			return fmt.Errorf("expected items maximum length %d, got %d", t1.s, t2.s)
		}
		return fmt.Errorf("expected s == 0, got %d", t2.s)
	}
	// Regular expression or values.
	if t1.pt == PtText {
		switch vl1 := t1.vl.(type) {
		case nil:
			if t2.vl != nil {
				return fmt.Errorf("expected no regular expression or values, got a %T value", t2.vl)
			}
		case *regexp.Regexp:
			if t2.vl == nil {
				return errors.New("expected regular expression, got nil")
			}
			vl2, ok := t2.vl.(*regexp.Regexp)
			if !ok {
				return fmt.Errorf("expected regular expression, got a %T value", t2.vl)
			}
			if vl1.String() != vl2.String() {
				return fmt.Errorf("expected regular expression %s, got %s", vl1, vl2)
			}
		case []string:
			if t2.vl == nil {
				return errors.New("expected enum, got nil")
			}
			vl2, ok := t2.vl.([]string)
			if !ok {
				return fmt.Errorf("expected enum, got %T value", t2.vl)
			}
			if len(vl1) != len(vl2) {
				return fmt.Errorf("expected %d enum values, got %d", len(vl1), len(vl2))
			}
			for i, v1 := range vl1 {
				if v2 := vl2[i]; v1 != v2 {
					return fmt.Errorf("expected enum value %q, got %q", v1, v2)
				}
			}
		}
	}
	// Unique items and item type.
	if t1.pt == PtArray {
		if t1.unique != t2.unique {
			if t1.unique {
				return errors.New("expected unique items, got non-unique")
			}
			return errors.New("expected non-unique items, got unique")
		}
		if t2.vl == nil {
			return errors.New("expected item type, got nil")
		}
		if err := equalTypes(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	// Properties.
	if t1.pt == PtObject {
		if t2.vl == nil {
			return errors.New("expected properties, got nil")
		}
		properties2, ok := t2.vl.([]Property)
		if !ok {
			return fmt.Errorf("expected properties, got a %T value", t2.vl)
		}
		properties1 := t1.vl.([]Property)
		if len(properties1) != len(properties2) {
			return fmt.Errorf("expected %d properties, got %d", len(properties1), len(properties2))
		}
		for i, p1 := range properties1 {
			p2 := properties2[i]
			if p1.Name != p2.Name {
				return fmt.Errorf("expected property name %q, got %q", p1.Name, p2.Name)
			}
			if p1.Label != p2.Label {
				return fmt.Errorf("expected property label %q, got %q", p1.Label, p2.Label)
			}
			if p1.Description != p2.Description {
				return fmt.Errorf("expected property description %q, got %q", p1.Description, p2.Description)
			}
			if p1.Required != p2.Required {
				return fmt.Errorf("expected property key 'required' with value %t, got %t", p1.Required, p2.Required)
			}
			if err := equalTypes(p1.Type, p2.Type); err != nil {
				return err
			}
			if p1.Nullable != p2.Nullable {
				return fmt.Errorf("expected property key 'nullable' with value %t, got %t", p1.Nullable, p2.Nullable)
			}
		}
		if t1.flat != t2.flat {
			return fmt.Errorf("expected flat to be %t, got %t", t1.flat, t2.flat)
		}
	}
	// Value type.
	if t1.pt == PtMap {
		if t2.vl == nil {
			return errors.New("expected value type, got nil")
		}
		if err := equalTypes(t1.vl.(Type), t2.vl.(Type)); err != nil {
			return err
		}
	}
	return nil
}
