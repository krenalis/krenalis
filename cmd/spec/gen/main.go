//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/meergo/meergo/cmd/spec"
	"github.com/meergo/meergo/json"
)

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("unable to determine caller")
	}
	destPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "doc", "src", "api", "spec.json")
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
