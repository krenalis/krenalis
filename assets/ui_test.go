//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// This regular expression is used to fix the files generated from esbuild.
var fixReg = regexp.MustCompile(`(React\d+ = __toESM\(require_react\(\)), 1\)`)

func Test_Build(t *testing.T) {

	t.Skip() // TODO(Gianluca): re-enable.

	resolve, err := readResolveFile()
	if err != nil {
		t.Fatalf("cannot read the resolve file: %s", err)
	}

	nodeOutDir := t.TempDir()
	err = build(nodeOutDir, ".", nil)
	if err != nil {
		t.Fatalf("cannot build from node_modules: %s", err)
	}

	vendorOutDir := t.TempDir()
	err = build(vendorOutDir, ".", resolve)
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

		vendorContent = bytes.ReplaceAll(vendorContent, []byte(`// ../vendor/`), []byte(`// ../node_modules/`))
		vendorContent = bytes.ReplaceAll(vendorContent, []byte(`"../vendor/`), []byte(`"../node_modules/`))
		vendorContent = fixReg.ReplaceAll(vendorContent, []byte(`$1)`))
		nodeContent = fixReg.ReplaceAll(nodeContent, []byte(`$1)`))

		if bytes.Equal(nodeContent, vendorContent) {
			continue
		}

		tmpDir, err := os.MkdirTemp("", "chichi-test-build")
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

		diff := cmp.Diff(nodeContent, vendorContent)
		err = os.WriteFile(filepath.Join(tmpDir, "diff"), []byte(diff), 0666)
		if err != nil {
			t.Fatalf("cannot write file 'diff': %s", err)
		}

		fmt.Printf("The files %q built from 'node_modules' and 'vendor' differ."+
			" Please refer to the files located in the directory %q for more information.", name, tmpDir)

	}

}
