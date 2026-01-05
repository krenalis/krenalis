// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func Test_Build(t *testing.T) {

	t.Skip() // TODO(Gianluca): re-enable. See https://github.com/meergo/meergo/issues/694.

	resolve, err := readResolveFile()
	if err != nil {
		t.Fatalf("cannot read the resolve file: %s", err)
	}

	nodeOutDir := t.TempDir()
	vendorDir := filepath.Join(".", "node_modules_vendor")
	entryPoint := filepath.Join(".", "src", "index.tsx")

	err = build(nodeOutDir, vendorDir, entryPoint, nil, nil)
	if err != nil {
		t.Fatalf("cannot build from node_modules: %s", err)
	}

	vendorOutDir := t.TempDir()
	err = build(vendorOutDir, vendorDir, entryPoint, nil, resolve)
	if err != nil {
		t.Fatalf("cannot build from vendor: %s", err)
	}

	for _, name := range []string{"index.js", "index.css"} {

		nodeContent, err := os.ReadFile(filepath.Join(nodeOutDir, name))
		if err != nil {
			t.Fatalf("cannot read file %q in directory %q: %s", name, nodeOutDir, err)
		}
		vendorContent, err := os.ReadFile(filepath.Join(vendorOutDir, name))
		if err != nil {
			t.Fatalf("cannot read file %q in directory %q: %s", name, vendorContent, err)
		}

		if !bytes.Equal(nodeContent, vendorContent) {

			tmpDir, err := os.MkdirTemp("", "meergo-test-build")
			if err != nil {
				t.Fatalf("cannot create temporary directory: %s", err)
			}

			err = os.WriteFile(filepath.Join(tmpDir, "node.js"), nodeContent, 0666)
			if err != nil {
				t.Fatalf("cannot write file 'node.js': %s", err)
			}
			err = os.WriteFile(filepath.Join(tmpDir, "vendor.js"), vendorContent, 0666)
			if err != nil {
				t.Fatalf("cannot write file 'vendor.js': %s", err)
			}

			t.Fatalf("The files %q built from 'node_modules' and 'vendor' differ."+
				" Please refer to the files located in the directory %q for more information.", name, tmpDir)

		}

	}

}
