//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"testing"
)

func Test_Buffer(t *testing.T) {

	s := struct {
		S string `json:"s"`
	}{
		S: "text",
	}

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
	expected := `["a",{"boo":true},true,"{\"s\":\"text\"}"]`
	if got := buf.String(); expected != got {
		t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
	}

	var buf2 Buffer
	err = buf2.Encode([]int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	expected = `[1,2,3]`
	if got := buf2.String(); expected != got {
		t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, got)
	}

}
