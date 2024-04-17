//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// There is no need to modify Chichi's source code to compile a customized
// version. Just follow these steps:
//
//  1. Copy this file (main.go) into a new empty folder
//  2. Execute 'go mod init chichi'
//  3. Execute 'go get github.com/open2b/chichi@latest'
//  4. Customize the imported connectors below
//  5. Execute 'go mod tidy'
//  6. Execute 'go install' or 'go build' to install/build a custom binary
package main

import (
	"github.com/open2b/chichi/cmd"

	// Imports Chichi's standard connectors. You can remove this import to stop
	// importing Chichi's standard connectors, or you can copy the imports from
	// "chichi/cmd/stdconnectors" here instead of this import and then choose
	// which standard connectors to import.
	_ "github.com/open2b/chichi/cmd/stdconnectors"
	//
	//
	// You can add the imports of your custom connectors here, for example:
	//
	// "myproject/chichi/connector/foo"
	// "myproject/chichi/connector/bar"
)

func main() {
	cmd.Main()
}
