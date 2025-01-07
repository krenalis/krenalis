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

	"github.com/meergo/meergo/cmd/spec"
	"github.com/meergo/meergo/json"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	destPath := filepath.Join(filepath.Dir(filepath.Dir(wd)), "doc", "src", "api", "spec.json")

	var b json.Buffer
	err = b.EncodeIndent(spec.Specification, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(destPath, b.Bytes(), 0666)
	if err != nil {
		log.Fatal(err)
	}
}
