// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestPlainJSONNumber checks that scientific notation expands to exact,
// normalized decimal notation within NUMBER's precision and scale limits.
// Plain numbers and out-of-range values must yield a nil replacement.
func TestPlainJSONNumber(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"1E2", "100"},
		{"1e-0", "1"},
		{"-1e+2", "-100"},
		{"1.1E0", "1.1"},
		{"1.00000000000000001E2", "100.000000000000001"},
		{"1.234567890123456789E19", "12345678901234567890"},
		{"9.99999999999999375E13", "99999999999999.9375"},
		{"1.2300e0", "1.23"},
		{"0.0012300e2", "0.123"},
		{"1000e-3", "1"},
		{"1000e-4", "0.1"},
		{"0E999999999999999999999999999", "0"},
		{"-0.000e-999999999999999999999999999", "-0"},
		{"1e0000000000000000000000000000000000002", "100"},
		{"1e37", "1" + strings.Repeat("0", 37)},
		{"1e38", ""},
		{"1e-37", "0." + strings.Repeat("0", 36) + "1"},
		{"1e-38", ""},
		{"100e-39", "0." + strings.Repeat("0", 36) + "1"},
		{"-1e37", "-1" + strings.Repeat("0", 37)},
		{"-1e38", ""},
		{"-1e-37", "-0." + strings.Repeat("0", 36) + "1"},
		{"-1e-38", ""},
		{strings.Repeat("9", 38) + "e0", strings.Repeat("9", 38)},
		{strings.Repeat("9", 39) + "e-1", ""},
		{"0." + strings.Repeat("9", 37) + "e0", "0." + strings.Repeat("9", 37)},
		{"0." + strings.Repeat("9", 38) + "e0", ""},
		{"1." + strings.Repeat("9", 37) + "e0", "1." + strings.Repeat("9", 37)},
		{"12." + strings.Repeat("9", 37) + "e0", ""},
		{strings.Repeat("9", 38) + "0e-1", strings.Repeat("9", 38)},
		{"0." + strings.Repeat("9", 37) + "0e0", "0." + strings.Repeat("9", 37)},
		{"1" + strings.Repeat("0", 1000) + "e-1000", "1"},
		{"0." + strings.Repeat("0", 1000) + "1e1001", "1"},
		{"1e9223372036854775807", ""},
		{"1e-9223372036854775808", ""},
		{"10e9223372036854775807", ""},
		{"0.1e-9223372036854775808", ""},
		{"1e999999999999999999999999999", ""},
		{"1e-999999999999999999999999999", ""},
		{"100", ""},
		{"1.1000", ""},
	} {
		t.Run(test.input, func(t *testing.T) {
			got := plainJSONNumber(test.input)
			if test.want == "" && got != nil {
				t.Fatalf("expected nil replacement, got %q", got)
			}
			if string(got) != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

// TestPlainJSONNumberExact uses rational arithmetic to check the exact decimal
// representation of each input. Inputs outside NUMBER's limits must yield nil.
func TestPlainJSONNumberExact(t *testing.T) {
	for _, coefficient := range []string{"0", "-0", "1", "-1", "1.1", "-0.00012300", "100000",
		"1.234567890123456789", strings.Repeat("9", 38), strings.Repeat("9", 39),
		strings.Repeat("9", 38) + "0", strings.Repeat("9", 37) + ".0", strings.Repeat("9", 38) + ".0"} {
		for exponent := -45; exponent <= 45; exponent++ {
			text := fmt.Sprintf("%se%d", coefficient, exponent)
			value, ok := (&big.Rat{}).SetString(text)
			if !ok {
				t.Fatalf("expected rational fixture, got %s", text)
			}
			fixed := value.FloatString(37)
			rounded, _ := (&big.Rat{}).SetString(fixed)
			fixed = strings.TrimRight(strings.TrimRight(fixed, "0"), ".")
			whole, fraction, _ := strings.Cut(strings.TrimPrefix(fixed, "-"), ".")
			fits := rounded.Cmp(value) == 0 && len(whole)+len(fraction) <= 38
			plain := plainJSONNumber(text)
			if (plain != nil) != fits {
				t.Fatalf("expected representability %t for %s, got %q", fits, text, plain)
			}
			// A rational zero loses the sign of the input.
			if value.Sign() == 0 && strings.HasPrefix(coefficient, "-") {
				fixed = "-0"
			}
			if fits && string(plain) != fixed {
				t.Fatalf("expected %q for %s, got %q", fixed, text, plain)
			}
		}
	}
}

// TestPlainJSONNumberGenerated uses rational arithmetic to check the exact
// decimal output for generated JSON numbers and expects nil outside NUMBER's
// precision and scale limits.
func TestPlainJSONNumberGenerated(t *testing.T) {
	random := rand.New(rand.NewPCG(4721, 987))
	for trial := range 2048 {

		text := "0"
		if trial%4 != 0 {
			text = strconv.Itoa(1 + random.IntN(9))
			for range random.IntN(25) {
				text += strconv.Itoa(random.IntN(10))
			}
		}
		if digits := random.IntN(46); digits > 0 {
			text += "." + strings.Repeat("0", random.IntN(16))
			for range digits {
				text += strconv.Itoa(random.IntN(10))
			}
			text += strings.Repeat("0", random.IntN(12))
		}
		if random.IntN(2) == 0 {
			text = "-" + text
		}
		exponent := random.IntN(81)
		sign := [...]string{"", "+", "-"}[random.IntN(3)]
		text += string("eE"[random.IntN(2)]) + sign + strings.Repeat("0", random.IntN(20)) + strconv.Itoa(exponent)
		if !json.Valid([]byte(text)) {
			t.Fatalf("expected valid JSON fixture, got %s", text)
		}
		value, ok := (&big.Rat{}).SetString(text)
		if !ok {
			t.Fatalf("expected rational fixture, got %s", text)
		}

		scaled := (&big.Rat{}).Set(value)
		ten := big.NewRat(10, 1)
		scale := 0
		for !scaled.IsInt() && scale < 37 {
			scaled.Mul(scaled, ten)
			scale++
		}
		digits := len((&big.Int{}).Abs(scaled.Num()).String())
		fits := scaled.IsInt() && digits <= 38
		plain := plainJSONNumber(text)
		if !fits {
			if plain != nil {
				t.Fatalf("expected nil replacement for %q, got %q", text, plain)
			}
			continue
		}
		want := value.FloatString(scale)
		if value.Sign() == 0 && strings.HasPrefix(text, "-") {
			want = "-0"
		}
		if string(plain) != want {
			t.Fatalf("expected %q for %q, got %q", want, text, plain)
		}
		if again := plainJSONNumber(string(plain)); again != nil {
			t.Fatalf("expected no conversion for %q, got %q", plain, again)
		}

	}
}

// TestPlainJSONNumbers checks that rewriting numbers leaves the input intact
// and preserves all other bytes in the result. When no replacements are needed,
// the function must return the original slice.
func TestPlainJSONNumbers(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"  1E2 \n", "  100 \n"},
		{`{"n":1e2,"s":"1e2","1E2":-1.1e0}`, `{"n":100,"s":"1e2","1E2":-1.1}`},
		{`{"a":[{"b":1.234567890123456789e19},null,true,-0e5]}`, `{"a":[{"b":12345678901234567890},null,true,-0]}`},
		{"{ \"é\" : 1E2, \"escape\":\"\\u0065\\n\\\"\", \"n\":1e-38 }", "{ \"é\" : 100, \"escape\":\"\\u0065\\n\\\"\", \"n\":1e-38 }"},
		{`[1e38,1e-38,100.00,1e99999999999999999999]`, `[1e38,1e-38,100.00,1e99999999999999999999]`},
		{`{"text":"1e2 and someone's name","n":100}`, `{"text":"1e2 and someone's name","n":100}`},
		{`1e+2`, `100`},
		{`1e-2`, `0.01`},
		{`-1.25E-3`, `-0.00125`},
		{`[1e2,2e3,3e4]`, `[100,2000,30000]`},
		{`"1e2"`, `"1e2"`},
		{`{}`, `{}`},
		{`[]`, `[]`},
		{`null`, `null`},
	} {
		t.Run(test.input, func(t *testing.T) {

			input := json.Value(test.input)
			got, err := plainJSONNumbers(input)
			if err != nil {
				t.Fatalf("expected successful first call, got %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
			if string(input) != test.input {
				t.Fatalf("expected unchanged input %q, got %q", test.input, input)
			}
			if test.input == test.want && &input[0] != &got[0] {
				t.Fatal("expected original slice without replacements, got a copy")
			}

			again, err := plainJSONNumbers(got)
			if err != nil {
				t.Fatalf("expected successful second call, got %v", err)
			}
			if string(again) != test.want {
				t.Fatalf("expected second result %q, got %q", test.want, again)
			}
			if string(got) != test.want {
				t.Fatalf("expected unchanged input %q after second call, got %q", test.want, got)
			}
			if &got[0] != &again[0] {
				t.Fatal("expected original slice on second call, got a copy")
			}

		})
	}
}

// TestSerializeJSONNumbers checks how the CSV writer expands JSON numbers while
// preserving the usual formatting of strings and native numbers.
func TestSerializeJSONNumbers(t *testing.T) {
	for _, test := range []struct {
		name  string
		typ   types.Type
		value any
		want  string
	}{
		{"json", types.JSON(), json.Value(`{"n":1e2,"s":"someone's"}`), `'{"n":100,"s":"someone''s"}'`},
		{"json numeric string", types.JSON(), json.Value(`{"s":"1e2","n":1e2}`), `'{"s":"1e2","n":100}'`},
		{"json large integers", types.JSON(), json.Value(`[12345678901234567890,1.234567890123456789e19]`), `'[12345678901234567890,12345678901234567890]'`},
		{"decimal array", types.Array(types.Decimal(38, 0)), []any{decimal.MustParse("12345678901234567890")}, `'[12345678901234567890]'`},
		{"float array", types.Array(types.Float(64)), []any{1234567.1}, `'[1234567.1]'`},
		{"json array", types.Array(types.JSON()), []any{json.Value(`{"n":1e2}`)}, `'[{"n":100}]'`},
		{"json map", types.Map(types.JSON()), map[string]any{"n": json.Value("1.1e0")}, `'{"n":1.1}'`},
		{"decimal map", types.Map(types.Decimal(38, 0)), map[string]any{"n": decimal.MustParse("12345678901234567890")}, `'{"n":12345678901234567890}'`},
		{"float outside fixed domain", types.Array(types.Float(64)), []any{1e100}, `'[1e+100]'`},
		{"native float", types.Float(64), 1234567.1, "1234567.1"},
		{"native decimal", types.Decimal(38, 0), decimal.MustParse("12345678901234567890"), "12345678901234567890"},
		{"string", types.String(), "1e2", "1e2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var b bytes.Buffer
			err := serializeValueToCSV(&b, test.typ, test.value)
			if err != nil {
				t.Fatalf("expected CSV value, got %v", err)
			}
			if got := b.String(); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
	t.Run("invalid JSON", func(t *testing.T) {
		var b bytes.Buffer
		err := serializeValueToCSV(&b, types.JSON(), json.Value(`{"n":1e2,`))
		if err == nil {
			t.Fatal("expected invalid JSON error, got nil")
		}
		if b.Len() != 0 {
			t.Fatalf("expected no CSV bytes on error, got %q", b.String())
		}
	})
}
