//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package local

import "testing"

func Test_versionFromFilename(t *testing.T) {
	const (
		js = "js"
		py = "py"
	)
	tests := []struct {
		ext      string
		name     string
		filename string
		version  int
		ok       bool
	}{
		{py, "", "", 0, false},
		{py, "12345", "", 0, false},
		{py, "", ".v10.py", 10, true},
		{js, "", ".v10.js", 10, true},
		{js, "", ".v10.py", 0, false},
		{py, "", ".v10.js", 0, false},
		{py, "12345", ".py", 0, false},
		{py, "789", "12345.v10.py", 0, false},
		{py, "12345", "12345.v10.py", 10, true},
		{py, "12345", "12345_v10.py", 0, false},
		{py, "12345", "12345_z10.py", 0, false},
		{py, "action", "action-12345.v1.py", 0, false},
		{py, "action", "action-12345.vA.py", 0, false},
		{py, "action", "action-12345.v1.txt", 0, false},
		{py, "action", "action-12345.v10.py", 0, false},
		{py, "action", "action-12345.v1042.py", 0, false},
		{py, "action-12345", "action-12345.v1.py", 1, true},
		{py, "action-12345", "action-12345.vA.py", 0, false},
		{py, "action-12345", "action-12345.v1.txt", 0, false},
		{py, "action-12345", "action-12345.v10.py", 10, true},
		{py, "action-12345", "action-12345.v1042.js", 0, false},
		{py, "action-12345", "action-12345.v1042.py", 1042, true},
		{js, "action-12345", "action-12345.v1042.js", 1042, true},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotV, gotOk := versionFromFilename(test.filename, test.name, test.ext)
			if test.ok != gotOk {
				t.Fatalf("filenameToVersion(%q, %q, %q): expected ok = %t, got %t", test.name, test.filename, test.ext, test.ok, gotOk)
			}
			if test.version != gotV {
				t.Fatalf("filenameToVersion(%q, %q, %q): expected version = %d, got %d", test.name, test.filename, test.ext, test.version, gotV)
			}
		})
	}
}
