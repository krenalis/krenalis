// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

// TestInternationalC1 distinguishes C1 from float64/UTF-16 canonicalizers.
func TestInternationalC1(t *testing.T) {

	for _, test := range []struct{ input, want string }{
		{`{"b":"Agli\u00e8", "a":0}`, "{\"a\":0,\"b\":\"Agliè\"}\n"},
		{`{"z":18446744073709551615,"a":9007199254740993}`, "{\"a\":9007199254740993,\"z\":18446744073709551615}\n"},
		{`{"\ud800\udc00":1,"\ue000":2}`, "{\"\ue000\":2,\"\U00010000\":1}\n"},
		{`{"a":[2,1,0,null],"aa":[],"A":{}}`, "{\"A\":{},\"a\":[2,1,0,null],\"aa\":[]}\n"},
		{`{"s":"\/\u2028\u2029\u0000\u001f\b\t\n\f\r\"\\<>&"}`,
			"{\"s\":\"/\u2028\u2029\\u0000\\u001f\\b\\t\\n\\f\\r\\\"\\\\<>&\"}\n"},
		{`{"s":"\u00e9","t":"e\u0301"}`, "{\"s\":\"é\",\"t\":\"é\"}\n"},
	} {
		got, err := canonicalInternationalJSON([]byte(test.input))
		if err != nil {
			t.Fatalf("expected C1 bytes, got %v", err)
		}
		if string(got) != test.want {
			t.Fatalf("expected %q, got %q", test.want, got)
		}
	}

	for _, input := range []string{
		`[]`, `null`, `true`, `{"x":true}`, `{"x":false}`, `{"x":-0}`, `{"x":-1}`, `{"x":1.0}`, `{"x":1e0}`,
		`{"x":+1}`, `{"x":01}`, `{"x":18446744073709551616}`, `{"a":1,"\u0061":2}`,
		`{"x":{"a":1,"\u0061":2}}`, `{"x":"\ud800"}`, `{"x":"\udc00"}`, `{"x":1,}`, `{} {}`,
		`{} trailing`, `/*comment*/{}`, "\ufeff{}", "{\"x\":\"\xff\"}", "{\"\xff\":0}",
	} {
		_, err := canonicalInternationalJSON([]byte(input))
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid C1 input %q rejected, got success", input)
	}

}

// TestInternationalTypedJSON checks lexical grammar, required/optional fields,
// closed objects, collection limits, exact widths and null handling.
func TestInternationalTypedJSON(t *testing.T) {

	type fixture struct {
		U64      uint64   `json:"u64"`
		U32      uint32   `json:"u32"`
		Text     string   `json:"text"`
		Optional *string  `json:"optional,omitempty"`
		Vector   []uint32 `json:"vector" max:"2"`
	}

	valid := `{"u64":18446744073709551615,"u32":4294967295,"text":"😀","vector":[]}`
	var out fixture
	err := decodeInternationalJSON(t.Context(), strings.NewReader(valid), &out)
	if err != nil {
		t.Fatalf("expected exact typed decoding, got %v", err)
	}
	if out.U64 != math.MaxUint64 || out.U32 != math.MaxUint32 || out.Optional != nil || out.Vector == nil {
		t.Fatalf("expected exact limits and preserved absence/empty array, got %+v", out)
	}

	for _, input := range []string{
		strings.Replace(valid, `"u32":4294967295`, `"u32":4294967296`, 1),
		strings.Replace(valid, `"u32":4294967295`, `"u32":"1"`, 1),
		strings.Replace(valid, `"u32":4294967295`, `"u32":1e0`, 1),
		strings.Replace(valid, `"u32":4294967295`, `"u32":1.0`, 1),
		strings.Replace(valid, `"u32":4294967295,`, ``, 1),
		strings.Replace(valid, `"text":"😀"`, `"text":""`, 1),
		strings.Replace(valid, `"text":"😀"`, `"text":"\u0000"`, 1),
		strings.Replace(valid, `"text":"😀"`, `"text":"\u007f"`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":null`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":[1,null]`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":[1,2,3]`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":[],"optional":null`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":[],"extra":1`, 1),
		strings.Replace(valid, `"vector":[]`, `"vector":[],"vector":[]`, 1),
		strings.Replace(valid, `"u32":`, `"U32":`, 1),
		valid + `{}`, valid + strings.Repeat(" ", maxInternationalDocumentBytes),
	} {
		var out fixture
		err := decodeInternationalJSON(t.Context(), strings.NewReader(input), &out)
		if err != nil {
			continue
		}
		t.Fatalf("expected malformed typed JSON rejected, got success for %.150s", input)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = decodeInternationalJSON(ctx, bytes.NewBufferString(valid), &out)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
		return
	}
	t.Fatal("expected canceled read rejected, got success")

}
