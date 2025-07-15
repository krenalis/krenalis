//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestEnvLoading(t *testing.T) {

	// Load the environment variables form 'test-env-file.env'.
	err := godotenv.Overload("testdata/test-env-file.env")
	if err != nil {
		t.Fatal(err)
	}

	// Determine the got key-values from the environment variables loaded by
	// godotenv.
	got := map[string]any{}
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok {
			t.Fatalf("unexpected: %q", env)
		}
		if strings.HasPrefix(key, "MEERGO_ENV_TEST_") {
			got[key] = value
		}
	}

	// Test the environment variables.
	expected := map[string]any{
		"MEERGO_ENV_TEST_A": "10",
		"MEERGO_ENV_TEST_B": "321",
		"MEERGO_ENV_TEST_C": "  hello  my   friend",
		"MEERGO_ENV_TEST_D": `"my-quoted-value"`,
		"MEERGO_ENV_TEST_E": "my-quoted-value",
		"MEERGO_ENV_TEST_F": "\"my-quoted-value",

		// TODO(Gianluca): this is caused by a bug in the parsing library.
		// See https://github.com/meergo/meergo/issues/1655 and
		// https://github.com/joho/godotenv/issues/226.
		// "MEERGO_ENV_TEST_G": "\"my-quoted-value\"",

		"MEERGO_ENV_TEST_H":     "3290",
		"MEERGO_ENV_TEST_I":     "hello\\ world",
		"MEERGO_ENV_TEST_EMPTY": "",
	}
	if !reflect.DeepEqual(got, expected) {
		for expectedK, expectedV := range expected {
			gotV, ok := got[expectedK]
			if !ok {
				t.Fatalf("env var %s expected but not read from env", expectedK)
			}
			if gotV != expectedV {
				t.Fatalf("invalid value for env var %s: expected %q, got %q", expectedK, expectedV, gotV)
			}
		}
		for gotK := range got {
			if _, ok := expected[gotK]; !ok {
				t.Fatalf("got var %s, but was not expected", gotK)
			}
		}
		t.Fatalf("expected and got env variables do not match")
	}
}
