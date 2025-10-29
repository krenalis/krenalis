// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/meergo/meergo/cmd/spec"
	"github.com/meergo/meergo/core/json"
)

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("unable to determine caller")
	}
	destPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "docs", "src", "api", "spec.json")
	destPath = filepath.Clean(destPath)

	var b json.Buffer
	err := b.EncodeIndent(spec.Specification, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(destPath, b.Bytes(), 0666)
	if err != nil {
		log.Fatal(err)
	}
}
