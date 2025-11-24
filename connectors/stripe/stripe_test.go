// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package stripe

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/meergo/meergo/connectors"
)

// TestEncodeProperties exercises common and edge cases for encodeProperties.
func TestEncodeProperties(t *testing.T) {
	encode := func(p map[string]any) []string {
		buf := connectors.GetBodyBuffer(connectors.NoEncoding, 1024)
		defer buf.Close()

		encodeAttributes(buf, p)
		if buf.Len() == 0 {
			return nil
		}
		req, err := buf.NewRequest(context.Background(), "GET", "https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		s, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(string(s), "&")
		sort.Strings(parts)
		return parts
	}

	// Empty input produces empty output.
	t.Run("empty", func(t *testing.T) {
		got := encode(map[string]any{})
		if len(got) != 0 {
			t.Fatalf("expected empty output, got %q", strings.Join(got, "&"))
		}
	})

	// Flat primitives with URL query form encoding.
	t.Run("primitives", func(t *testing.T) {
		props := map[string]any{
			"i":  42,
			"s":  "hello world",
			"s2": "a+b c",
			"n":  nil,
		}
		got := encode(props)
		want := []string{
			"i=42",
			"s=hello+world",
			"s2=a%2Bb+c",
			"n=",
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Unsupported types trigger a panic during encoding.
	t.Run("unsupported_type_panics", func(t *testing.T) {
		buf := connectors.GetBodyBuffer(connectors.NoEncoding, 0)
		defer buf.Close()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("encodeProperties did not panic for unsupported type")
			}
		}()

		encodeAttributes(buf, map[string]any{"flag": true})
	})

	// Nested object encoding with bracketed keys.
	t.Run("nested_object", func(t *testing.T) {
		props := map[string]any{
			"customer": map[string]any{
				"address": map[string]any{
					"city": "Rome",
				},
			},
		}
		got := encode(props)
		want := []string{"customer[address][city]=Rome"}
		if len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("expected %q, got %q", want[0], strings.Join(got, "&"))
		}
	})

	// Array of scalars encoded with [] repeated.
	t.Run("array_scalars", func(t *testing.T) {
		props := map[string]any{
			"items": []any{"a", "b"},
		}
		got := encode(props)
		want := []string{"items[]=a", "items[]=b"}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Array of objects indexed to keep fields grouped per element.
	t.Run("array_objects", func(t *testing.T) {
		props := map[string]any{
			"items": []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
			},
		}
		got := encode(props)
		want := []string{"items[0][id]=1", "items[1][id]=2"}
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Arrays of objects with multiple keys retain the same index.
	t.Run("array_objects_multiple_fields", func(t *testing.T) {
		props := map[string]any{
			"tax_id_data": []any{
				map[string]any{"type": "eu_vat", "value": "BE0123456789"},
				map[string]any{"type": "it_vat", "value": "IT12345678901"},
			},
		}
		got := encode(props)
		want := []string{
			"tax_id_data[0][type]=eu_vat",
			"tax_id_data[0][value]=BE0123456789",
			"tax_id_data[1][type]=it_vat",
			"tax_id_data[1][value]=IT12345678901",
		}
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Nested arrays inside objects using [] notation.
	t.Run("nested_array_in_object", func(t *testing.T) {
		props := map[string]any{
			"a": map[string]any{
				"b": []any{1, 2},
			},
		}
		got := encode(props)
		want := []string{"a[b][]=1", "a[b][]=2"}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Keys requiring percent-encoding, including brackets in literal keys and UTF-8.
	t.Run("key_encoding", func(t *testing.T) {
		props := map[string]any{
			"key_value": map[string]any{
				"sp ce": "x",
				"a[b]":  "y",
				"ñ":     "z",
			},
		}
		got := encode(props)
		want := []string{
			"key_value[a%5Bb%5D]=y", // literal brackets must be percent-encoded.
			"key_value[sp+ce]=x",    // space encoded as '+'.
			"key_value[%C3%B1]=z",   // UTF-8 bytes percent-encoded.
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})

	// Deep but reasonable nesting remains supported within the depth limit.
	t.Run("deep_nesting", func(t *testing.T) {
		props := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": "v",
				},
			},
		}
		got := encode(props)
		want := []string{"a[b][c]=v"}
		if len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("expected %q, got %q", want[0], strings.Join(got, "&"))
		}
	})

	// Structures deeper than the schema allow must panic.
	t.Run("too_deep_panics", func(t *testing.T) {
		buf := connectors.GetBodyBuffer(connectors.NoEncoding, 0)
		defer buf.Close()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("encodeProperties did not panic for excessive depth")
			}
		}()

		attributes := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": map[string]any{
						"d": map[string]any{
							"e": "v",
						},
					},
				},
			},
		}
		encodeAttributes(buf, attributes)
	})

	// Empty arrays emit nothing by design.
	t.Run("empty_array", func(t *testing.T) {
		attributes := map[string]any{
			"items": []any{},
			"ok":    "v",
		}
		got := encode(attributes)
		want := []string{"ok=v"}
		if len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("expected %q, got %q", want[0], strings.Join(got, "&"))
		}
	})

	// encodeProperties leaves inclusion decisions to callers.
	t.Run("no_skipping", func(t *testing.T) {
		attributes := map[string]any{
			"default_source": "tok",
			"payment_method": "pm",
			"tax_id_data":    "tax",
		}
		got := encode(attributes)
		want := []string{
			"default_source=tok",
			"payment_method=pm",
			"tax_id_data=tax",
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %d pairs, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %q, got %q", want[i], got[i])
			}
		}
	})
}

// TestEncodePropertiesAllocations ensures encodeProperties keeps allocations
// stable for representative payloads, making it easier to detect regressions.
func TestEncodePropertiesAllocations(t *testing.T) {

	const expectedAllocs = 2

	buildRichPayload := func() map[string]any {
		metadata := make(map[string]any, 10)
		for i := 0; i < 10; i++ {
			metadata[fmt.Sprintf("key_%02d", i)] = fmt.Sprintf("value-%d", i)
		}

		lineItems := make([]any, 0, 5)
		for i := 0; i < 5; i++ {
			lineItems = append(lineItems, map[string]any{
				"product": fmt.Sprintf("prod_%d", i),
				"price":   1000 + i,
			})
		}

		return map[string]any{
			"name":        "Sample Customer",
			"description": "Customer with rich payload",
			"metadata":    metadata,
			"shipping": map[string]any{
				"name": "Recipient",
				"address": map[string]any{
					"city":        "Rome",
					"line1":       "Via Roma 1",
					"line2":       "Scala A",
					"postal_code": "00100",
				},
			},
			"tax_id_data": []any{
				map[string]any{"type": "eu_vat", "value": "BE0123456789"},
				map[string]any{"type": "it_vat", "value": "IT12345678901"},
			},
			"preferred_locales": []any{"it-IT", "en-US"},
			"line_items":        lineItems,
		}
	}

	cases := map[string]map[string]any{
		"scalars": {
			"id":   "cus_123",
			"name": "Alice",
			"age":  42,
		},
		"array_scalars": {
			"items": []any{"a", "b", "c"},
		},
		"array_objects": {
			"tax_id_data": []any{
				map[string]any{"type": "eu_vat", "value": "BE0123456789"},
				map[string]any{"type": "it_vat", "value": "IT12345678901"},
			},
		},
		"nested_objects": {
			"customer": map[string]any{
				"address": map[string]any{
					"city":  "Rome",
					"lines": []any{"via", "Roma"},
				},
			},
		},
		"rich_payload": buildRichPayload(),
	}

	for name, attributes := range cases {
		attributes := attributes
		t.Run(name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(100, func() {
				bb := connectors.GetBodyBuffer(connectors.NoEncoding, 1024)
				encodeAttributes(bb, attributes)
				bb.Close()
			})
			if allocs != expectedAllocs {
				t.Fatalf("expected %.2f allocations, got %.2f", float64(expectedAllocs), allocs)
			}
		})
	}

}
