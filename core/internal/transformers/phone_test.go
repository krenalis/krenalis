// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

// TestUnmarshalPhone checks canonical output, nested string leaves and post-normalization uniqueness.
func TestUnmarshalPhone(t *testing.T) {

	phone := types.String().AsPhone()
	tests := []struct {
		typ  types.Type
		data string
		want any
	}{
		{types.Array(types.Map(phone)), `[{"home":"+39 02-36618 300"}]`, []any{map[string]any{"home": "+390236618300"}}},
		{
			types.Array(phone).WithMaxElements(2), `["+39 02-36618 300","+12001230101"]`,
			[]any{"+390236618300", "+12001230101"},
		},
		{types.Array(phone).WithUnique(), `["+390236618300","+39 02-36618 300"]`, nil},
		{types.Array(phone), `["+390236618300","+39 02-36618 300"]`, []any{"+390236618300", "+390236618300"}},
	}
	for _, test := range tests {
		schema := types.Object([]types.Property{{Name: "phone", Type: test.typ}})
		for _, language := range []state.Language{state.JavaScript, state.Python} {
			records := make([]Record, 1)
			input := strings.NewReader(`{"records":[{"value":{"phone":` + test.data + `}}]}`)
			err := Unmarshal(input, records, schema, language, false)
			if err != nil {
				t.Fatalf("expected no error for %s and %s, got %v", language, test.data, err)
			}
			if err := records[0].Err; err != nil {
				if test.want != nil {
					t.Fatalf("expected no record error, got %v", err)
				}
				continue
			}
			if test.want == nil || !reflect.DeepEqual(records[0].Attributes["phone"], test.want) {
				t.Fatalf("expected phone %#v, got %#v", test.want, records[0].Attributes["phone"])
			}
		}
	}

}
