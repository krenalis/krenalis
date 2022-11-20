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
)

func TestSerialization(t *testing.T) {

	tests := []struct {
		Data    string
		Type    Type
		Resolve Resolver
	}{
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
			Data: `{"name":"Text","values":["a","b"]}`,
			Type: Text().WithValues([]string{"a", "b"}),
		}, {
			Data: `{"name":"Decimal"}`,
			Type: Decimal(0, 0),
		}, {
			Data: `{"name":"Decimal","precision":10}`,
			Type: Decimal(10, 0),
		}, {
			Data: `{"name":"Decimal","precision":10,"scale":8}`,
			Type: Decimal(10, 8),
		}, {
			Data: `{"name":"Array","items":{"name":"Text"}}`,
			Type: Array(Text()),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","type":{"name":"Text"}},{"name":"size","type":{"name":"Decimal"}}]}`,
			Type: Object([]Property{{Name: "email", Type: Text()}, {Name: "size", Type: Decimal(0, 0)}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","type":"Email"}]}`,
			Type: Object([]Property{{Name: "email", Type: Text(Chars(120)).WithRegexp(`@`).AsCustom("Email")}}),
			Resolve: func(name string) (Type, error) {
				if name != "Email" {
					return Type{}, ErrCustomTypeNotExist
				}
				return Text(Chars(120)).WithRegexp(`@`).AsCustom("Email"), nil
			},
		},
	}

	for _, test := range tests {
		got, err := UnmarshalType([]byte(test.Data), test.Resolve)
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
func equalTypes(t1, t2 Type) error {
	// Physical type.
	if t1.pt != t2.pt {
		if !t2.pt.Valid() {
			return fmt.Errorf("unknows physical type %d", t2.pt)
		}
		return fmt.Errorf("expected physical type %s, got %s", t1.pt, t2.pt)
	}
	// Logical type.
	if t1.lt != t2.lt {
		if t2.lt == 0 {
			return fmt.Errorf("expected logical type %s, got no logical type", t1.pt)
		}
		if !t2.lt.Valid() {
			return fmt.Errorf("unknows logical type %d", t2.pt)
		}
		return fmt.Errorf("expected logical type %s, got %s", t1.pt, t2.pt)
	}
	// Precision or byte length.
	if t1.p != t2.p {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected precision %d, got %d", t1.p, t2.p)
		case PtText:
			return fmt.Errorf("expected byte length %d, got %d", t1.p, t2.p)
		}
		return fmt.Errorf("expected p == 0, got %d", t2.p)
	}
	// Scale or character length.
	if t1.s != t2.s {
		switch t1.pt {
		case PtDecimal:
			return fmt.Errorf("expected scale %d, got %d", t1.s, t2.s)
		case PtText:
			return fmt.Errorf("expected character length %d, got %d", t1.s, t2.s)
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
				return errors.New("expected values, got nil")
			}
			vl2, ok := t2.vl.([]string)
			if !ok {
				return fmt.Errorf("expected values, got %T value", t2.vl)
			}
			if len(vl1) != len(vl2) {
				return fmt.Errorf("expected %d values, got %d", len(vl1), len(vl2))
			}
			for i, v1 := range vl1 {
				if v2 := vl2[i]; v1 != v2 {
					return fmt.Errorf("expected values element %q, got %q", v1, v2)
				}
			}
		}
	}
	// Array items type.
	if t1.pt == PtArray {
		if t2.vl == nil {
			return errors.New("expected array items type, got nil")
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
			if err := equalTypes(p1.Type, p2.Type); err != nil {
				return err
			}
		}
	}
	if t1.custom != t2.custom {
		if t1.custom == "" {
			return fmt.Errorf("expected non-custom type, got custom type %q", t2.custom)
		}
		if t2.custom == "" {
			return fmt.Errorf("expected custom type %q, got non-custom type", t1.custom)
		}
		return fmt.Errorf("expected custom type %q, got custom type %q", t1.custom, t2.custom)
	}
	return nil
}
