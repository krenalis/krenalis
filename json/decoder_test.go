//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func Test_Decoder(t *testing.T) {

	t.Run("Decode", func(t *testing.T) {

		dec := NewDecoder(strings.NewReader(`{"name":5}`))
		var got map[string]any
		err := dec.Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		if expected := map[string]any{"name": 5.0}; !reflect.DeepEqual(expected, got) {
			t.Fatalf("unexpected value.\n\nexpected: %#v\ngot:      %#v\n\n", expected, got)
		}

	})

	t.Run("SkipOut", func(t *testing.T) {
		tests := []struct {
			data     string
			offset   int64
			expected int64
			err      error
		}{
			{`{"foo":5,"boo":8}`, 1, 17, nil},
			{`{"foo":5,"boo":8}`, 6, 17, nil},
			{`{"foo":5,"boo":8}`, 8, 17, nil},
			{`{"foo":5,"boo":8}`, 16, 17, nil},
			{`{"foo":{"boo":8}}`, 1, 17, nil},
			{`{"foo":{"boo":8}}`, 6, 17, nil},
			{`{"foo":{"boo":8}}`, 8, 16, nil},
			{`{"foo":{"boo":8}}`, 15, 16, nil},
			{`[1,2,3]`, 1, 7, nil},
			{`[1,2,3]`, 4, 7, nil},
			{`[1,{"a":[2,3]},4]`, 10, 13, nil},
			{`12.67`, 0, 0, io.EOF},
			{`12.67`, 5, 0, io.EOF},
			{`{"foo"}`, 1, 0, &SyntaxError{err: errors.New("missing character ':' after object name")}},
			{`{"foo":5`, 1, 0, io.ErrUnexpectedEOF},
		}
		for _, test := range tests {
			dec := NewDecoder(strings.NewReader(test.data))
			for {
				off := dec.dec.InputOffset()
				if off == test.offset {
					break
				}
				_ = dec.SkipToken()
			}
			err := dec.SkipOut()
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error %#v", err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("expecting error %q (type %T), got %q (type %T)", test.err, test.err, err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expecting error %q (type %T), got no error", test.err, test.err)
			}
			got := dec.dec.InputOffset()
			if test.expected != got {
				t.Fatalf("expecting offset %d, got %d", test.expected, got)
			}
		}
	})

}
