// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"bytes"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/go-cmp/cmp"
)

// TestMappingFailedConversionPreservesInput verifies that a failed conversion
// leaves the input unchanged, even if an earlier element was converted.
func TestMappingFailedConversionPreservesInput(t *testing.T) {

	source := types.Array(types.String())
	target := types.Array(types.Int(32))
	inSchema := types.Object([]types.Property{{Name: "value", Type: source}})
	outSchema := types.Object([]types.Property{{Name: "out", Type: target}})
	mapping, err := New(map[string]string{"out": "value"}, inSchema, outSchema, "")
	if err != nil {
		t.Fatalf("expected mapping construction to succeed, got %v", err)
	}

	input := map[string]any{"value": []any{"42", "invalid"}}
	expected := map[string]any{"value": []any{"42", "invalid"}}
	_, err = mapping.Transform(input, None)
	if err != nil {
		if _, ok := errors.AsType[ValidationError](err); !ok {
			t.Errorf("expected ValidationError, got %T: %v", err, err)
		}
	}
	if err == nil {
		t.Error("expected conversion error, got nil")
	}

	if !reflect.DeepEqual(input, expected) {
		t.Errorf("expected unchanged input %#v, got %#v", expected, input)
	}

}

// TestMappingMatchingValidation checks the output matching argument and its
// relationship with mapped paths.
func TestMappingMatchingValidation(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "other", Type: types.String()},
		{Name: "email", Type: types.String()},
		{Name: "number", Type: types.Int(32)},
		{Name: "uuid", Type: types.UUID()},
		{Name: "flag", Type: types.Boolean()},
		{Name: "payload", Type: types.JSON()},
		{Name: "items", Type: types.Array(types.String())},
		{Name: "values", Type: types.Map(types.String())},
		{Name: "parent", Type: types.Object([]types.Property{{Name: "email", Type: types.String()}})},
	})
	tests := []struct {
		name        string
		matching    string
		expressions map[string]string
		err         string
	}{
		{"none", "", nil, ""},
		{"string", "email", nil, ""},
		{"int", "number", nil, ""},
		{"uuid", "uuid", nil, ""},
		{"nested", "parent.email", nil, ""},
		{"invalid path", "parent..email", nil, "output matching property path is not valid"},
		{"trailing separator", "email.", nil, "output matching property path is not valid"},
		{"missing", "missing", nil, `property path "missing" does not exist`},
		{"object", "parent", nil, "output matching property must have type string, int, or uuid"},
		{"array", "items", nil, "output matching property must have type string, int, or uuid"},
		{"map", "values", nil, "output matching property must have type string, int, or uuid"},
		{"boolean", "flag", nil, "output matching property must have type string, int, or uuid"},
		{"json", "payload", nil, "output matching property must have type string, int, or uuid"},
		{"mapped", "email", map[string]string{"email": "'value'"}, "mapping cannot map over the output matching property"},
		{"mapped ancestor", "parent.email", map[string]string{"parent": "json_parse('{}')"}, "mapping cannot map over the output matching property"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expressions := test.expressions
			if expressions == nil {
				expressions = map[string]string{"other": "'value'"}
			}
			_, err := New(expressions, types.Type{}, schema, test.matching)
			if err != nil {
				if test.err == "" {
					t.Fatalf("expected mapping construction to succeed, got %v", err)
				}
				if err.Error() != test.err {
					t.Fatalf("expected error %q, got %q", test.err, err)
				}
				return
			}
			if test.err != "" {
				t.Fatalf("expected error %q, got nil", test.err)
			}
		})
	}

}

// TestMappingOutputValidation checks that required properties are present
// after path assembly.
func TestMappingOutputValidation(t *testing.T) {

	child := types.Object([]types.Property{
		{Name: "created", Type: types.String(), CreateRequired: true},
		{Name: "updated", Type: types.String(), UpdateRequired: true, Nullable: true},
		{Name: "optional", Type: types.String()},
	})
	matching := types.Property{Name: "email", Type: types.String(), CreateRequired: true}
	matchingOnUpdate := matching
	matchingOnUpdate.UpdateRequired = true
	optionalName := types.Property{Name: "name", Type: types.String()}
	requiredName := types.Property{Name: "name", Type: types.String(), CreateRequired: true, UpdateRequired: true}
	tests := []struct {
		name              string
		properties        []types.Property
		matching          string
		inSchema          types.Type
		attributes        map[string]any
		expressions       map[string]string
		expected          map[string]any
		createMissingPath string
		updateMissingPath string
	}{
		{
			name:              "unmapped required property",
			properties:        []types.Property{{Name: "required", Type: types.String(), CreateRequired: true, UpdateRequired: true}},
			expressions:       map[string]string{"other": "'value'"},
			expected:          map[string]any{"other": "value"},
			createMissingPath: "required",
			updateMissingPath: "required",
		},
		{
			name:              "unmapped nullable required property",
			properties:        []types.Property{{Name: "required", Type: types.String(), CreateRequired: true, Nullable: true}},
			expressions:       map[string]string{"other": "'value'"},
			expected:          map[string]any{"other": "value"},
			createMissingPath: "required",
		},
		{
			name:              "omitted required ancestor",
			properties:        []types.Property{{Name: "parent", Type: child, CreateRequired: true, UpdateRequired: true}},
			expressions:       map[string]string{"parent.optional": "null"},
			expected:          map[string]any{},
			createMissingPath: "parent",
			updateMissingPath: "parent",
		},
		{
			name:        "absent optional ancestor",
			properties:  []types.Property{{Name: "parent", Type: child}},
			expressions: map[string]string{"other": "'value'"},
			expected:    map[string]any{"other": "value"},
		},
		{
			name:        "required leaf omitted with optional ancestor",
			properties:  []types.Property{{Name: "parent", Type: child}},
			expressions: map[string]string{"parent.created": "null"},
			expected:    map[string]any{},
		},
		{
			name:              "empty object missing required children",
			properties:        []types.Property{{Name: "parent", Type: child}},
			expressions:       map[string]string{"parent": "json_parse('{}')"},
			expected:          map[string]any{"parent": map[string]any{}},
			createMissingPath: "parent.created",
			updateMissingPath: "parent.updated",
		},
		{
			name:              "assembled object missing required children",
			properties:        []types.Property{{Name: "parent", Type: child}},
			expressions:       map[string]string{"parent.optional": "'value'"},
			expected:          map[string]any{"parent": map[string]any{"optional": "value"}},
			createMissingPath: "parent.created",
			updateMissingPath: "parent.updated",
		},
		{
			name: "assembled object with multiple levels",
			properties: []types.Property{
				{Name: "parent", Type: types.Object([]types.Property{{Name: "child", Type: child}})},
			},
			expressions:       map[string]string{"parent.child.optional": "'value'"},
			expected:          map[string]any{"parent": map[string]any{"child": map[string]any{"optional": "value"}}},
			createMissingPath: "parent.child.created",
			updateMissingPath: "parent.child.updated",
		},
		{
			name:              "assembled object missing update property",
			properties:        []types.Property{{Name: "parent", Type: child}},
			expressions:       map[string]string{"parent.created": "'value'"},
			expected:          map[string]any{"parent": map[string]any{"created": "value"}},
			updateMissingPath: "parent.updated",
		},
		{
			name:        "assembled required siblings including nullable property",
			properties:  []types.Property{{Name: "parent", Type: child, CreateRequired: true, UpdateRequired: true}},
			expressions: map[string]string{"other": "'value'", "parent.created": "'value'", "parent.updated": "null", "parent.optional": "'extra'"},
			expected:    map[string]any{"other": "value", "parent": map[string]any{"created": "value", "updated": nil, "optional": "extra"}},
		},
		{
			name:        "null required ancestor",
			properties:  []types.Property{{Name: "parent", Type: child, CreateRequired: true, UpdateRequired: true, Nullable: true}},
			expressions: map[string]string{"parent": "null"},
			expected:    map[string]any{"parent": nil},
		},
		{
			name:        "null required JSON property",
			properties:  []types.Property{{Name: "required", Type: types.JSON(), CreateRequired: true, UpdateRequired: true}},
			expressions: map[string]string{"required": "null"},
			expected:    map[string]any{"required": json.Value("null")},
		},

		{
			name:        "required matching property",
			properties:  []types.Property{matchingOnUpdate},
			matching:    "email",
			expressions: map[string]string{"other": "'value'"},
			expected:    map[string]any{"other": "value"},
		},
		{
			name:              "required ancestor containing only matching",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matching}), CreateRequired: true, UpdateRequired: true}},
			matching:          "parent.email",
			expressions:       map[string]string{"other": "'value'"},
			expected:          map[string]any{"other": "value"},
			updateMissingPath: "parent",
		},
		{
			name:        "required ancestor containing only update-required matching",
			properties:  []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate}), CreateRequired: true, UpdateRequired: true}},
			matching:    "parent.email",
			expressions: map[string]string{"other": "'value'"},
			expected:    map[string]any{"other": "value"},
		},
		{
			name:              "required ancestor with null optional sibling",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matching, optionalName}), CreateRequired: true, UpdateRequired: true}},
			matching:          "parent.email",
			expressions:       map[string]string{"parent.name": "null"},
			expected:          map[string]any{},
			updateMissingPath: "parent",
		},
		{
			name:        "required ancestor with null optional sibling and update-required matching",
			properties:  []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate, optionalName}), CreateRequired: true, UpdateRequired: true}},
			matching:    "parent.email",
			expressions: map[string]string{"parent.name": "null"},
			expected:    map[string]any{},
		},
		{
			name:              "optional ancestor with null required sibling",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matching, requiredName})}},
			matching:          "parent.email",
			expressions:       map[string]string{"parent.name": "null"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
		},
		{
			name:              "optional ancestor with null required sibling and update-required matching",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate, requiredName})}},
			matching:          "parent.email",
			expressions:       map[string]string{"parent.name": "null"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
			updateMissingPath: "parent.name",
		},
		{
			name:        "required sibling present",
			properties:  []types.Property{{Name: "parent", Type: types.Object([]types.Property{matching, requiredName}), CreateRequired: true, UpdateRequired: true}},
			matching:    "parent.email",
			expressions: map[string]string{"parent.name": "'value'"},
			expected:    map[string]any{"parent": map[string]any{"name": "value"}},
		},
		{
			name:        "required sibling present with update-required matching",
			properties:  []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate, requiredName}), CreateRequired: true, UpdateRequired: true}},
			matching:    "parent.email",
			expressions: map[string]string{"parent.name": "'value'"},
			expected:    map[string]any{"parent": map[string]any{"name": "value"}},
		},
		{
			name:              "two matching ancestors with required sibling",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{requiredName, {Name: "child", Type: types.Object([]types.Property{matching})}})}},
			matching:          "parent.child.email",
			expressions:       map[string]string{"parent.name": "null"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
		},
		{
			name:              "two matching ancestors with required sibling and update-required matching",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{requiredName, {Name: "child", Type: types.Object([]types.Property{matchingOnUpdate})}})}},
			matching:          "parent.child.email",
			expressions:       map[string]string{"parent.name": "null"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
			updateMissingPath: "parent.name",
		},
		{
			name:              "required second ancestor with present first ancestor",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{optionalName, {Name: "child", Type: types.Object([]types.Property{matching}), CreateRequired: true, UpdateRequired: true}})}},
			matching:          "parent.child.email",
			expressions:       map[string]string{"parent.name": "'value'"},
			expected:          map[string]any{"parent": map[string]any{"name": "value"}},
			updateMissingPath: "parent.child",
		},
		{
			name:              "reused nullable required sibling",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate, requiredName})}},
			matching:          "parent.email",
			inSchema:          types.Object([]types.Property{{Name: "source", Type: types.String(), Nullable: true}}),
			attributes:        map[string]any{"source": nil},
			expressions:       map[string]string{"parent.name": "source"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
			updateMissingPath: "parent.name",
		},
		{
			name:              "reused object sibling preserves matching ancestor checks",
			properties:        []types.Property{{Name: "parent", Type: types.Object([]types.Property{matchingOnUpdate, requiredName, {Name: "details", Type: types.Object([]types.Property{requiredName})}})}},
			matching:          "parent.email",
			inSchema:          types.Object([]types.Property{{Name: "source", Type: types.Object([]types.Property{requiredName}), Nullable: true}}),
			attributes:        map[string]any{"source": nil},
			expressions:       map[string]string{"parent.details": "source"},
			expected:          map[string]any{},
			createMissingPath: "parent.name",
			updateMissingPath: "parent.name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			properties := append([]types.Property{{Name: "other", Type: types.String()}}, test.properties...)
			mapping, err := New(test.expressions, test.inSchema, types.Object(properties), test.matching)
			if err != nil {
				t.Fatalf("expected mapping creation to succeed, got %v", err)
			}

			purposes := []struct {
				name        string
				purpose     Purpose
				missingPath string
				operation   string
			}{
				{"none", None, "", ""},
				{"create", Create, test.createMissingPath, "creation"},
				{"update", Update, test.updateMissingPath, "update"},
			}
			for _, purpose := range purposes {
				t.Run(purpose.name, func(t *testing.T) {
					got, err := mapping.Transform(test.attributes, purpose.purpose)
					if err != nil {
						if purpose.missingPath == "" {
							t.Fatalf("expected transformation to succeed, got %v", err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("expected ValidationError, got %T: %v", err, err)
						}
						expected := "property «" + purpose.missingPath + "» is missing"
						expected += " but it is required for " + purpose.operation
						if err.Error() != expected {
							t.Fatalf("expected error %q, got %q", expected, err)
						}
						return
					}
					if purpose.missingPath != "" {
						t.Fatalf("expected validation error for missing property %q, got %#v", purpose.missingPath, got)
					}
					if !reflect.DeepEqual(got, test.expected) {
						t.Fatalf("expected output %#v, got %#v", test.expected, got)
					}
				})
			}

		})
	}

}

// TestMappingRequiredExpressions checks that required properties are validated
// in conditional expression results and nested objects.
func TestMappingRequiredExpressions(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "name", Type: types.String(), ReadOptional: true, CreateRequired: true},
	})
	array := types.Array(object)
	nested := types.Object([]types.Property{{Name: "child", Type: object}})
	inSchema := types.Object([]types.Property{
		{Name: "items", Type: array},
		{Name: "nested", Type: nested},
	})
	tests := []struct {
		expression  string
		typ         types.Type
		missingPath string
	}{
		{"coalesce(items, null)", array, "target[0].name"},
		{"coalesce(null, items)", array, "target[0].name"},
		{"if(true, items, null)", array, "target[0].name"},
		{"if(false, null, items)", array, "target[0].name"},
		{"nested", nested, "target.child.name"},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {

			outSchema := types.Object([]types.Property{{Name: "target", Type: test.typ, CreateRequired: true}})
			mapping, err := New(map[string]string{"target": test.expression}, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("expected mapping creation to succeed, got %v", err)
			}
			input := map[string]any{
				"items":  []any{map[string]any{}},
				"nested": map[string]any{"child": map[string]any{}},
			}
			got, err := mapping.Transform(input, Create)
			if err != nil {
				if _, ok := errors.AsType[ValidationError](err); !ok {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				expected := "property «" + test.missingPath + "» is missing but it is required for creation"
				if err.Error() != expected {
					t.Fatalf("expected error %q, got %q", expected, err)
				}
			}
			if err == nil {
				t.Fatalf("expected validation error for missing property %q, got %#v", test.missingPath, got)
			}

			input = map[string]any{
				"items":  []any{map[string]any{"name": "example"}},
				"nested": map[string]any{"child": map[string]any{"name": "example"}},
			}
			_, err = mapping.Transform(input, Create)
			if err != nil {
				t.Fatalf("expected valid input to succeed, got %v", err)
			}

		})
	}

}

// TestMappingRequiredProperties checks that required properties are validated
// for each purpose, including within nested containers.
func TestMappingRequiredProperties(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "created", Type: types.String(), CreateRequired: true, ReadOptional: true, Nullable: true},
		{Name: "updated", Type: types.String(), UpdateRequired: true, ReadOptional: true, Nullable: true},
		{Name: "other", Type: types.String()},
	})
	tests := []struct {
		name    string
		purpose Purpose
		fields  map[string]any
		missing string
	}{
		{"none missing", None, nil, ""},
		{"none present", None, map[string]any{"created": nil, "updated": nil}, ""},
		{"create missing", Create, nil, "created"},
		{"create missing with updated", Create, map[string]any{"updated": nil}, "created"},
		{"create present null", Create, map[string]any{"created": nil}, ""},
		{"create present value", Create, map[string]any{"created": "value"}, ""},
		{"create both present", Create, map[string]any{"created": nil, "updated": nil}, ""},
		{"update missing", Update, nil, "updated"},
		{"update missing with created", Update, map[string]any{"created": nil}, "updated"},
		{"update present null", Update, map[string]any{"updated": nil}, ""},
		{"update present value", Update, map[string]any{"updated": "value"}, ""},
		{"update both present", Update, map[string]any{"created": nil, "updated": nil}, ""},
	}

	for _, shape := range []string{"object", "array", "map", "nested"} {
		for _, mode := range []string{"equal types", "map", "json"} {
			for _, test := range tests {
				name := shape + "/" + mode + "/" + test.name
				t.Run(name, func(t *testing.T) {

					source, destination := object, object
					value := map[string]any{"other": "value"}
					maps.Copy(value, test.fields)
					var expected any = maps.Clone(value)
					complete := map[string]any{"created": "created", "updated": "updated", "other": "complete"}
					expectedComplete := maps.Clone(complete)
					switch mode {
					case "equal types", "json":
						// Keep the object type; encode JSON after wrapping.
					case "map":
						source = types.Map(types.JSON())
						for _, object := range []map[string]any{value, complete} {
							for name, item := range object {
								encoded, err := json.Marshal(item)
								if err != nil {
									t.Fatalf("expected JSON encoding to succeed, got %v", err)
								}
								object[name] = encoded
							}
						}
					default:
						t.Fatalf("expected known mode, got %q", mode)
					}

					var input any = value
					errorPath := "target"
					switch shape {
					case "object":
						// Keep the object at the root.
					case "array":
						source, destination = types.Array(source), types.Array(destination)
						input, expected = []any{complete, input}, []any{expectedComplete, expected}
						errorPath = "target[1]"
					case "map":
						source, destination = types.Map(source), types.Map(destination)
						input = map[string]any{"valid": complete, "key": input}
						expected = map[string]any{"valid": expectedComplete, "key": expected}
						errorPath = `target["key"]`
					case "nested":
						source = types.Array(types.Map(types.Array(source)))
						destination = types.Array(types.Map(types.Array(destination)))
						input = []any{
							map[string]any{"valid": []any{complete}},
							map[string]any{"valid": []any{complete}, "key": []any{complete, input}},
						}
						expected = []any{
							map[string]any{"valid": []any{expectedComplete}},
							map[string]any{"valid": []any{expectedComplete}, "key": []any{expectedComplete, expected}},
						}
						errorPath = `target[1]["key"][1]`
					default:
						t.Fatalf("expected known shape, got %q", shape)
					}
					if mode == "json" {
						source = types.JSON()
						encoded, err := json.Marshal(input)
						if err != nil {
							t.Fatalf("expected JSON encoding to succeed, got %v", err)
						}
						input = encoded
					}

					inSchema := types.Object([]types.Property{{Name: "source", Type: source}})
					outSchema := types.Object([]types.Property{{Name: "target", Type: destination}})
					mapping, err := New(map[string]string{"target": "source"}, inSchema, outSchema, "")
					if err != nil {
						t.Fatalf("expected mapping creation to succeed, got %v", err)
					}
					got, err := mapping.Transform(map[string]any{"source": input}, test.purpose)
					if err != nil {
						if test.missing == "" {
							t.Fatalf("expected transformation to succeed, got %v", err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("expected ValidationError, got %T: %v", err, err)
						}
						operation := "creation"
						if test.purpose == Update {
							operation = "update"
						}
						expectedError := "property «" + errorPath + "." + test.missing
						expectedError += "» is missing but it is required for " + operation
						if err.Error() != expectedError {
							t.Fatalf("expected error %q, got %q", expectedError, err)
						}
						return
					}
					if test.missing != "" {
						t.Fatalf("expected validation error, got %#v", got)
					}
					expectedOutput := map[string]any{"target": expected}
					if !reflect.DeepEqual(got, expectedOutput) {
						t.Fatalf("expected output %#v, got %#v", expectedOutput, got)
					}

				})
			}
		}
	}

}

// TestMappingRequiredPropertyMapKey checks how map keys are formatted and
// truncated in missing-property validation errors.
func TestMappingRequiredPropertyMapKey(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "required", Type: types.String(), CreateRequired: true, ReadOptional: true},
	})
	schema := types.Object([]types.Property{{Name: "target", Type: types.Map(object)}})
	mapping, err := New(map[string]string{"target": "target"}, schema, schema, "")
	if err != nil {
		t.Fatalf("expected mapping creation to succeed, got %v", err)
	}
	tests := []struct {
		name string
		key  string
		path string
	}{
		{"plain", "key", `target["key"].required`},
		{"escaped", "a\"\n»", `target["a\"\n≫"].required`},
		{"max length", strings.Repeat("x", 128), `target["` + strings.Repeat("x", 128) + `"].required`},
		{"truncated", "é" + strings.Repeat("x", 126) + "y", `target["é` + strings.Repeat("x", 126) + `…"].required`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := map[string]any{"target": map[string]any{test.key: map[string]any{}}}
			got, err := mapping.Transform(input, Create)
			if err != nil {
				if _, ok := errors.AsType[ValidationError](err); !ok {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				expected := "property «" + test.path + "» is missing but it is required for creation"
				if err.Error() != expected {
					t.Fatalf("expected error %q, got %q", expected, err)
				}
				return
			}
			t.Fatalf("expected validation error, got %#v", got)
		})
	}

}

// TestMappingReusePreservesBehavior checks that skipping conversions preserves
// results and validation errors for each purpose.
func TestMappingReusePreservesBehavior(t *testing.T) {

	var clone func(any) any
	clone = func(value any) any {
		switch value := value.(type) {
		case map[string]any:
			copied := maps.Clone(value)
			for key, item := range copied {
				copied[key] = clone(item)
			}
			return copied
		case []any:
			copied := slices.Clone(value)
			for i, item := range copied {
				copied[i] = clone(item)
			}
			return copied
		case json.Value:
			return slices.Clone(value)
		default:
			return value
		}
	}

	object := types.Object([]types.Property{
		{Name: "name", Type: types.String(), CreateRequired: true, UpdateRequired: true},
		{Name: "value", Type: types.Int(32), CreateRequired: true, UpdateRequired: true},
	})
	metadata := types.Object([]types.Property{
		{Name: "value", Type: types.Int(32), CreateRequired: true, UpdateRequired: true},
		{
			Name:           "name",
			Type:           types.String(),
			CreateRequired: true,
			UpdateRequired: true,
			Prefilled:      "preset",
			DisplayName:    "Name",
			Description:    "Display name",
		},
	})
	differentName := types.Object([]types.Property{
		{Name: "name", Type: types.String()},
		{Name: "other", Type: types.Int(32)},
	})
	optional := types.Object([]types.Property{
		{Name: "name", Type: types.String(), ReadOptional: true, CreateRequired: true, UpdateRequired: true},
	})
	createRequired := types.Object([]types.Property{
		{Name: "name", Type: types.String(), ReadOptional: true, CreateRequired: true},
	})
	updateRequired := types.Object([]types.Property{
		{Name: "name", Type: types.String(), ReadOptional: true, UpdateRequired: true},
	})
	payload := types.Object([]types.Property{{Name: "payload", Type: types.JSON(), CreateRequired: true}})
	nullablePayload := types.Object([]types.Property{
		{Name: "payload", Type: types.JSON(), Nullable: true, CreateRequired: true},
	})
	nullable := types.Object([]types.Property{{Name: "name", Type: types.String(), Nullable: true}})
	nonNullable := types.Object([]types.Property{{Name: "name", Type: types.String()}})
	readOptional := types.Object([]types.Property{{Name: "name", Type: types.String(), ReadOptional: true}})
	// Profile properties are read-optional and not nullable; destination properties are not read-optional.
	profile := types.Object([]types.Property{
		{Name: "name", Type: types.String(), ReadOptional: true},
		{Name: "value", Type: types.Int(32), ReadOptional: true},
	})
	destination := types.Object([]types.Property{
		{Name: "name", Type: types.String(), Nullable: true},
		{Name: "value", Type: types.Int(32)},
	})
	nestedProfile := types.Object([]types.Property{{Name: "child", Type: profile, ReadOptional: true}})
	nestedDestination := types.Object([]types.Property{{Name: "child", Type: destination}})
	wide := types.Object([]types.Property{{Name: "number", Type: types.Int(32)}})
	narrow := types.Object([]types.Property{{Name: "number", Type: types.Int(8)}})
	array := types.Array(object)
	alternatives := types.Object([]types.Property{
		{Name: "text", Type: types.String(), Nullable: true},
		{Name: "number", Type: types.Int(32)},
	})
	value := map[string]any{"name": "value", "value": 1}
	allPurposes := []Purpose{None, Create, Update}
	writePurposes := []Purpose{Create, Update}
	tests := []struct {
		name          string
		source        types.Type
		target        types.Type
		value         any
		expression    string
		wantConvert   bool
		errorPurposes []Purpose
	}{
		{"string", types.String(), types.String(), "", "source", false, nil},
		{"boolean", types.Boolean(), types.Boolean(), false, "source", false, nil},
		{"unsigned", types.Int(32).Unsigned(), types.Int(32).Unsigned(), uint(0), "source", false, nil},
		{"float", types.Float(64), types.Float(64), 1.5, "source", false, nil},
		{"decimal", types.Decimal(5, 0), types.Decimal(5, 0), decimal.MustInt(123), "source", false, nil},
		{"datetime", types.DateTime(), types.DateTime(), time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), "source", false, nil},
		{"equal optional object", optional, optional, map[string]any{}, "source", false, writePurposes},
		{"property metadata", types.AsRole(object, types.Source), metadata, value, "source", false, nil},
		{"added required property", nonNullable, object, map[string]any{"name": "value"}, "source", true, writePurposes},
		{"different property name", object, differentName, value, "source", true, nil},
		{"read optional", readOptional, nonNullable, map[string]any{}, "source", false, nil},
		{"empty profile object", profile, destination, map[string]any{}, "source", false, nil},
		{"partial profile object", profile, destination, map[string]any{"value": 1}, "source", false, nil},
		{"complete profile object", profile, destination, value, "source", false, nil},
		{"nested profile object", nestedProfile, nestedDestination, map[string]any{"child": map[string]any{"value": 1}}, "source", false, nil},
		{"profile object array", types.Array(profile), types.Array(destination), []any{map[string]any{}, value}, "source", false, nil},
		{"profile object map", types.Map(profile), types.Map(destination), map[string]any{"key": map[string]any{"name": "value"}}, "source", false, nil},
		{"required profile property", profile, object, map[string]any{"value": 1}, "source", false, writePurposes},
		{"complete required profile", profile, object, value, "source", false, nil},
		{"required profile array", types.Array(profile), array, []any{value, map[string]any{"value": 1}}, "source", false, writePurposes},
		{"required profile map", types.Map(profile), types.Map(object), map[string]any{"key": map[string]any{"value": 1}}, "source", false, writePurposes},
		{"required profile coalesce", profile, object, map[string]any{"value": 1}, "coalesce(source, null)", false, writePurposes},
		{"required profile conditional", profile, object, map[string]any{"value": 1}, "if(true, source, null)", false, writePurposes},
		{"empty array", types.Array(types.Int(32)), types.Array(types.Int(32)), []any{}, "source", false, nil},
		{"nil map", types.Map(types.String()), types.Map(types.String()), map[string]any(nil), "source", false, nil},
		{"array write metadata", types.Array(types.AsRole(object, types.Source)), array, []any{value}, "source", false, nil},
		{"map write metadata", types.Map(types.AsRole(object, types.Source)), types.Map(object), map[string]any{"key": value}, "source", false, nil},
		{"optional create field", types.AsRole(createRequired, types.Source), createRequired, map[string]any{}, "source", false, []Purpose{Create}},
		{"optional update field", types.AsRole(updateRequired, types.Source), updateRequired, map[string]any{}, "source", false, []Purpose{Update}},
		{"array element constraints", types.Array(wide), types.Array(narrow), []any{map[string]any{"number": 1000}}, "source", true, allPurposes},
		{"optional map write field", types.Map(types.AsRole(optional, types.Source)), types.Map(optional), map[string]any{"key": map[string]any{}}, "source", false, writePurposes},
		{"nil JSON write metadata", types.AsRole(payload, types.Source), payload, map[string]any{"payload": nil}, "source", false, nil},
		{"JSON null write metadata", types.AsRole(payload, types.Source), payload, map[string]any{"payload": json.Value("null")}, "source", false, nil},
		{"nullable JSON write metadata", types.AsRole(nullablePayload, types.Source), nullablePayload, map[string]any{"payload": nil}, "source", false, nil},
		{"nullable widening", nonNullable, nullable, map[string]any{"name": "value"}, "source", false, nil},
		{"nullability", nullable, nonNullable, map[string]any{"name": nil}, "source", true, allPurposes},
		{"array minimum", types.Array(types.Int(32)), types.Array(types.Int(32)).WithMinElements(2), []any{1}, "source", true, allPurposes},
		{"array maximum", types.Array(types.Int(32)), types.Array(types.Int(32)).WithMaxElements(1), []any{1, 2}, "source", true, allPurposes},
		{"array uniqueness", types.Array(types.Int(32)), types.Array(types.Int(32)).WithUnique(), []any{1, 1}, "source", true, allPurposes},
		{"semantics", types.String(), types.String().AsCountry(types.ISO3166Alpha2), "ZZ", "source", true, allPurposes},
		{"JSON null", types.JSON(), types.JSON(), json.Value("null"), "source", false, nil},
		{"nil JSON", types.JSON(), types.JSON(), nil, "source", false, nil},
		{"JSON array", types.Array(types.JSON()), types.Array(types.JSON()), []any{json.Value("null")}, "source", false, nil},
		{"nested conditionals", array, array, []any{value}, "coalesce(if(false, null, source), source)", false, nil},
		{"coalesce conversion", alternatives, types.String(), map[string]any{"text": nil, "number": 1}, "coalesce(source.text, source.number)", true, nil},
		{"conditional conversion", types.String(), types.Int(32), "10", "if(true, source, null)", true, nil},
		{"else conversion", types.String(), types.String(), "value", "if(false, source, len(source))", true, nil},
		{"function", types.String(), types.String(), "VALUE", "lower(source)", true, nil},
		{"constant", types.String(), types.Int(32), "value", "'10'", false, nil},
		{"concatenation", types.Int(32), types.String(), 7, "'prefix' source", false, nil},
		{"multiple parts", types.Int(32), types.String(), 7, "source '-' source", false, nil},
		{"conditional prefix", types.Int(32), types.Int(32), 7, "'prefix' if(true, source, source)", true, allPurposes},
		{"conditional suffix", types.Int(32), types.Int(32), 7, "if(true, source, source) 'suffix'", true, allPurposes},
	}
	purposes := []struct {
		name    string
		purpose Purpose
	}{
		{"none", None},
		{"create", Create},
		{"update", Update},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "source", Type: test.source, Nullable: true}})
			outSchema := types.Object([]types.Property{
				{Name: "target", Type: test.target, CreateRequired: true, UpdateRequired: true},
			})
			expressions := map[string]string{"target": test.expression}
			mapping, err := New(expressions, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("expected mapping creation to succeed, got %v", err)
			}
			gotConvert := mapping.expressions[0].convert
			if gotConvert != test.wantConvert {
				t.Fatalf("expected conversion %v, got %v", test.wantConvert, gotConvert)
			}
			reference, err := New(expressions, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("expected reference mapping creation to succeed, got %v", err)
			}
			reference.createRequired = newRequiredProperties(outSchema, types.Type{}, Create)
			reference.updateRequired = newRequiredProperties(outSchema, types.Type{}, Update)
			for i := range reference.expressions {
				reference.expressions[i].convert = true
				reference.expressions[i].encodeNilAsJSON = false
			}

			for _, purpose := range purposes {
				t.Run(purpose.name, func(t *testing.T) {

					wantError := slices.Contains(test.errorPurposes, purpose.purpose)
					referenceInput := map[string]any{"source": clone(test.value)}
					input := map[string]any{"source": clone(test.value)}
					refOut, refErr := reference.Transform(referenceInput, purpose.purpose)
					got, err := mapping.Transform(input, purpose.purpose)

					expectedInput := map[string]any{"source": test.value}
					if !reflect.DeepEqual(referenceInput, expectedInput) {
						t.Fatalf("expected unchanged reference input %#v, got %#v", expectedInput, referenceInput)
					}
					if !reflect.DeepEqual(input, expectedInput) {
						t.Fatalf("expected unchanged input %#v, got %#v", expectedInput, input)
					}
					if err != nil {
						if !wantError {
							t.Fatalf("expected transformation to succeed, got %v", err)
						}
						if _, ok := errors.AsType[ValidationError](err); !ok {
							t.Fatalf("expected ValidationError, got %T: %v", err, err)
						}
						if refErr == nil {
							t.Fatalf("expected reference validation error, got %#v", refOut)
						}
						if reflect.TypeOf(err) != reflect.TypeOf(refErr) || err.Error() != refErr.Error() {
							t.Fatalf("expected error %T: %v, got %T: %v", refErr, refErr, err, err)
						}
						return
					}
					if refErr != nil {
						t.Fatalf("expected error %T: %v, got %#v", refErr, refErr, got)
					}
					if wantError {
						t.Fatalf("expected validation error, got %#v", got)
					}
					if !reflect.DeepEqual(got, refOut) {
						t.Fatalf("expected output %#v, got %#v", refOut, got)
					}

				})
			}

		})
	}

}

// TestMappingSemanticsValidation verifies that mappings apply semantic rules
// to values, including those nested in containers and object properties.
func TestMappingSemanticsValidation(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)

	tests := []struct {
		name     string
		typ      types.Type
		value    string
		expected string
		valid    bool
	}{
		{"current country", country, "US", "US", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "USA", "USA", true},
		{"unknown alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ZZZ", "", false},
		{"unknown country", country, "ZZ", "", false},
		{"empty country", country, "", "", false},
		{"canonical phone", types.String().AsPhone(), "+390236618300", "+390236618300", true},
		{"formatted phone", types.String().AsPhone(), "+39 02-36618 300", "+390236618300", true},
		{"empty phone", types.String().AsPhone(), "", "", false},
		{"invalid phone", types.String().AsPhone(), "a (b)", "", false},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.typ
					var value any = test.value
					var expected any = test.expected
					data := strconv.Quote(test.value)
					switch shape {
					case "array":
						typ = types.Array(typ)
						value = []any{test.value}
						expected = []any{test.expected}
						data = "[" + data + "]"
					case "map":
						typ = types.Map(typ)
						value = map[string]any{"home": test.value}
						expected = map[string]any{"home": test.expected}
						data = "{\"home\":" + data + "}"
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						value = []any{map[string]any{"home": []any{test.value}}}
						expected = []any{map[string]any{"home": []any{test.expected}}}
						data = "[{\"home\":[" + data + "]}]"
					case "object":
						typ = types.Object([]types.Property{{Name: "inner", Type: typ}})
						value = map[string]any{"inner": test.value}
						expected = map[string]any{"inner": test.expected}
						data = "{\"inner\":" + data + "}"
					}
					outSchema := types.Object([]types.Property{{Name: "target", Type: typ, CreateRequired: true}})
					for _, mode := range []string{"convert", "equal types", "json", "constant"} {

						if mode == "constant" && shape != "string" {
							continue
						}
						if mode == "equal types" && (!test.valid || test.value != test.expected) {
							continue
						}

						t.Run(mode, func(t *testing.T) {

							inputType, inputValue := typ, value
							expr := "source"
							switch mode {
							case "convert":
								inputType = types.String()
								switch shape {
								case "array":
									inputType = types.Array(inputType)
								case "map":
									inputType = types.Map(inputType)
								case "nested":
									inputType = types.Array(types.Map(types.Array(inputType)))
								case "object":
									inputType = types.Object([]types.Property{{Name: "inner", Type: inputType}})
								}
							case "equal types":
								inputType = typ
							case "json":
								inputType, inputValue = types.JSON(), json.Value(data)
							case "constant":
								expr = strconv.Quote(test.value)
							}
							inSchema := types.Object([]types.Property{{Name: "source", Type: inputType}})
							mapping, err := New(map[string]string{"target": expr}, inSchema, outSchema, "")
							if err != nil {
								if test.valid {
									t.Fatalf("expected mapping creation to succeed, got %v", err)
								}
								if mode != "constant" {
									t.Fatalf("expected mapping creation to succeed for %s mode, got %v", mode, err)
								}
								want := " is not a valid phone number"
								if test.typ.Semantics() == types.CountrySemantics {
									want = " is not a 2-letter country code"
									if test.typ.CountryFormat() == types.ISO3166Alpha3 {
										want = " is not a 3-letter country code"
									}
								}
								want = strconv.Quote(test.value) + want
								if !strings.Contains(err.Error(), want) {
									t.Fatalf("expected error containing %q, got %v", want, err)
								}
								return
							}
							got, err := mapping.Transform(map[string]any{"source": inputValue}, Create)
							if err != nil {
								if test.valid {
									t.Fatalf("expected transformation to succeed, got %v", err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("expected ValidationError, got %T: %v", err, err)
								}
								return
							}
							if !test.valid {
								t.Fatalf("expected invalid value to be rejected, got %#v", got["target"])
							}
							if !reflect.DeepEqual(got["target"], expected) {
								t.Fatalf("expected value %#v, got %#v", expected, got["target"])
							}

						})

					}

				})

			}

		})

	}

}

// TestMappingSharedValues checks conversions of values shared across
// expressions and nested structures.
func TestMappingSharedValues(t *testing.T) {

	stringObject := types.Object([]types.Property{{Name: "x", Type: types.String()}})
	intObject := types.Object([]types.Property{{Name: "x", Type: types.Int(32)}})
	fullObject := types.Object([]types.Property{
		{Name: "keep", Type: types.String()}, {Name: "extra", Type: types.String()},
	})
	projection := types.Object([]types.Property{{Name: "keep", Type: types.String()}})
	tests := []struct {
		name           string
		source, target types.Type
		value          func() any
		expected       any
	}{
		{
			name:     "object",
			source:   stringObject,
			target:   intObject,
			value:    func() any { return map[string]any{"x": "42"} },
			expected: map[string]any{"x": 42},
		},
		{
			name:     "map",
			source:   types.Map(types.String()),
			target:   types.Map(types.Int(32)),
			value:    func() any { return map[string]any{"x": "42"} },
			expected: map[string]any{"x": 42},
		},
		{
			name:     "array",
			source:   types.Array(types.String()),
			target:   types.Array(types.Int(32)),
			value:    func() any { return []any{"42"} },
			expected: []any{42},
		},
		{
			name:     "object to map",
			source:   stringObject,
			target:   types.Map(types.Int(32)),
			value:    func() any { return map[string]any{"x": "42"} },
			expected: map[string]any{"x": 42},
		},
		{
			name:     "map to object",
			source:   types.Map(types.String()),
			target:   intObject,
			value:    func() any { return map[string]any{"x": "42"} },
			expected: map[string]any{"x": 42},
		},
		{
			name:     "object projection",
			source:   fullObject,
			target:   projection,
			value:    func() any { return map[string]any{"keep": "x", "extra": "y"} },
			expected: map[string]any{"keep": "x"},
		},
		{
			name:     "map projection",
			source:   types.Map(types.String()),
			target:   projection,
			value:    func() any { return map[string]any{"keep": "x", "extra": "y"} },
			expected: map[string]any{"keep": "x"},
		},
		{
			name:   "shared array elements",
			source: types.Array(stringObject),
			target: types.Array(intObject),
			value: func() any {
				value := map[string]any{"x": "42"}
				return []any{value, value}
			},
			expected: []any{map[string]any{"x": 42}, map[string]any{"x": 42}},
		},
		{
			name:   "shared map values",
			source: types.Map(stringObject),
			target: types.Map(intObject),
			value: func() any {
				value := map[string]any{"x": "42"}
				return map[string]any{"left": value, "right": value}
			},
			expected: map[string]any{"left": map[string]any{"x": 42}, "right": map[string]any{"x": 42}},
		},
		{
			name:   "shared nested arrays",
			source: types.Array(types.Array(types.String())),
			target: types.Array(types.Array(types.Int(32))),
			value: func() any {
				value := []any{"42"}
				return []any{value, value}
			},
			expected: []any{[]any{42}, []any{42}},
		},
		{
			name:   "shared projected objects",
			source: types.Array(fullObject),
			target: types.Array(projection),
			value: func() any {
				value := map[string]any{"keep": "x", "extra": "y"}
				return []any{value, value}
			},
			expected: []any{map[string]any{"keep": "x"}, map[string]any{"keep": "x"}},
		},
		{
			name: "shared object with distinct projections",
			source: types.Object([]types.Property{
				{Name: "left", Type: fullObject},
				{Name: "right", Type: fullObject},
			}),
			target: types.Object([]types.Property{
				{Name: "left", Type: projection},
				{Name: "right", Type: types.Object([]types.Property{{Name: "extra", Type: types.String()}})},
			}),
			value: func() any {
				shared := map[string]any{"keep": "x", "extra": "y"}
				return map[string]any{"left": shared, "right": shared}
			},
			expected: map[string]any{
				"left":  map[string]any{"keep": "x"},
				"right": map[string]any{"extra": "y"},
			},
		},
	}

	// Expressions are evaluated in path order, with "a" before "b".
	scenarios := []struct {
		name, convertedPath, originalPath string
	}{
		{"converted first", "a", "b"},
		{"original first", "b", "a"},
		{"single output", "out", ""},
	}

	for _, test := range tests {
		for _, scenario := range scenarios {
			t.Run(test.name+"/"+scenario.name, func(t *testing.T) {

				expressions := map[string]string{scenario.convertedPath: "value"}
				properties := []types.Property{{Name: scenario.convertedPath, Type: test.target}}
				expected := map[string]any{scenario.convertedPath: test.expected}
				if scenario.originalPath != "" {
					expressions[scenario.originalPath] = "value"
					properties = append(properties, types.Property{Name: scenario.originalPath, Type: test.source})
					expected[scenario.originalPath] = test.value()
				}
				inSchema := types.Object([]types.Property{{Name: "value", Type: test.source}})
				mapping, err := New(expressions, inSchema, types.Object(properties), "")
				if err != nil {
					t.Fatalf("expected mapping construction to succeed, got %v", err)
				}

				input := map[string]any{"value": test.value()}
				expectedInput := map[string]any{"value": test.value()}
				got, err := mapping.Transform(input, None)
				if err != nil {
					t.Fatalf("expected transformation to succeed, got %v", err)
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("expected matching output, got diff (-expected +got):\n%s", diff)
				}
				if diff := cmp.Diff(expectedInput, input); diff != "" {
					t.Errorf("expected unchanged input, got diff (-expected +got):\n%s", diff)
				}

			})
		}
	}

}

// TestTransformDoesNotMutateInput verifies that Transform does not modify its
// input.
func TestTransformDoesNotMutateInput(t *testing.T) {

	roundTrip := func(t *testing.T, v map[string]any, schema types.Type) map[string]any {
		t.Helper()
		data, err := types.Marshal(v, schema)
		if err != nil {
			t.Fatalf("expected input serialization to succeed, got %v", err)
		}
		copied, err := types.Decode[map[string]any](bytes.NewReader(data), schema)
		if err != nil {
			t.Fatalf("expected input decoding to succeed, got %v", err)
		}
		return copied
	}

	objectType := types.Object([]types.Property{{Name: "a", Type: types.Int(16)}, {Name: "b", Type: types.UUID()}})
	tests := []struct {
		name     string
		inType   types.Type
		outType  types.Type
		value    any
		expected any
	}{
		{
			name:     "array of integers to strings",
			inType:   types.Array(types.Int(32)),
			outType:  types.Array(types.String()),
			value:    []any{1, 2, 3},
			expected: []any{"1", "2", "3"},
		},
		{
			name:     "empty array",
			inType:   types.Array(types.Int(32)),
			outType:  types.Array(types.String()),
			value:    []any{},
			expected: []any{},
		},
		{
			name:     "map of booleans to strings",
			inType:   types.Map(types.Boolean()),
			outType:  types.Map(types.String()),
			value:    map[string]any{"a": true, "b": false},
			expected: map[string]any{"a": "true", "b": "false"},
		},
		{
			name:     "empty map",
			inType:   types.Map(types.Boolean()),
			outType:  types.Map(types.String()),
			value:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name:     "object with removed property",
			inType:   objectType,
			outType:  types.Object([]types.Property{{Name: "a", Type: types.String()}}),
			value:    map[string]any{"a": 22, "b": "90620928-691e-4aab-9b5c-ce202cad156f"},
			expected: map[string]any{"a": "22"},
		},
		{
			name:     "nullable object property",
			inType:   types.Object([]types.Property{{Name: "a", Type: types.String(), Nullable: true}}),
			outType:  types.Object([]types.Property{{Name: "a", Type: types.Int(32), Nullable: true}}),
			value:    map[string]any{"a": nil},
			expected: map[string]any{"a": nil},
		},
		{
			name:     "array of maps from integers to floats",
			inType:   types.Array(types.Map(types.Int(32))),
			outType:  types.Array(types.Map(types.Float(64))),
			value:    []any{map[string]any{"x": 12, "y": -68}, map[string]any{"a": 5, "b": 8032}},
			expected: []any{map[string]any{"x": 12.0, "y": -68.0}, map[string]any{"a": 5.0, "b": 8032.0}},
		},
		{
			name:     "map of arrays from floats to strings",
			inType:   types.Map(types.Array(types.Float(64))),
			outType:  types.Map(types.Array(types.String())),
			value:    map[string]any{"foo": []any{4.67, -1.02}},
			expected: map[string]any{"foo": []any{"4.67", "-1.02"}},
		},
		{
			name:     "unchanged array",
			inType:   types.Array(types.Int(32)),
			outType:  types.Array(types.Int(32)),
			value:    []any{1, 2, 3},
			expected: []any{1, 2, 3},
		},
		{
			name:     "unchanged map",
			inType:   types.Map(types.Boolean()),
			outType:  types.Map(types.Boolean()),
			value:    map[string]any{"a": true, "b": false},
			expected: map[string]any{"a": true, "b": false},
		},
		{
			name:     "unchanged object",
			inType:   objectType,
			outType:  objectType,
			value:    map[string]any{"a": 22, "b": "90620928-691e-4aab-9b5c-ce202cad156f"},
			expected: map[string]any{"a": 22, "b": "90620928-691e-4aab-9b5c-ce202cad156f"},
		},
	}
	expressions := map[string]string{"out": "in"}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			inSchema := types.Object([]types.Property{{Name: "in", Type: test.inType}})
			outSchema := types.Object([]types.Property{{Name: "out", Type: test.outType}})
			input := roundTrip(t, map[string]any{"in": test.value}, inSchema)
			original := roundTrip(t, input, inSchema)
			mapping, err := New(expressions, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("expected mapping construction to succeed, got %v", err)
			}

			got, err := mapping.Transform(input, None)
			if err != nil {
				t.Fatalf("expected transformation to succeed, got %v", err)
			}

			expected := map[string]any{"out": test.expected}
			if diff := cmp.Diff(expected, got); diff != "" {
				t.Errorf("expected matching output, got diff (-expected +got):\n%s", diff)
			}
			if diff := cmp.Diff(original, input); diff != "" {
				t.Errorf("expected unchanged input, got diff (-expected +got):\n%s", diff)
			}

		})
	}

}

func Test_InOutPaths(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "a", Type: types.String()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
	})

	outSchema := types.Object([]types.Property{
		{Name: "foo", Type: types.String()},
		{Name: "boo", Type: types.Int(32).Unsigned()},
	})

	tests := []struct {
		expressions map[string]string
		inPaths     []string
		outPaths    []string
	}{
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "5",
			},
			inPaths:  []string{},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "'a'",
				"boo": "b.k.y",
			},
			inPaths:  []string{"b.y"},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "b.k.x a b.k.x",
				"boo": "a b.k.x a",
			},
			inPaths:  []string{"a", "b.x"},
			outPaths: []string{"boo", "foo"},
		},
		{
			expressions: map[string]string{
				"foo": "a '-' d.p.s? ' ' b['k'].z '*' c.z c.x",
			},
			inPaths:  []string{"a", "b.z", "c.x", "c.z", "d"},
			outPaths: []string{"foo"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := mapping.InPaths()
			if got == nil || !slices.Equal(test.inPaths, got) {
				t.Fatalf("expected input properties %#v, got %#v", test.inPaths, got)
			}
			got = mapping.OutPaths()
			if got == nil || !slices.Equal(test.outPaths, got) {
				t.Fatalf("expected output properties %#v, got %#v", test.outPaths, got)
			}
		})
	}

}

func Test_Transform(t *testing.T) {

	inSchema := types.Object([]types.Property{
		{Name: "a", Type: types.String()},
		{Name: "b", Type: types.Map(types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		}))},
		{Name: "c", Type: types.Object([]types.Property{
			{Name: "x", Type: types.String()},
			{Name: "y", Type: types.Int(32).Unsigned()},
			{Name: "z", Type: types.String(), Nullable: true},
		})},
		{Name: "d", Type: types.JSON()},
		{Name: "e", Type: types.JSON(), Nullable: true},
		{Name: "f", Type: types.Object([]types.Property{
			{Name: "a", Type: types.String()},
			{Name: "d", Type: types.String()},
		}), Nullable: true},
		{Name: "emptyObject", Type: types.Object([]types.Property{{Name: "foo", Type: types.String(), ReadOptional: true}})},
	})

	outSchema := types.Object([]types.Property{
		{Name: "A", Type: types.String()},
		{Name: "B", Type: types.String(), Nullable: true},
		{Name: "C", Type: types.JSON()},
		{Name: "D", Type: types.JSON(), Nullable: true},
		{Name: "E", Type: types.Int(32)},
		{Name: "F", Type: types.String(), CreateRequired: true},
		{Name: "G", Type: types.String(), UpdateRequired: true},
		{Name: "H", Type: types.Int(64).Unsigned()},
		{Name: "I", Type: types.Float(64)},
		{Name: "J", Type: types.Decimal(5, 2)},
	})

	tests := []struct {
		name        string
		expressions map[string]string
		attributes  map[string]any
		purpose     Purpose
		expected    map[string]any
		err         error
	}{

		{
			name:        `An empty string -> an empty string`,
			expressions: map[string]string{"A": "''"},
			expected:    map[string]any{"A": ""},
		},
		{
			name:        `null assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "null"},
			expected:    map[string]any{},
		},
		{
			name:        `A property without a value -> no properties`,
			expressions: map[string]string{"A": "a"},
			expected:    map[string]any{},
		},
		{
			name:        `A nil property assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{},
		},
		{
			name:        `A property with a non-empty string -> the non-empty string`,
			expressions: map[string]string{"A": "a"},
			attributes:  map[string]any{"a": "boo"},
			expected:    map[string]any{"A": "boo"},
		},
		{
			name:        `null assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "null"},
			expected:    map[string]any{"B": nil},
		},
		{
			name:        `A nil property assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"B": nil},
		},
		{
			name:        `A property of a null object -> nil`,
			expressions: map[string]string{"B": "f.a"},
			attributes:  map[string]any{"f": nil},
			expected:    map[string]any{"B": nil},
		},
		{
			name:        `A property with a null ancestor -> nil`,
			expressions: map[string]string{"A": "a", "B": "f.a"},
			attributes:  map[string]any{"a": "outer", "f": nil},
			expected:    map[string]any{"A": "outer", "B": nil},
		},

		{
			name:        `An empty string assigned to a json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "''"},
			expected:    map[string]any{"C": json.Value(`""`)},
		},
		{
			name:        `A non-empty string assigned to a json property -> the string as JSON`,
			expressions: map[string]string{"C": "'boo'"},
			expected:    map[string]any{"C": json.Value(`"boo"`)},
		},
		{
			name:        `null assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "null"},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A property without a value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "a"},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable json property -> nil as JSON`,
			expressions: map[string]string{"C": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A property without a value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', a, 'h', 5)"},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `An empty object in array() -> an empty JSON object`,
			expressions: map[string]string{"C": "array(emptyObject)"},
			attributes:  map[string]any{"emptyObject": map[string]any{}},
			expected:    map[string]any{"C": json.Value(`[{}]`)},
		},
		{
			name:        `An empty object in map() -> an empty JSON object`,
			expressions: map[string]string{"C": "map('k', emptyObject)"},
			attributes:  map[string]any{"emptyObject": map[string]any{}},
			expected:    map[string]any{"C": json.Value(`{"k":{}}`)},
		},
		{
			name:        `A property with a nil value assigned to a map(json) key -> no properties`,
			expressions: map[string]string{"C": "map('k', c.z, 'h', 5)"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"C": json.Value(`{"h":5}`)},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			attributes:  map[string]any{"a": ""},
			expected:    map[string]any{"C": json.Value(`""`)},
		},

		{
			name:        `An empty string assigned to a nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"D": "''"},
			expected:    map[string]any{"D": json.Value(`""`)},
		},
		{
			name:        `A non-empty string assigned to a nullable json property -> the string as JSON`,
			expressions: map[string]string{"D": "'boo'"},
			expected:    map[string]any{"D": json.Value(`"boo"`)},
		},
		{
			name:        `null assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "null"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property without a value assigned to a non-nullable json property -> nil`,
			expressions: map[string]string{"D": "a"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with a nil value assigned to a non-nullable json property -> nil`,
			expressions: map[string]string{"D": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A property with an empty string assigned to a non-nullable json property -> the empty string as JSON`,
			expressions: map[string]string{"C": "a"},
			attributes:  map[string]any{"a": ""},
			expected:    map[string]any{"C": json.Value(`""`)},
		},

		{
			name:        `A json property without a value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "d"},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A json property with a nil value assigned to a non-nullable json property -> JSON null`,
			expressions: map[string]string{"C": "e"},
			attributes:  map[string]any{"e": nil},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A json property with a JSON null value assigned to a non-nullable json property -> no properties`,
			expressions: map[string]string{"C": "d"},
			attributes:  map[string]any{"d": json.Value(`null`)},
			expected:    map[string]any{"C": json.Value(`null`)},
		},
		{
			name:        `A json property without a value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "e"},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A json property with a nil value assigned to a nullable json property -> nil`,
			expressions: map[string]string{"D": "e"},
			attributes:  map[string]any{"e": nil},
			expected:    map[string]any{"D": nil},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable json property -> no properties`,
			expressions: map[string]string{"D": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"D": json.Value(`null`)},
		},

		{
			name:        `A json property with a JSON null value assigned to a non-nullable property -> no properties`,
			expressions: map[string]string{"A": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{},
		},
		{
			name:        `A json property with a JSON null value assigned to a nullable property -> nil`,
			expressions: map[string]string{"B": "e"},
			attributes:  map[string]any{"e": json.Value(`null`)},
			expected:    map[string]any{"B": nil},
		},

		{
			name:        `"a ' ' c.x ': ' 5.45"`,
			expressions: map[string]string{"A": "a ' ' c.x ': ' 5.45"},
			attributes:  map[string]any{"a": "foo", "c": map[string]any{"x": "boo"}},
			expected:    map[string]any{"A": "foo boo: 5.45"},
		},
		{
			name:        `"len(a)"`,
			expressions: map[string]string{"E": "len(a)"},
			attributes:  map[string]any{"a": "foo"},
			expected:    map[string]any{"E": 3},
		},
		{
			name:        `lower(f.d) with a null ancestor -> nil`,
			expressions: map[string]string{"B": "lower(f.d)", "C": "d"},
			attributes:  map[string]any{"d": json.Value(`123`), "f": nil},
			expected:    map[string]any{"B": nil, "C": json.Value(`123`)},
		},
		{
			name:        `Spurious properties`,
			expressions: map[string]string{"A": "a"},
			attributes:  map[string]any{"a": "foo", "b": "boo", "c": 24},
			expected:    map[string]any{"A": "foo"},
		},

		{
			name:        `null assigned to a non nullable create required property -> error`,
			expressions: map[string]string{"F": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			purpose:     Create,
			err:         ValidationError{msg: `property «F» is missing but it is required for creation`},
		},
		{
			name:        `null assigned to a non nullable update required property -> error`,
			expressions: map[string]string{"G": "c.z"},
			attributes:  map[string]any{"c.z": nil},
			purpose:     Update,
			err:         ValidationError{msg: `property «G» is missing but it is required for update`},
		},
		{
			name:        `A string exceeding the int64 range assigned to an int property -> error`,
			expressions: map[string]string{"E": "a"},
			attributes:  map[string]any{"a": "9223372036854775808"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «int(32)» type while mapping to «E»`},
		},
		{
			name:        `A JSON number exceeding the int64 range assigned to an int property -> error`,
			expressions: map[string]string{"E": "d"},
			attributes:  map[string]any{"d": json.Value(`9223372036854775808`)},
			err:         ValidationError{msg: `number «d» is too small or too large for the «int(32)» type while mapping to «E»`},
		},
		{
			name:        `A string exceeding the uint64 range assigned to an unsigned int property -> error`,
			expressions: map[string]string{"H": "a"},
			attributes:  map[string]any{"a": "18446744073709551616"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «unsigned int(64)» type while mapping to «H»`},
		},
		{
			name:        `A string exceeding the float64 range assigned to a float property -> error`,
			expressions: map[string]string{"I": "a"},
			attributes:  map[string]any{"a": "1e400"},
			err:         ValidationError{msg: `number «a» is too small or too large for the «float(64)» type while mapping to «I»`},
		},
		{
			name:        `A JSON number exceeding the precision assigned to a decimal property -> error`,
			expressions: map[string]string{"J": "d"},
			attributes:  map[string]any{"d": json.Value(`1000`)},
			err:         ValidationError{msg: `number «d» is too small or too large for the «decimal(5,2)» type while mapping to «J»`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping, err := New(test.expressions, inSchema, outSchema, "")
			if err != nil {
				t.Fatalf("unexpected error calling New: %q (%T)", err, err)
			}
			got, err := mapping.Transform(test.attributes, test.purpose)
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error: %q (%T)", err, err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("expected error %q (%T), got %q (%T)", test.err, test.err, err, err)
				}
				return
			}
			if !cmp.Equal(test.expected, got) {
				t.Fatalf("unexpected result from Transform:\n\n%s\n", cmp.Diff(test.expected, got))
			}
		})
	}

}

func Test_sortMappingExpressions(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		wantErr bool
	}{
		{
			name:    "no conflict",
			paths:   []string{"a", "b.c", "b.d", "c.d.e"},
			wantErr: false,
		},
		{
			name:    "duplicate",
			paths:   []string{"a", "b", "b", "c"},
			wantErr: true,
		},
		{
			name:    "logical prefix",
			paths:   []string{"foo", "foo.bar"},
			wantErr: true,
		},
		{
			name:    "logical prefix reversed order",
			paths:   []string{"foo.bar", "foo"},
			wantErr: true,
		},
		{
			name:    "no conflict with similar prefix",
			paths:   []string{"value", "value_currency"},
			wantErr: false,
		},
		{
			name:    "similar but not prefix",
			paths:   []string{"foo.bar", "foo.barb"},
			wantErr: false,
		},
		{
			name:    "single element",
			paths:   []string{"only"},
			wantErr: false,
		},
		{
			name:    "multiple prefixes",
			paths:   []string{"a", "a.b", "a.b.c", "b", "b.c"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exprs []mappingExpr
			for _, p := range tt.paths {
				exprs = append(exprs, mappingExpr{path: p})
			}
			err := sortMappingExpressions(exprs)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("unexpected error %q", err)
				}
			} else if tt.wantErr {
				t.Error("expected error, got no error")
			}
		})
	}
}

func Test_storeValue(t *testing.T) {
	tests := []struct {
		value    map[string]any
		path     string
		v        any
		expected map[string]any
	}{
		{
			value:    map[string]any{},
			path:     "email",
			v:        "test@example.com",
			expected: map[string]any{"email": "test@example.com"},
		},
		{
			value:    map[string]any{},
			path:     "user.email",
			v:        "test@example.com",
			expected: map[string]any{"user": map[string]any{"email": "test@example.com"}},
		},
		{
			value:    map[string]any{"user": map[string]any{"name": "Mike"}},
			path:     "user.email",
			v:        "test@example.com",
			expected: map[string]any{"user": map[string]any{"name": "Mike", "email": "test@example.com"}},
		},
		{
			value:    map[string]any{"user": map[string]any{"address": map[string]any{"city": "Milan"}}},
			path:     "user.address.zip",
			v:        "20122",
			expected: map[string]any{"user": map[string]any{"address": map[string]any{"city": "Milan", "zip": "20122"}}},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			storeValue(test.value, test.path, test.v)
			if !reflect.DeepEqual(test.value, test.expected) {
				t.Fatalf("expected %#v, got %#v", test.expected, test.value)
			}
		})
	}
}
