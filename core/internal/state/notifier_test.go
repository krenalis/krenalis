// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"testing"
)

func TestParsePayload(t *testing.T) {

	tests := []struct {
		notification string
		id           int64
		name         string
		payload      string
		err          bool
	}{
		{`foo{}`, 0, `foo`, `{}`, false},
		{`foo{}@5`, 5, `foo`, `{}`, false},
		{`boo{"a":{"b":5}}@6301`, 6301, `boo`, `{"a":{"b":5}}`, false},
		{``, 0, ``, ``, true},
		{`{}`, 0, ``, ``, true},
		{`boo`, 0, ``, ``, true},
		{`boo123`, 0, ``, ``, true},
		{`boo{}0`, 0, ``, ``, true},
		{`boo{}-1`, 0, ``, ``, true},
		{`boo{} 5`, 0, ``, ``, true},
	}

	for _, test := range tests {
		id, name, payload, err := parsePayload(test.notification)
		if err != nil {
			if !test.err {
				t.Fatalf("%s: cannot parse notification: %s", test.notification, err)
			}
			continue
		}
		if test.err {
			t.Fatalf("%s: expected error, got no errors", test.notification)
		}
		if id != test.id {
			t.Fatalf("%s: expected identifier %d, got %d", test.notification, test.id, id)
		}
		if name != test.name {
			t.Fatalf("%s: expected name %q, got %q", test.notification, test.name, name)
		}
		if payload != test.payload {
			t.Fatalf("%s: expected payload %q, got %q", test.notification, test.payload, payload)
		}
	}

}
