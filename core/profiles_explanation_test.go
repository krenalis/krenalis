// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestExplainProfileFilterValidation(t *testing.T) {
	workspace := &Workspace{core: &Core{}, workspace: &state.Workspace{ProfileSchema: types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
	})}}
	for _, test := range []struct {
		name, kpid, at string
		filter         string
		code           errors.Code
	}{
		{"bad KPID", "bad", "", `{"operator":"and","rules":[]}`, ""},
		{"missing filter", "00000000-0000-0000-0000-000000000001", "", `null`, ""},
		{"bad at", "00000000-0000-0000-0000-000000000001", "bad", `{"operator":"and","rules":[{"property":"email","operator":"exists"}]}`, ""},
		{"empty group", "00000000-0000-0000-0000-000000000001", "", `{"operator":"and","rules":[]}`, ""},
		{"missing property", "00000000-0000-0000-0000-000000000001", "", `{"operator":"and","rules":[{"property":"missing","operator":"exists"}]}`, PropertyNotExist},
		{"unsupported", "00000000-0000-0000-0000-000000000001", "", strings.Replace(`{"operator":"and","rules":[`+countConditionJSON+`]}`, `"values":[0]`, `"values":[18446744073709551616]`, 1), FilterCapabilityNotImplemented},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter *Filter
			err := json.Unmarshal([]byte(test.filter), &filter)
			if err != nil {
				t.Fatalf("expected decodable filter fixture, got %v", err)
			}
			result, err := workspace.ExplainProfileFilter(t.Context(), test.kpid, filter, test.at)
			if err != nil {
				if result != nil {
					t.Fatalf("expected no partial explanation, got %+v", result)
				}
				if test.code != "" {
					failure, ok := errors.AsType[*errors.UnprocessableError](err)
					if !ok || failure.Code != test.code {
						t.Fatalf("expected Unprocessable %s, got %v", test.code, err)
					}
					return
				}
				if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
					t.Fatalf("expected BadRequest, got %v", err)
				}
				return
			}
			t.Fatal("expected request validation failure, got success")
		})
	}
	for _, at := range []string{time.Now().Add(time.Hour).Format(time.RFC3339),
		"2026-01-01T00:00:00.000000001Z", "2026-01-01T00:00:00.0000000001Z"} {
		root := &Filter{Operator: OpAnd, Rules: []FilterRule{&FilterCondition{Property: "email", Operator: OpExists}}}
		_, err := workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000001", root, at)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid evaluation instant, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid evaluation instant, got success")
	}
}
