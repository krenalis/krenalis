// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	"testing"
)

func Test_Buffer(t *testing.T) {

	s := struct {
		S string `json:"s"`
	}{
		S: "text",
	}

	t.Run("Encode", func(t *testing.T) {
		expected := `["a",{"boo":true},true,"{\"s\":\"text\"}"]`
		buf := NewBuffer()
		buf.WriteString("[")
		err := buf.Encode("a")
		if err != nil {
			t.Fatal(err)
		}
		buf.WriteByte(',')
		err = buf.Encode(map[string]bool{"boo": true})
		if err != nil {
			t.Fatal(err)
		}
		buf.WriteByte(',')
		buf.Write(Value("true"))
		buf.WriteByte(',')
		err = buf.EncodeQuoted(s)
		if err != nil {
			t.Fatal(err)
		}
		buf.WriteString("]")
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})

	t.Run("Encode (2)", func(t *testing.T) {
		expected := `[1,2,3]`
		var buf Buffer
		err := buf.Encode([]int{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})

	t.Run("EncodeIndent", func(t *testing.T) {
		var buf Buffer
		err := buf.EncodeIndent(map[string]any{"a": 45, "f": false, "b": 2, "d": "foo", "e": []int{5, 9, 2}, "c": true}, "\t", " ")
		if err != nil {
			t.Fatal(err)
		}
		expected := "{\n\t \"a\": 45,\n\t \"b\": 2,\n\t \"c\": true,\n\t \"d\": \"foo\",\n\t \"e\": [\n\t  5,\n\t  9,\n\t  2\n\t ],\n\t \"f\": false\n\t}"
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})

	t.Run("EncodeKeyValue", func(t *testing.T) {
		name := "Bob"
		age := 36
		var buf Buffer
		buf.WriteByte('[')
		for i := range 2 {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('{')
			err := buf.EncodeKeyValue("name", name)
			if err != nil {
				t.Fatal(err)
			}
			err = buf.EncodeKeyValue("age", age)
			if err != nil {
				t.Fatal(err)
			}
			buf.WriteByte('}')
		}
		buf.WriteByte(']')
		expected := `[{"name":"Bob","age":36},{"name":"Bob","age":36}]`
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})

	t.Run("EncodeSorted", func(t *testing.T) {
		var buf Buffer
		err := buf.EncodeSorted(map[string]any{"a": 45, "f": false, "b": 2, "d": "foo", "e": []int{5, 9, 2}, "c": true})
		if err != nil {
			t.Fatal(err)
		}
		expected := `{"a":45,"b":2,"c":true,"d":"foo","e":[5,9,2],"f":false}`
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})

	t.Run("Value", func(t *testing.T) {

		expected := Value(`{"a":5,"b":{"x":true}}`)
		var buf Buffer
		buf.WriteString(`{"a":5,"b":`)
		err := buf.Encode(map[string]any{"x": true})
		if err != nil {
			t.Fatal(err)
		}
		buf.WriteString(`}`)
		got, err := buf.Value()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, got) {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}

		buf.Truncate(0)

		buf.WriteString(`[1,2,3]4`)
		_, err = buf.Value()
		if err == nil {
			t.Fatal("expected error, got no error")
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Fatalf("expected *SyntaxError, got %T", err)
		}

	})

	t.Run("Reset", func(t *testing.T) {

		var buf Buffer
		if b := buf.Bytes(); b != nil {
			t.Fatalf("expected nil, got %#v", b)
		}
		buf.Reset(nil)
		if b := buf.Bytes(); b != nil {
			t.Fatalf("expected nil, got %#v", b)
		}
		buf.WriteString("foo")
		if s := string(buf.Bytes()); s != "foo" {
			t.Fatalf("expected \"foo\", got %q", s)
		}
		buf.Reset(nil)
		if b := buf.Bytes(); b != nil {
			t.Fatalf("expected nil, got %#v", b)
		}
		b2 := []byte("boo")
		buf.Reset(b2)
		if s := string(buf.Bytes()); s != "boo" {
			t.Fatalf("expected \"boo\", got %q", s)
		}
		b3 := []byte("foo")
		buf.Reset(b3)
		if s := string(buf.Bytes()); s != "foo" {
			t.Fatalf("expected \"foo\", got %q", s)
		}

	})

	t.Run("mixed", func(t *testing.T) {
		s := []int{1, 2}
		var buf Buffer
		if err := buf.Encode(s); err != nil {
			t.Fatal(err)
		}
		if err := buf.EncodeIndent(s, "\t", " "); err != nil {
			t.Fatal(err)
		}
		if err := buf.EncodeSorted(s); err != nil {
			t.Fatal(err)
		}
		if err := buf.EncodeIndent(s, " ", "\t"); err != nil {
			t.Fatal(err)
		}
		if err := buf.EncodeQuoted(s); err != nil {
			t.Fatal(err)
		}
		if err := buf.EncodeIndent(s, "\t", " "); err != nil {
			t.Fatal(err)
		}
		expected := "[1,2][\n\t 1,\n\t 2\n\t][1,2][\n \t1,\n \t2\n ]\"[1,2]\"[\n\t 1,\n\t 2\n\t]"
		if got := buf.String(); expected != got {
			t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
		}
	})
}
