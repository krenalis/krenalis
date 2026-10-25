// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package filters

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// BenchmarkOpIsInJSON measures scalar membership, excluding filter preparation.
// String fixtures share a prefix and decoded length with their candidates.
// Run with -run '^$' -bench BenchmarkOpIsInJSON -benchmem, adding
// -benchtime=100ms -count=5 -cpu=1 for a quick repeated comparison.
func BenchmarkOpIsInJSON(b *testing.B) {
	for _, test := range []struct {
		name      string
		value     json.Value
		match     json.Value
		candidate func(int) json.Value
	}{
		{"String", json.Value(`"candidate-999"`), json.Value(`"candidate-999"`), func(i int) json.Value { return json.Value(strconv.Quote(fmt.Sprintf("candidate-%03d", i))) }},
		{"StringEscaped", json.Value(`"candidate-99\u0039\\\""`), json.Value(`"candidate-999\\\u0022"`), func(i int) json.Value { return json.Value(strconv.Quote(fmt.Sprintf("candidate-%03d\\\"", i))) }},
		{"Number", json.Value(`1000`), json.Value(`1e3`), func(i int) json.Value { return json.Value(strconv.Itoa(i + 1)) }},
		{"NumberLarge", json.Value(`18446744073709551615`), json.Value(`184467440737095516150e-1`), func(i int) json.Value { return json.Value("18446744073709551" + fmt.Sprintf("%03d", i)) }},
		{"NumberPrecision76", json.Value(strings.Repeat("9", 39) + "." + strings.Repeat("1", 37)), json.Value(strings.Repeat("9", 39) + strings.Repeat("1", 37) + "e-37"), func(i int) json.Value {
			return json.Value(strings.Repeat("8", 39) + "." + strings.Repeat("1", 34) + fmt.Sprintf("%03d", i))
		}},
		{"NumberStringOperands", json.Value(`1000`), json.Value(`1e3`), func(i int) json.Value { return json.Value(strconv.Quote(strconv.Itoa(i + 1))) }},
		{"StringEscapedNumberOperands", json.Value(`"candidate-99\u0039\\\""`), json.Value(`"candidate-999\\\u0022"`), func(i int) json.Value { return json.Value(strconv.Itoa(i + 1)) }},
		{"MixedNumber", json.Value(`1000`), json.Value(`1000.0`), mixedJSONMembershipCandidate},
		{"MixedString", json.Value(`"target"`), json.Value(`"target"`), mixedJSONMembershipCandidate},
		{"MixedTrue", json.Value(`true`), json.Value(`true`), mixedJSONMembershipCandidate},
		{"MixedNull", json.Value(`null`), json.Value(`null`), func(i int) json.Value {
			value := mixedJSONMembershipCandidate(i)
			if value.IsNull() {
				return json.Value(`true`)
			}
			return value
		}},
	} {
		b.Run(test.name, func(b *testing.B) {
			for _, count := range []int{1, 8, 32, 128} {
				b.Run(strconv.Itoa(count), func(b *testing.B) {
					for _, position := range []string{"NoMatch", "First", "Last"} {
						b.Run(position, func(b *testing.B) {

							values := make([]any, count)
							for i := range values {
								values[i] = test.candidate(i)
							}
							switch position {
							case "First":
								values[0] = test.match
							case "Last":
								values[len(values)-1] = test.match
							default:
								// No candidate matches the property value.
							}

							var value any = test.value
							var got bool
							b.ReportAllocs()
							for b.Loop() {
								got = opIsIn(value, values)
							}
							if expected := position != "NoMatch"; got != expected {
								b.Fatalf("expected membership %t, got %t", expected, got)
							}

						})
					}
				})
			}
		})
	}
}

// mixedJSONMembershipCandidate reserves true for an explicit match position.
func mixedJSONMembershipCandidate(i int) json.Value {
	switch i % 5 {
	case 0:
		return json.Value(strconv.Quote(strconv.Itoa(i + 1)))
	case 1:
		return json.Value(strconv.Itoa(i + 1))
	case 2:
		return json.Value(`false`)
	case 3:
		return json.Value(`null`)
	default:
		return json.Value(`"null"`)
	}
}
