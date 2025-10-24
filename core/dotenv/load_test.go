//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package dotenv

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestLoad tests the Load function.
func TestLoad(t *testing.T) {
	t.Helper()

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read current working directory: %v", err)
	}

	tmpDir := t.TempDir()

	origEnv := snapshotEnv()
	restoreEnv := func() {
		restoreEnvironment(t, origEnv)
	}

	// Ensure the starting environment matches the snapshot.
	restoreEnv()

	defer func() {
		restoreEnv()
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	happyContent := func() []byte {
		var buf bytes.Buffer
		buf.WriteString("# Leading comment.\n")
		buf.WriteString("   # Comment with leading spaces.\n")
		buf.WriteString("\r\n")
		buf.WriteString("MEERGO_SIMPLE=abc\n")
		buf.WriteString("  MEERGO_SPACED =   value  # inline comment with # symbols # should be trimmed.\n")
		buf.WriteString("export MEERGO_EXPORT=exported\n")
		buf.WriteString(" export\t MEERGO_EXPORT_2 \t=exported_2\n")
		buf.WriteString("MEERGO_TRAIL=value  \n")
		buf.WriteString("MEERGO_HASH=value#literal\n")
		buf.WriteString("MEERGO_CR=carriage\r\n")
		buf.WriteString("MEERGO_DOUBLE=\"line1\\nline2\\t\\\"quote\\\"\\\\slash\"\n")
		buf.WriteString("MEERGO_QHASH=\"foo # bar\"\n")
		buf.WriteString("MEERGO_QCOMMENT=\"value\" # comment\n")
		buf.WriteString("MEERGO_SINGLE='don\\'t'\n")
		buf.WriteString("MEERGO_NULL=good\n")
		buf.WriteByte(0)
		buf.WriteString("bad\n")
		buf.WriteString("MEERGO_NO_EQUALS\n")
		buf.WriteString("=should_be_skipped\n")
		buf.WriteString("MEERGO_LAST=value_with_no_newline")
		return buf.Bytes()
	}()

	tests := []struct {
		name            string
		skipFile        bool
		dotEnvIsDir     bool
		noReadPerm      bool
		envContent      []byte
		preEnv          map[string]string
		expected        map[string]string
		expectedAbsent  []string
		wantErr         bool
		expectedErrPart string
		skipOnWindows   bool
	}{
		{
			name:     "missing file",
			skipFile: true,
			preEnv:   map[string]string{"MEERGO_BASELINE": "preserve"},
			expected: map[string]string{"MEERGO_BASELINE": "preserve"},
		},
		{
			name:       "happy path",
			envContent: happyContent,
			expected: map[string]string{
				"MEERGO_SIMPLE":   "abc",
				"MEERGO_SPACED":   "value",
				"MEERGO_EXPORT":   "exported",
				"MEERGO_EXPORT_2": "exported_2",
				"MEERGO_TRAIL":    "value  ",
				"MEERGO_HASH":     "value#literal",
				"MEERGO_CR":       "carriage",
				"MEERGO_DOUBLE":   "line1\nline2\t\"quote\"\\slash",
				"MEERGO_QHASH":    "foo # bar",
				"MEERGO_SINGLE":   "don't",
				"MEERGO_QCOMMENT": "value",
				"MEERGO_LAST":     "value_with_no_newline",
			},
			expectedAbsent: []string{
				"MEERGO_NO_EQUALS",
			},
		},
		{
			name:       "override existing variable",
			envContent: []byte("MEERGO_OVERRIDE=from_file\n"),
			preEnv:     map[string]string{"MEERGO_OVERRIDE": "from_env"},
			expected:   map[string]string{"MEERGO_OVERRIDE": "from_file"},
		},
		{
			name:       "utf8 bom stripped",
			envContent: append([]byte("\xEF\xBB\xBF"), []byte("MEERGO_BOM=value\n")...),
			expected:   map[string]string{"MEERGO_BOM": "value"},
		},
		{
			name:           "skip key containing NUL",
			envContent:     []byte("MEERGO_ZERO\x00KEY=value\n"),
			expectedAbsent: []string{"MEERGO_ZERO\x00KEY"},
		},
		{
			name:           "skip key with inner spaces",
			envContent:     []byte("MEERGO SPACE\tKEY=value\n"),
			expectedAbsent: []string{"MEERGO SPACE\tKEY"},
		},
		{
			name:           "skip value containing NUL",
			envContent:     []byte("MEERGO_KEY=zero\x00value\n"),
			expectedAbsent: []string{"MEERGO_KEY=zero\x00value"},
		},
		{
			name:            "file without read permission",
			envContent:      []byte("MEERGO_FORBIDDEN=value\n"),
			preEnv:          map[string]string{"MEERGO_FORBIDDEN": "from_env"},
			expected:        map[string]string{"MEERGO_FORBIDDEN": "from_env"},
			noReadPerm:      true,
			wantErr:         true,
			expectedErrPart: "cannot open",
			skipOnWindows:   true,
		},
		{
			name:            "invalid escape sequence",
			envContent:      []byte("MEERGO_BAD=\"bad\\q\"\n"),
			preEnv:          map[string]string{"MEERGO_BAD": "from_env"},
			expected:        map[string]string{"MEERGO_BAD": "from_env"},
			expectedAbsent:  nil,
			wantErr:         true,
			expectedErrPart: "invalid escape sequence",
		},
		{
			name:            "unterminated double quoted value",
			envContent:      []byte("MEERGO_OPEN=\"missing\n"),
			preEnv:          map[string]string{"MEERGO_OPEN": "from_env"},
			expected:        map[string]string{"MEERGO_OPEN": "from_env"},
			wantErr:         true,
			expectedErrPart: "unterminated quoted value",
		},
		{
			name:            "text after quoted value",
			envContent:      []byte("MEERGO_TRAIL_BAD=\"value\"not-comment\n"),
			preEnv:          map[string]string{"MEERGO_TRAIL_BAD": "from_env"},
			expected:        map[string]string{"MEERGO_TRAIL_BAD": "from_env"},
			wantErr:         true,
			expectedErrPart: "characters after the closing quote",
		},
		{
			name:            "quoted comment missing space",
			envContent:      []byte("MEERGO_QUOTED_FAIL=\"value\"#comment\n"),
			preEnv:          map[string]string{"MEERGO_QUOTED_FAIL": "from_env"},
			expected:        map[string]string{"MEERGO_QUOTED_FAIL": "from_env"},
			wantErr:         true,
			expectedErrPart: "characters after the closing quote",
		},
		{
			name:            "non regular file",
			dotEnvIsDir:     true,
			preEnv:          map[string]string{"MEERGO_DIR": "from_env"},
			expected:        map[string]string{"MEERGO_DIR": "from_env"},
			wantErr:         true,
			expectedErrPart: "file is not a regular file",
		},
		{
			name: "file exceeds max size",
			envContent: append(
				append([]byte("MEERGO_BIG="), bytes.Repeat([]byte{'x'}, maxEnvFileSize+1)...),
				'\n',
			),
			preEnv:          map[string]string{"MEERGO_BIG": "from_env"},
			expected:        map[string]string{"MEERGO_BIG": "from_env"},
			wantErr:         true,
			expectedErrPart: "file size exceeds 10 MiB limit",
		},
		{
			name: "partial success before error",
			envContent: []byte(strings.Join([]string{
				"MEERGO_OK=good",
				"MEERGO_FAIL=\"bad\\q\"",
				"",
			}, "\n")),
			expected: map[string]string{"MEERGO_OK": "good"},
			expectedAbsent: []string{
				"MEERGO_FAIL",
			},
			wantErr:         true,
			expectedErrPart: "invalid escape sequence",
		},
	}

	for i, test := range tests {
		if runtime.GOOS == "windows" && test.skipOnWindows {
			continue
		}

		restoreEnv()

		for _, key := range test.expectedAbsent {
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("%s: failed to unset %q before test: %v", test.name, key, err)
			}
		}

		caseDir := filepath.Join(tmpDir, fmt.Sprintf("case-%02d", i))
		if err := os.Mkdir(caseDir, 0o755); err != nil {
			t.Fatalf("%s: failed to create temporary directory: %v", test.name, err)
		}
		if err := os.Chdir(caseDir); err != nil {
			t.Fatalf("%s: failed to change directory: %v", test.name, err)
		}

		if test.dotEnvIsDir {
			if err := os.Mkdir(".env", 0o755); err != nil {
				t.Fatalf("%s: failed to create .env directory: %v", test.name, err)
			}
		} else if !test.skipFile {
			if err := os.WriteFile(".env", test.envContent, 0o600); err != nil {
				t.Fatalf("%s: failed to write .env file: %v", test.name, err)
			}
			if test.noReadPerm {
				if err := os.Chmod(".env", 0); err != nil {
					t.Fatalf("%s: failed to remove read permissions from .env: %v", test.name, err)
				}
			}
		}

		for key, value := range test.preEnv {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("%s: failed to seed environment variable %q: %v", test.name, key, err)
			}
		}

		err := Load(".env")
		if test.wantErr {
			if err == nil {
				t.Fatalf("%s: expected an error but got nil", test.name)
			}
			if test.expectedErrPart != "" && !strings.Contains(err.Error(), test.expectedErrPart) {
				t.Fatalf("%s: expected error to contain %q, got %q", test.name, test.expectedErrPart, err)
			}
		} else if err != nil {
			t.Fatalf("%s: expected no error, got %v", test.name, err)
		}

		for key, want := range test.expected {
			if got := os.Getenv(key); got != want {
				t.Errorf("%s: expected %s=%q, got %q", test.name, key, want, got)
			}
		}

		for _, key := range test.expectedAbsent {
			if value, ok := os.LookupEnv(key); ok {
				t.Errorf("%s: expected %s to be unset, got %q", test.name, key, value)
			}
		}
	}
}

// snapshotEnv captures the current environment as a key/value map.
func snapshotEnv() map[string]string {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		key, value, found := strings.Cut(kv, "=")
		if !found {
			continue
		}
		env[key] = value
	}
	return env
}

// restoreEnvironment clears the process environment and restores the provided
// snapshot.
func restoreEnvironment(t *testing.T, env map[string]string) {
	t.Helper()
	os.Clearenv()
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("failed to restore environment variable %q: %v", key, err)
		}
	}
}
